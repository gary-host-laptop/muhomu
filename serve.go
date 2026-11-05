package main

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

// serveHTML writes the index page with an ETag and revalidation semantics:
// the browser always revalidates (no-cache) but a tab opened within the
// cache TTL — or whenever the page bytes are unchanged — gets a 304 instead
// of re-downloading the full HTML.
func serveHTML(w http.ResponseWriter, r *http.Request, html []byte) {
	sum := sha256.Sum256(html)
	etag := fmt.Sprintf("%x", sum[:])

	if inm := r.Header.Get("If-None-Match"); inm == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(html)
}

// cacheControl marks a handler's responses cacheable with a fixed
// Cache-Control value.
func cacheControl(value string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", value)
		next.ServeHTTP(w, r)
	})
}

// setContentType sets the Content-Type for static assets by extension and
// gives fonts/CSS/JS a long browser cache.
func setContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch filepath.Ext(r.URL.Path) {
		case ".js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
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
		case ".ttf":
			w.Header().Set("Content-Type", "font/ttf")
		case ".otf":
			w.Header().Set("Content-Type", "font/otf")
		case ".mp3":
			w.Header().Set("Content-Type", "audio/mpeg")
		}
		w.Header().Del("X-Content-Type-Options")
		p := r.URL.Path
		if strings.HasSuffix(p, ".css") || strings.HasSuffix(p, ".js") ||
			strings.HasSuffix(p, ".woff2") || strings.HasSuffix(p, ".ttf") {
			w.Header().Set("Cache-Control", "public, max-age=86400")
		}
		next.ServeHTTP(w, r)
	})
}
