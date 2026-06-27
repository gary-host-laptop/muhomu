package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	port := flag.String("port", "8080", "port to listen on")
	dataDir := flag.String("data", "./data", "path to data directory")
	staticDir := flag.String("static", "./static", "path to static files")
	flag.Parse()

	imagesDir := filepath.Join(*dataDir, "images")
	dbPath := filepath.Join(*dataDir, "mutabu.db")

	// Ensure directories exist
	os.MkdirAll(imagesDir, 0755)

	// Init DB
	if err := initDB(dbPath); err != nil {
		log.Fatal("failed to init db:", err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	// ── API ──────────────────────────────────────────────────────────────────

	mux.HandleFunc("GET /api/data", handleGetData)
	mux.HandleFunc("POST /api/data", handlePostData)

	mux.HandleFunc("GET /api/bookmarks/search", handleSearchBookmarks)

	mux.HandleFunc("POST /api/images/upload", func(w http.ResponseWriter, r *http.Request) {
		handleImageUpload(w, r, imagesDir)
	})
	mux.HandleFunc("DELETE /api/images/{filename}", func(w http.ResponseWriter, r *http.Request) {
		handleImageDelete(w, r, imagesDir)
	})

	// Serve uploaded images
	mux.Handle("GET /api/images/", http.StripPrefix("/api/images/", http.FileServer(http.Dir(imagesDir))))

	mux.HandleFunc("GET /api/weather", handleWeather)

	// ── STATIC ───────────────────────────────────────────────────────────────

	// Create a file server with proper MIME types
	fs := http.FileServer(http.Dir(*staticDir))
	
	// Serve static files under /static/ path
	mux.Handle("/static/", http.StripPrefix("/static/", setContentType(fs)))

	// Also serve root-level files (for backward compatibility)
	// This handles requests to /theme.css, /newtab.css, etc.
	mux.Handle("/", setContentType(http.FileServer(http.Dir(*staticDir))))

	log.Printf("mutabu running on http://localhost:%s", *port)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		log.Fatal(err)
	}
}

// setContentType middleware that sets proper MIME types for static files
func setContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the file extension
		ext := filepath.Ext(r.URL.Path)
		
		// Set Content-Type based on extension
		switch ext {
		case ".js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case ".json":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		case ".svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		case ".png":
			w.Header().Set("Content-Type", "image/png")
		case ".jpg", ".jpeg":
			w.Header().Set("Content-Type", "image/jpeg")
		case ".gif":
			w.Header().Set("Content-Type", "image/gif")
		case ".webp":
			w.Header().Set("Content-Type", "image/webp")
		case ".ico":
			w.Header().Set("Content-Type", "image/x-icon")
		case ".woff2":
			w.Header().Set("Content-Type", "font/woff2")
		case ".woff":
			w.Header().Set("Content-Type", "font/woff")
		case ".ttf":
			w.Header().Set("Content-Type", "font/ttf")
		case ".otf":
			w.Header().Set("Content-Type", "font/otf")
		case ".mp3":
			w.Header().Set("Content-Type", "audio/mpeg")
		case ".html", ".htm":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		
		// Remove X-Content-Type-Options: nosniff to allow MIME type sniffing
		// Or set it to allow sniffing for specific paths
		if strings.HasPrefix(r.URL.Path, "/static/") || strings.HasPrefix(r.URL.Path, "/") {
			w.Header().Del("X-Content-Type-Options")
		}
		
		// Add cache control for static assets
		if strings.Contains(r.URL.Path, ".css") || 
		   strings.Contains(r.URL.Path, ".js") || 
		   strings.Contains(r.URL.Path, ".woff2") ||
		   strings.Contains(r.URL.Path, ".woff") ||
		   strings.Contains(r.URL.Path, ".ttf") {
			w.Header().Set("Cache-Control", "public, max-age=86400")
		}
		
		next.ServeHTTP(w, r)
	})
}
