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

	"muhomu/render"
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

// ── widget image directory helpers ────────────────────────────
var allowedImageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true,
	".webp": true, ".gif": true, ".avif": true,
}

func listWidgetImages(dir string) []string {
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

func pickWidgetImage(dir string, rng *rand.Rand) string {
	files := listWidgetImages(dir)
	if len(files) == 0 {
		return ""
	}
	return "/api/widget-images/files/" + files[rng.Intn(len(files))]
}

// ── app config (set once in main, read everywhere) ────────────
var appCfg struct {
	widgetImagesDir string
	faviconDir      string
	staticDir       string
}

func main() {
	port          := flag.String("port", "8080", "port to listen on")
	dataDir       := flag.String("data", "./data", "path to data directory")
	staticDir     := flag.String("static", "./static", "path to static files")
	widgetImgDir  := flag.String("widget-images", "", "path to widget images dir (default: <data>/widget-images)")
	flag.Parse()

	imagesDir := filepath.Join(*dataDir, "images")
	dbPath     := filepath.Join(*dataDir, "mutabu.db")

	appCfg.widgetImagesDir = *widgetImgDir
	if appCfg.widgetImagesDir == "" {
		appCfg.widgetImagesDir = filepath.Join(*dataDir, "widget-images")
	}
	appCfg.faviconDir = filepath.Join(*dataDir, "images", "favicons")
	appCfg.staticDir = *staticDir

	os.MkdirAll(imagesDir, 0755)
	os.MkdirAll(appCfg.widgetImagesDir, 0755)
	os.MkdirAll(appCfg.faviconDir, 0755)

	if err := initDB(dbPath); err != nil {
		log.Fatal("failed to init db:", err)
	}
	defer db.Close()

	// migrate existing external favicon urls to local cache in background
	go migrateFavicons(appCfg.faviconDir)

	mux := http.NewServeMux()

	// ── API ──────────────────────────────────────────────────
	mux.HandleFunc("GET /api/data",              handleGetData)
	mux.HandleFunc("POST /api/data",             handlePostData)
	mux.HandleFunc("GET /api/stats",             handleStats)
	mux.HandleFunc("GET /api/bookmarks/search",  handleSearchBookmarks)
	mux.HandleFunc("GET /api/weather",           handleWeather)

	// favicon cache
	mux.Handle("GET /api/images/favicons/",
		http.StripPrefix("/api/images/favicons/",
			http.FileServer(http.Dir(appCfg.faviconDir))))

	// widget images — directory-based, no db
	mux.HandleFunc("GET /api/widget-images/next", handleWidgetImageNext)
	mux.Handle("GET /api/widget-images/files/",
		http.StripPrefix("/api/widget-images/files/",
			http.FileServer(http.Dir(appCfg.widgetImagesDir))))

	// profile / bg images — still upload-based
	mux.HandleFunc("POST /api/images/upload", func(w http.ResponseWriter, r *http.Request) {
		handleImageUpload(w, r, imagesDir)
	})
	mux.HandleFunc("DELETE /api/images/{filename}", func(w http.ResponseWriter, r *http.Request) {
		handleImageDelete(w, r, imagesDir)
	})
	mux.Handle("GET /api/images/",
		http.StripPrefix("/api/images/", http.FileServer(http.Dir(imagesDir))))

	// ── static ───────────────────────────────────────────────
	fs := http.FileServer(http.Dir(*staticDir))
	mux.Handle("/static/", http.StripPrefix("/static/", setContentType(fs)))

	// ── page ─────────────────────────────────────────────────
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

	data, err := fetchAllData()
	if err != nil {
		http.Error(w, "failed to load data", 500)
		return
	}

	settings, _ := data["settings"].(map[string]interface{})
	if settings == nil {
		settings = make(map[string]interface{})
	}
	notes, _ := data["notes"].(string)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// ── helpers ──────────────────────────────────────────────
	getStr := func(key, def string) string {
		if v, ok := settings[key].(string); ok && v != "" {
			return v
		}
		return def
	}
	getBool := func(key string, def bool) bool {
		if v, ok := settings[key].(bool); ok {
			return v
		}
		return def
	}

	// ── theme ─────────────────────────────────────────────────
	theme := getStr("nt_theme", "dark")

	// ── fonts ─────────────────────────────────────────────────
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

	fontLatinKey := getStr("nt_font_latin", "share-tech-mono")
	fontJpKey    := getStr("nt_font_jp", "dotgothic16")
	fontClockKey := getStr("nt_font_clock", "orbitron")

	fontPixel := ""
	if fl, ok := fontLatinMap[fontLatinKey]; ok {
		jp := fontJpMap[fontJpKey]
		if jp == "" {
			jp = "'DotGothic16', monospace"
		}
		fontPixel = fl + ", " + jp
	} else if fj, ok := fontJpMap[fontJpKey]; ok {
		fontPixel = "'DotGothic16', " + fj
	}
	fontDoto  := fontClockMap[fontClockKey]
	fontClass := "font-" + fontClockKey

	// ── background image ──────────────────────────────────────
	bgImages, _ := data["bg"].([]ImageRecord)
	bgImageURL  := ""
	bgBlur      := getBool("nt_bg_blur", true)
	if len(bgImages) > 0 {
		bgImageURL = bgImages[rng.Intn(len(bgImages))].URL
	}

	// ── clock ─────────────────────────────────────────────────
	clockTheme := getStr("nt_clock_theme", "theme")
	clockClass  := ""
	switch clockTheme {
	case "light":
		clockClass = "clock-force-light"
	case "dark":
		clockClass = "clock-force-dark"
	}

	// ── profile image ─────────────────────────────────────────
	profImages, _ := data["profile"].([]ImageRecord)
	profImageURL  := ""
	if len(profImages) > 0 {
		profImageURL = profImages[rng.Intn(len(profImages))].URL
	}

	// ── misc settings ─────────────────────────────────────────
	username  := getStr("nt_username", "")
	uiLang   := getStr("nt_ui_lang", "en")
	titleLang := getStr("nt_title_lang", "ja")

	// ── search engines ────────────────────────────────────────
	var engines []render.Engine
	if rawEngines, ok := settings["nt_engines"].([]interface{}); ok {
		for _, e := range rawEngines {
			if m, ok := e.(map[string]interface{}); ok {
				engines = append(engines, render.Engine{
					Name:      getString(m, "name"),
					URL:       getString(m, "url"),
					IsDefault: m["isDefault"] == true,
				})
			}
		}
	}
	if len(engines) == 0 {
		engines = []render.Engine{
			{Name: "duckduckgo", URL: "https://duckduckgo.com/?q=", IsDefault: true},
			{Name: "youtube",    URL: "https://www.youtube.com/results?search_query="},
			{Name: "github",     URL: "https://github.com/search?q="},
		}
	}

	// ── quick access ──────────────────────────────────────────
	rawQuick, _ := data["quick"].([]QuickItem)
	var quickItems []render.QuickItem
	if len(rawQuick) == 0 {
		quickItems = []render.QuickItem{
			{Label: "GitHub",  URL: "https://github.com",  Favicon: "https://icons.duckduckgo.com/ip3/github.com.ico"},
			{Label: "YouTube", URL: "https://youtube.com", Favicon: "https://icons.duckduckgo.com/ip3/youtube.com.ico"},
			{Label: "Reddit",  URL: "https://reddit.com",  Favicon: "https://icons.duckduckgo.com/ip3/reddit.com.ico"},
		}
	} else {
		for _, q := range rawQuick {
			quickItems = append(quickItems, render.QuickItem{
				Label: q.Label, URL: q.URL, Favicon: q.Favicon, Fav: q.Favicon,
			})
		}
	}

	// ── bookmarks ─────────────────────────────────────────────
	rawBookmarks, _ := data["bookmarks"].([]BookmarkFolder)
	var bookmarkFolders []render.BookmarkFolder
	if len(rawBookmarks) == 0 {
		bookmarkFolders = []render.BookmarkFolder{
			{Folder: "Development", Links: []render.BookmarkLink{
				{Label: "GitHub",         URL: "https://github.com",        Fav: "https://icons.duckduckgo.com/ip3/github.com.ico"},
				{Label: "Stack Overflow", URL: "https://stackoverflow.com", Fav: "https://icons.duckduckgo.com/ip3/stackoverflow.com.ico"},
			}},
		}
	} else {
		for _, f := range rawBookmarks {
			folder := render.BookmarkFolder{Folder: f.Folder}
			for _, l := range f.Links {
				folder.Links = append(folder.Links, render.BookmarkLink{
					Label: l.Label, URL: l.URL, Fav: l.Fav,
				})
			}
			bookmarkFolders = append(bookmarkFolders, folder)
		}
	}

	// ── recent ────────────────────────────────────────────────
	rawRecent, _ := data["recent"].([]RecentItem)
	var recentItems []render.RecentItem
	for _, r := range rawRecent {
		recentItems = append(recentItems, render.RecentItem{Name: r.Name, URL: r.URL})
	}

	// ── widget image (directory-based) ────────────────────────
	firstWidgetImage := pickWidgetImage(appCfg.widgetImagesDir, rng)

	// ── quotes ────────────────────────────────────────────────
	var quotes []render.Quote
	if rawQ, ok := settings["nt_custom_quotes"].([]interface{}); ok {
		for _, q := range rawQ {
			if m, ok := q.(map[string]interface{}); ok {
				quotes = append(quotes, render.Quote{
					Text:   getString(m, "text"),
					Author: getString(m, "author"),
				})
			}
		}
	}
	if len(quotes) == 0 {
		quotes = []render.Quote{
			{Text: "The only way to do great work is to love what you do.", Author: "Steve Jobs"},
			{Text: "In the middle of difficulty lies opportunity.",          Author: "Albert Einstein"},
		}
	}
	quote := quotes[rng.Intn(len(quotes))]

	// ── kotoba ────────────────────────────────────────────────
	// full word list is in data/words.js; go uses a small fallback
	// jlpt filter is applied here if set
	jlptLevel := getStr("nt_jlpt_level", "all")
	words := []render.Word{
		{K: "語",   R: "ご",         M: "word, language", L: "n5"},
		{K: "勉強", R: "べんきょう", M: "study",           L: "n5"},
		{K: "漢字", R: "かんじ",     M: "kanji",           L: "n5"},
		{K: "時間", R: "じかん",     M: "time",            L: "n5"},
		{K: "友達", R: "ともだち",   M: "friend",          L: "n5"},
	}
	_ = jlptLevel // js handles full jlpt filtering from words.js
	randomWord := words[rng.Intn(len(words))]

	// ── widget layout ─────────────────────────────────────────
	type widgetLayout struct {
		ID      string
		Col     string
		Order   int
		Visible bool
	}
	defaultLayout := []widgetLayout{
		{ID: "quick-access",    Col: "left",   Order: 1, Visible: true},
		{ID: "timer",           Col: "left",   Order: 2, Visible: true},
		{ID: "rain",            Col: "left",   Order: 3, Visible: true},
		{ID: "quote",           Col: "left",   Order: 4, Visible: true},
		{ID: "bookmarks",       Col: "center", Order: 1, Visible: true},
		{ID: "notes",           Col: "center", Order: 2, Visible: true},
		{ID: "recently-visited",Col: "center", Order: 3, Visible: true},
		{ID: "image",           Col: "right",  Order: 1, Visible: true},
		{ID: "status",          Col: "right",  Order: 2, Visible: true},
		{ID: "system-stats",    Col: "right",  Order: 3, Visible: true},
		{ID: "kotoba",          Col: "right",  Order: 4, Visible: true},
	}
	layout := defaultLayout
	if rawLayout, ok := settings["nt_widget_layout"].([]interface{}); ok && len(rawLayout) > 0 {
		layout = nil
		for _, item := range rawLayout {
			if m, ok := item.(map[string]interface{}); ok {
				vis := true
				if v, ok := m["visible"].(bool); ok {
					vis = v
				}
				order := 1
				if o, ok := m["order"].(float64); ok {
					order = int(o)
				}
				layout = append(layout, widgetLayout{
					ID:      getString(m, "id"),
					Col:     getString(m, "col"),
					Order:   order,
					Visible: vis,
				})
			}
		}
	}

	// ── JS seed data ──────────────────────────────────────────
	// Only what js needs for in-memory mutations; everything else is SSR'd.
	initialData := map[string]interface{}{
		"nt_bookmarks":     rawBookmarks,
		"nt_quick":         rawQuick,
		"nt_recent":        rawRecent,
		"nt_location":      settings["nt_location"],
		"nt_search_target": getStr("nt_search_target", "_blank"),
	}
	jsonInitial, _ := json.Marshal(initialData)

	// ── build sorted column slices ────────────────────────────
	type colWidget struct {
		ID      string
		Visible bool
		Order   int
	}
	var colLeft, colCenter, colRight []colWidget
	for _, w := range layout {
		cw := colWidget{ID: w.ID, Visible: w.Visible, Order: w.Order}
		switch w.Col {
		case "center":
			colCenter = append(colCenter, cw)
		case "right":
			colRight = append(colRight, cw)
		default:
			colLeft = append(colLeft, cw)
		}
	}
	sortCol := func(col []colWidget) {
		for i := 1; i < len(col); i++ {
			for j := i; j > 0 && col[j].Order < col[j-1].Order; j-- {
				col[j], col[j-1] = col[j-1], col[j]
			}
		}
	}
	sortCol(colLeft)
	sortCol(colCenter)
	sortCol(colRight)

	// ── build inline styles ───────────────────────────────────
	// criticalStyle: color vars inlined BEFORE theme.css loads.
	// Eliminates the flash of unstyled/wrong-colored content.
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
	vars := themeVars[theme]
	if vars == nil {
		vars = themeVars["dark"]
	}

	var colorParts []string
	for k, v := range vars {
		colorParts = append(colorParts, k+":"+v)
	}
	criticalStyle := template.CSS(":root{" + strings.Join(colorParts, ";") + "}")

	// fontStyle: font vars inlined AFTER theme.css so they override
	// theme.css's default --font-pixel and --font-doto values.
	var fontParts []string
	if fontPixel != "" {
		fontParts = append(fontParts, "--font-pixel:"+fontPixel)
	}
	if fontDoto != "" {
		fontParts = append(fontParts, "--font-doto:"+fontDoto)
	}
	fontStyle := template.CSS("")
	if len(fontParts) > 0 {
		// Must target body.theme-X to match theme.css's specificity
		// (:root alone loses to body.theme-dark { --font-pixel: ... })
		fontStyle = template.CSS(
			"body.theme-" + theme + "{" + strings.Join(fontParts, ";") + "}",
		)
	}

	// ── bg html ───────────────────────────────────────────────
	bgHTML := template.HTML(`<div id="bg"></div>`)
	if bgImageURL != "" {
		cls := ""
		if bgBlur {
			cls = ` class="blurred"`
		}
		bgHTML = template.HTML(`<div id="bg"` + cls + `><img src="` + bgImageURL + `" alt=""></div>`)
	}

	// ── page data ─────────────────────────────────────────────
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
		ColLeft           []colWidget
		ColCenter         []colWidget
		ColRight          []colWidget
		InitialData       template.JS
	}{
		BodyClass: strings.TrimSpace("theme-" + theme + " " + fontClass + " " + clockClass),
		BodyData: template.HTMLAttr(fmt.Sprintf(
			`data-clock-format="%s" data-clock-seconds="%v" data-ui-lang="%s"`,
			getStr("nt_clock_format", "24h"),
			getBool("nt_clock_seconds", true),
			uiLang,
		)),
		CriticalStyle:     criticalStyle,
		FontStyle:         fontStyle,
		BgHTML:            bgHTML,
		ProfileImageURL:   profImageURL,
		Username:          username,
		UILang:            uiLang,
		TitleLang:         titleLang,
		SearchEnginesHTML: render.SearchEngines(engines),
		Widgets: map[string]template.HTML{
			"quick-access":     render.WidgetQuickAccess(quickItems),
			"bookmarks":        render.WidgetBookmarks(bookmarkFolders),
			"notes":            render.WidgetNotes(notes),
			"recently-visited": render.WidgetRecent(recentItems),
			"image":            render.WidgetImage(firstWidgetImage),
			"status":           render.WidgetStatus("—", "—", "—", "—"),
			"system-stats":     render.WidgetSystemStats(),
			"kotoba":           render.WidgetKotoba(randomWord),
			"timer":            render.WidgetTimer(),
			"rain":             render.WidgetRain(),
			"quote":            render.WidgetQuote(quote),
		},
		ColLeft:     colLeft,
		ColCenter:   colCenter,
		ColRight:    colRight,
		InitialData: template.JS(jsonInitial),
	}

	tmpl, err := template.New("index.tmpl").Funcs(template.FuncMap{
		"widgetHTML": func(widgets map[string]template.HTML, id string) template.HTML {
			return widgets[id]
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

// ── setContentType middleware ──────────────────────────────────
func setContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch filepath.Ext(r.URL.Path) {
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
		w.Header().Del("X-Content-Type-Options")
		p := r.URL.Path
		if strings.HasSuffix(p, ".css") || strings.HasSuffix(p, ".js") ||
			strings.HasSuffix(p, ".woff2") || strings.HasSuffix(p, ".ttf") {
			w.Header().Set("Cache-Control", "public, max-age=86400")
		}
		next.ServeHTTP(w, r)
	})
}

// ── helper ────────────────────────────────────────────────────
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
