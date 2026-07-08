package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"muhomu/widgets"
)

// ── cache ──────────────────────────────────────────────────────
type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

var (
	pageCache = struct {
		sync.RWMutex
		data map[string]*cacheEntry
	}{data: make(map[string]*cacheEntry)}
	cacheTTL = 5 * time.Second
)

func invalidateCache() {
	pageCache.Lock()
	pageCache.data = make(map[string]*cacheEntry)
	pageCache.Unlock()
}

// ── image directory helpers ────────────────────────────────────
var allowedImageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true,
	".webp": true, ".gif": true, ".avif": true,
}

func listImagesDir(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if allowedImageExts[strings.ToLower(filepath.Ext(e.Name()))] {
			files = append(files, e.Name())
		}
	}
	return files
}

func pickImage(dir, urlPrefix string, rng *rand.Rand) string {
	files := listImagesDir(dir)
	if len(files) == 0 {
		return ""
	}
	return urlPrefix + files[rng.Intn(len(files))]
}

// ── app runtime dirs ───────────────────────────────────────────
var appCfg struct {
	faviconDir      string
	profileDir      string
	bgDir           string
	widgetImagesDir string
	staticDir       string
}

func main() {
	port       := flag.String("port", "8080", "port to listen on")
	dataDir    := flag.String("data", "./data", "path to data directory")
	staticDir  := flag.String("static", "./static", "path to static files")
	configPath := flag.String("config", "./config.yaml", "path to config file")
	flag.Parse()

	cfg = loadConfig(*configPath)

	resolve := func(cfgVal, def string) string {
		if cfgVal != "" {
			return cfgVal
		}
		return filepath.Join(*dataDir, def)
	}

	appCfg.profileDir      = resolve(cfg.ProfileImagesDir, "images/profile")
	appCfg.bgDir           = resolve(cfg.BGImagesDir,      "images/bg")
	appCfg.widgetImagesDir = resolve(cfg.WidgetImagesDir,  "widget-images")
	appCfg.faviconDir      = filepath.Join(*dataDir, "images", "favicons")
	appCfg.staticDir       = *staticDir

	for _, d := range []string{
		appCfg.profileDir, appCfg.bgDir,
		appCfg.widgetImagesDir, appCfg.faviconDir,
	} {
		os.MkdirAll(d, 0755)
	}

	dbPath := filepath.Join(*dataDir, "mutabu.db")
	if err := initDB(dbPath); err != nil {
		log.Fatal("failed to init db:", err)
	}
	defer db.Close()

	go migrateFavicons(appCfg.faviconDir)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/data",             handleGetData)
	mux.HandleFunc("POST /api/data",            handlePostData)
	mux.HandleFunc("GET /api/stats",            handleStats)
	mux.HandleFunc("GET /api/bookmarks/search", handleSearchBookmarks)
	mux.HandleFunc("GET /api/weather",          handleWeather)
	mux.HandleFunc("GET /api/quotes",           handleGetQuotes)
	mux.HandleFunc("POST /api/quotes",          handleAddQuote)
	mux.HandleFunc("DELETE /api/quotes/{id}",   handleDeleteQuote)

	mux.HandleFunc("GET /api/widget-images/next", handleWidgetImageNext)
	mux.Handle("GET /api/widget-images/files/",
		http.StripPrefix("/api/widget-images/files/",
			http.FileServer(http.Dir(appCfg.widgetImagesDir))))

	mux.Handle("GET /api/images/profile/",
		http.StripPrefix("/api/images/profile/",
			http.FileServer(http.Dir(appCfg.profileDir))))
	mux.Handle("GET /api/images/bg/",
		http.StripPrefix("/api/images/bg/",
			http.FileServer(http.Dir(appCfg.bgDir))))
	mux.Handle("GET /api/images/favicons/",
		http.StripPrefix("/api/images/favicons/",
			http.FileServer(http.Dir(appCfg.faviconDir))))

	fs := http.FileServer(http.Dir(*staticDir))
	mux.Handle("/static/", http.StripPrefix("/static/", setContentType(fs)))
	mux.HandleFunc("/", serveIndex)

	log.Printf("muhomu running on http://localhost:%s", *port)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		log.Fatal(err)
	}
}

// ── serveIndex ───────────────────────────────────────────────
func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	pageCache.RLock()
	entry, found := pageCache.data["/"]
	pageCache.RUnlock()
	if found && time.Now().Before(entry.expiresAt) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(entry.data)
		return
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Build widget html — only active widgets produce output.
	// The registry contains all possible widgets; config.yaml
	// declares which ones actually exist on this instance.
	registry := widgets.Registry()
	ctx := widgets.RenderContext{
		DB:             db,
		RNG:            rng,
		FaviconDir:     appCfg.faviconDir,
		ProfileDir:     appCfg.profileDir,
		BGDir:          appCfg.bgDir,
		WidgetImageDir: appCfg.widgetImagesDir,
		JLPTLevel:      cfg.JLPTLevel,
		TitleLang:      cfg.TitleLang,
	}

	widgetMap := make(map[string]template.HTML)
	var widgetScripts strings.Builder
	for _, col := range cfg.Columns {
		for _, wCfg := range col.Widgets {
			widget, ok := registry[wCfg.ID]
			if !ok {
				log.Printf("serveIndex: unknown widget %q in config — skipping", wCfg.ID)
				continue
			}
			html, err := widget.Render(ctx)
			if err != nil {
				log.Printf("serveIndex: widget %q render error: %v", wCfg.ID, err)
				continue
			}
			widgetMap[wCfg.ID] = html
			if s, ok := widget.(widgets.Scriptable); ok {
				widgetScripts.WriteString("\n/* " + wCfg.ID + " */\n")
				widgetScripts.WriteString(s.Script())
				widgetScripts.WriteString("\n")
			}
		}
	}

	// Columns are passed directly to the template as typed structs.
	// The template iterates columns and renders each widget in declaration order.
	type tmplWidget struct{ ID string }
	type tmplColumn struct {
		Size    string
		Widgets []tmplWidget
	}
	var tmplCols []tmplColumn
	for _, col := range cfg.Columns {
		tc := tmplColumn{Size: col.Size}
		for _, w := range col.Widgets {
			tc.Widgets = append(tc.Widgets, tmplWidget{ID: w.ID})
		}
		tmplCols = append(tmplCols, tc)
	}

	// Fonts.
	fontLatinMap := map[string]string{
		"inter":           "'Inter', sans-serif",
		"share-tech-mono": "'Share Tech Mono', monospace",
		"vt323":           "'VT323', monospace",
		"courier-prime":   "'Courier Prime', monospace",
	}
	fontJpMap := map[string]string{
		"dotgothic16":  "'DotGothic16', monospace",
		"biz-udgothic": "'BIZ UDGothic', sans-serif",
		"noto-sans-jp": "'Noto Sans JP', sans-serif",
	}
	fontClockMap := map[string]string{
		"medodica": "'Medodica', monospace",
		"orbitron": "'Orbitron', monospace",
		"oxanium":  "'Oxanium', monospace",
	}
	fontPixel := ""
	if fl, ok := fontLatinMap[cfg.FontLatin]; ok {
		jp := fontJpMap[cfg.FontJP]
		if jp == "" {
			jp = "'DotGothic16', monospace"
		}
		fontPixel = fl + ", " + jp
	} else if fj, ok := fontJpMap[cfg.FontJP]; ok {
		fontPixel = "'DotGothic16', " + fj
	}
	fontDoto  := fontClockMap[cfg.FontClock]
	fontClass := "font-" + cfg.FontClock

	clockClass := ""
	switch cfg.ClockTheme {
	case "light":
		clockClass = "clock-force-light"
	case "dark":
		clockClass = "clock-force-dark"
	}

	// Background.
	bgImageURL := pickImage(appCfg.bgDir, "/api/images/bg/", rng)
	bgHTML := template.HTML(`<div id="bg"></div>`)
	if bgImageURL != "" {
		cls := ""
		if cfg.BGBlur {
			cls = ` class="blurred"`
		}
		bgHTML = template.HTML(`<div id="bg"` + cls + `><img src="` + bgImageURL + `" alt=""></div>`)
	}

	// Profile image.
	profImageURL := pickImage(appCfg.profileDir, "/api/images/profile/", rng)

	// Search engines.
	var enginesHTML strings.Builder
	for _, e := range cfg.SearchEngines {
		active := ""
		if e.Default {
			active = " active"
		}
		fmt.Fprintf(&enginesHTML,
			`<button class="engine-btn%s" data-url="%s">%s</button>`,
			active, e.URL, e.Name)
	}
	if enginesHTML.Len() == 0 {
		enginesHTML.WriteString(`<button class="engine-btn active" data-url="https://duckduckgo.com/?q=">duckduckgo</button>`)
	}

	// Critical style — colors inlined before theme.css to prevent flash.
	themeVars := map[string]map[string]string{
		"dark": {
			"--bg":        "#060608",
			"--panel":     "#12111a",
			"--panel2":    "#16151f",
			"--border":    "#302d42",
			"--border-lt": "#4a4660",
			"--white":     "#e8e8f0",
			"--dim":       "#8a88a0",
			"--dimmer":    "#4a4660",
			"--accent":    "#8ab4f8",
			"--accent2":   "#c4a0f8",
			"--accent3":   "#90d870",
			"--red":       "#f28080",
			"--titlebar":  "#0e0d16",
		},
		"light": {
			"--bg":        "#e8e8e8",
			"--panel":     "#f2f2f2",
			"--panel2":    "#eaeaea",
			"--border":    "#c8c8cc",
			"--border-lt": "#b0b0b8",
			"--white":     "#1a1a24",
			"--dim":       "#5a5a6a",
			"--dimmer":    "#a0a0b0",
			"--accent":    "#3a6fd8",
			"--accent2":   "#7040c8",
			"--accent3":   "#3a9a3a",
			"--red":       "#c83030",
			"--titlebar":  "#dcdce0",
		},
	}
	vars := themeVars[cfg.Theme]
	if vars == nil {
		vars = themeVars["dark"]
	}
	var colorParts []string
	for k, v := range vars {
		colorParts = append(colorParts, k+":"+v)
	}
	criticalStyle := template.CSS(":root{" + strings.Join(colorParts, ";") + "}")

	var fontParts []string
	if fontPixel != "" {
		fontParts = append(fontParts, "--font-pixel:"+fontPixel)
	}
	if fontDoto != "" {
		fontParts = append(fontParts, "--font-doto:"+fontDoto)
	}
	fontStyle := template.CSS("")
	if len(fontParts) > 0 {
		fontStyle = template.CSS("body.theme-" + cfg.Theme + "{" + strings.Join(fontParts, ";") + "}")
	}

	// JS seed — only mutable user data.
	rawBookmarks, _ := getBookmarks()
	rawQuick, _     := getQuickAccess()
	rawRecent, _    := getRecent()
	initialData := map[string]interface{}{
		"nt_bookmarks":     rawBookmarks,
		"nt_quick":         rawQuick,
		"nt_recent":        rawRecent,
		"nt_location":      map[string]string{"city": cfg.Location.City, "lat": cfg.Location.Lat, "lon": cfg.Location.Lon},
		"nt_search_target": cfg.SearchTarget,
	}
	jsonInitial, _ := json.Marshal(initialData)

	pageData := struct {
		BodyClass         string
		BodyData          template.HTMLAttr
		CriticalStyle     template.CSS
		FontStyle         template.CSS
		BgHTML            template.HTML
		ProfileImageURL   string
		Username          string
		UILang            string
		TitleLang         string
		SearchEnginesHTML template.HTML
		Widgets           map[string]template.HTML
		Columns           []tmplColumn
		InitialData       template.JS
		WidgetScripts     template.JS
	}{
		BodyClass: strings.TrimSpace("theme-" + cfg.Theme + " " + fontClass + " " + clockClass),
		BodyData: template.HTMLAttr(fmt.Sprintf(
			`data-clock-format="%s" data-clock-seconds="%v" data-ui-lang="%s"`,
			cfg.ClockFormat, cfg.ClockSeconds, cfg.UILang,
		)),
		CriticalStyle:     criticalStyle,
		FontStyle:         fontStyle,
		BgHTML:            bgHTML,
		ProfileImageURL:   profImageURL,
		Username:          cfg.Username,
		UILang:            cfg.UILang,
		TitleLang:         cfg.TitleLang,
		SearchEnginesHTML: template.HTML(enginesHTML.String()),
		Widgets:           widgetMap,
		Columns:           tmplCols,
		InitialData:       template.JS(jsonInitial),
		WidgetScripts:     template.JS(widgetScripts.String()),
	}

	tmpl, err := template.New("index.tmpl").Funcs(template.FuncMap{
		"widgetHTML": func(ws map[string]template.HTML, id string) template.HTML {
			return ws[id]
		},
	}).ParseFiles("templates/index.tmpl")
	if err != nil {
		http.Error(w, "template error: "+err.Error(), 500)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, pageData); err != nil {
		http.Error(w, "render error: "+err.Error(), 500)
		return
	}
	html := buf.Bytes()

	pageCache.Lock()
	pageCache.data["/"] = &cacheEntry{data: html, expiresAt: time.Now().Add(cacheTTL)}
	pageCache.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(html)
}

// ── setContentType ─────────────────────────────────────────────
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

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
