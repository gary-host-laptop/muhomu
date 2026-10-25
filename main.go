package main

import (
	"bytes"
	"crypto/sha256"
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

	// indexTmpl is the parsed new-tab page template, created once at startup.
	indexTmpl *template.Template
)

// version is baked in at build time via -ldflags "-X main.version=…";
// shown in the footer as "muhomu v…". The Dockerfile passes a VERSION
// build arg. Defaults to the current release if built without ldflags.
var version = "1.1.1"

// githubRepoURL is where the footer's GitHub icon links.
const githubRepoURL = "https://github.com/gary-host-laptop/muhomu"

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
	widgetThumbsDir string
	themesDir       string
	staticDir       string
}

func main() {
	var err error

	port := flag.String("port", "8080", "port to listen on")
	dataDir := flag.String("data", "./data", "path to data directory")
	staticDir := flag.String("static", "./static", "path to static files")
	configPath := flag.String("config", "./data/config.yaml", "path to config file")
	flag.Parse()

	startupCfg := loadConfig(*configPath)
	setConfig(startupCfg)
	quotesPath := filepath.Join(*dataDir, "quotes.yaml")
	loadQuotes(quotesPath)

	// The settings API writes these files — keep the resolved paths
	// around at package level.
	configFilePath = *configPath
	quotesFilePath = quotesPath

	resolve := func(cfgVal, def string) string {
		if cfgVal != "" {
			return cfgVal
		}
		return filepath.Join(*dataDir, def)
	}

	// Data-directory resolution happens once at startup: these paths are
	// also overridable by flags, so they are not part of hot-reload.
	appCfg.profileDir = resolve(startupCfg.ProfileImagesDir, "images/profile")
	appCfg.bgDir = resolve(startupCfg.BGImagesDir, "images/bg")
	appCfg.widgetImagesDir = resolve(startupCfg.WidgetImagesDir, "images/widget")
	appCfg.widgetThumbsDir = filepath.Join(appCfg.widgetImagesDir, "thumbs")
	appCfg.faviconDir = filepath.Join(*dataDir, "images", "favicons")
	appCfg.themesDir = resolve(startupCfg.ThemesDir, "themes")
	appCfg.staticDir = *staticDir

	for _, d := range []string{
		appCfg.profileDir, appCfg.bgDir,
		appCfg.widgetImagesDir, appCfg.faviconDir,
		appCfg.themesDir, appCfg.widgetThumbsDir,
	} {
		os.MkdirAll(d, 0755)
	}

	dbPath := filepath.Join(*dataDir, "muhomu.db")
	if err := initDB(dbPath); err != nil {
		log.Fatal("failed to init db:", err)
	}
	defer db.Close()

	go migrateFavicons(appCfg.faviconDir)
	go ensureWidgetThumbs(appCfg.widgetImagesDir)
	startStatsCollector()
	go startRSSWarmer()

	// Hot-reload config.yaml: edits to the running config are picked up
	// within a couple of seconds, no restart needed.
	go watchConfig(*configPath)

	// ── Parse template once at startup ──
	indexTmpl, err = template.New("index.tmpl").Funcs(template.FuncMap{
		"widgetHTML": func(ws map[string]template.HTML, key string) template.HTML {
			return ws[key]
		},
	}).ParseFiles("templates/index.tmpl")
	if err != nil {
		log.Fatal("failed to parse index template:", err)
	}

	settingsTmpl, err = template.ParseFiles("templates/settings.tmpl")
	if err != nil {
		log.Fatal("failed to parse settings template:", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/footer-stats", handleFooterStats)
	mux.HandleFunc("GET /api/bookmarks/search", handleSearchBookmarks)
	mux.HandleFunc("GET /api/weather", handleWeather)
	mux.HandleFunc("GET /api/quotes", handleGetQuotes)
	mux.HandleFunc("POST /api/quotes", handleAddQuote)
	mux.HandleFunc("DELETE /api/quotes/{id}", handleDeleteQuote)
	mux.HandleFunc("GET /api/quote/random", handleQuoteRandom)
	mux.HandleFunc("GET /api/kotoba/next", handleKotobaNext)

	// Settings page + API — the GUI that edits config.yaml.
	mux.HandleFunc("GET /settings", handleSettingsPage)
	mux.HandleFunc("GET /api/settings", handleSettingsGet)
	mux.HandleFunc("POST /api/settings", handleSettingsPost)
	mux.HandleFunc("POST /api/settings/reset", handleSettingsReset)
	mux.HandleFunc("POST /api/settings/quotes", handleQuotesSave)
	mux.HandleFunc("POST /api/settings/images", handleImageUpload)
	mux.HandleFunc("DELETE /api/settings/images", handleImageDelete)

	// htmx widget fragment endpoints — render a single widget from
	// server state (GET) or persist its state and re-render it (POST).
	mux.HandleFunc("GET /api/widget/{id}", handleWidgetGet)
	mux.HandleFunc("POST /api/widget/{id}", handleWidgetPost)

	mux.HandleFunc("GET /api/widget-images/next", handleWidgetImageNext)
	// Widget image filenames are content-addressed (danbooru_<id>_<hash>),
	// so the same filename always means the same bytes — immutable caching
	// is safe and makes repeat tabs read straight from the browser cache.
	mux.Handle("GET /api/widget-images/files/",
		http.StripPrefix("/api/widget-images/files/",
			cacheControl("public, max-age=31536000, immutable",
				http.FileServer(http.Dir(appCfg.widgetImagesDir)))))
	mux.Handle("GET /api/widget-images/thumbs/",
		http.StripPrefix("/api/widget-images/thumbs/",
			cacheControl("public, max-age=86400",
				http.FileServer(http.Dir(appCfg.widgetThumbsDir)))))

	mux.Handle("GET /api/images/profile/",
		http.StripPrefix("/api/images/profile/",
			cacheControl("public, max-age=86400",
				http.FileServer(http.Dir(appCfg.profileDir)))))
	mux.Handle("GET /api/images/bg/",
		http.StripPrefix("/api/images/bg/",
			cacheControl("public, max-age=86400",
				http.FileServer(http.Dir(appCfg.bgDir)))))
	// Favicon files are content-addressed (md5-based names), so a given
	// filename always means the same bytes — a day-long cache never
	// serves stale content, and revisits (e.g. re-entering a folder)
	// load the icons instantly.
	mux.Handle("GET /api/images/favicons/",
		http.StripPrefix("/api/images/favicons/",
			cacheControl("public, max-age=86400", http.FileServer(http.Dir(appCfg.faviconDir)))))

	mux.Handle("GET /api/themes/",
		http.StripPrefix("/api/themes/",
			cacheControl("public, max-age=86400",
				http.FileServer(http.Dir(appCfg.themesDir)))))

	fs := http.FileServer(http.Dir(*staticDir))
	mux.Handle("/static/", http.StripPrefix("/static/", setContentType(fs)))
	mux.HandleFunc("/", serveIndex)

	log.Printf("muhomu running on http://localhost:%s", *port)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		log.Fatal(err)
	}
}

// widgetIDsWithScript is the set of widget IDs that have a JS file
// in static/js/. Widgets without JS (calendar, weather, and the ones
// rendered purely via htmx: image, kotoba, quote, system-stats) are
// excluded to avoid 404s.
var widgetIDsWithScript = map[string]bool{
	"bookmarks":        true,
	"notes":            true,
	"quick-access":     true,
	"rain":             true,
	"recently-visited": true,
	"timer":            true,
}

// collectActiveWidgetIDs recursively walks the widget tree and returns
// all unique widget IDs that have a corresponding JS file in static/js/.
func collectActiveWidgetIDs(configs []WidgetConfig) []string {
	seen := map[string]bool{}
	var ids []string
	var walk func([]WidgetConfig)
	walk = func(cfgs []WidgetConfig) {
		for _, w := range cfgs {
			if w.ID == "split-column" {
				if rawChildren, ok := w.Options["widgets"].([]interface{}); ok {
					var childConfigs []WidgetConfig
					for _, raw := range rawChildren {
						m, ok := raw.(map[string]interface{})
						if !ok {
							continue
						}
						id, _ := m["id"].(string)
						if id == "" {
							id, _ = m["type"].(string)
						}
						if id == "" {
							continue
						}
						opts, _ := m["options"].(map[string]interface{})
						if opts == nil {
							opts = make(map[string]interface{})
						}
						for k, v := range m {
							if k != "id" && k != "type" && k != "options" {
								opts[k] = v
							}
						}
						childConfigs = append(childConfigs, WidgetConfig{ID: id, Options: opts})
					}
					walk(childConfigs)
				}
			} else if widgetIDsWithScript[w.ID] && !seen[w.ID] {
				seen[w.ID] = true
				ids = append(ids, w.ID)
			}
		}
	}
	walk(configs)
	return ids
}

// widgetListContains reports whether id appears in a widget config
// list, recursing into split-column children (same child-shape parsing
// as collectActiveWidgetIDs).
func widgetListContains(cfgs []WidgetConfig, id string) bool {
	var walk func([]WidgetConfig) bool
	walk = func(cs []WidgetConfig) bool {
		for _, w := range cs {
			if w.ID == id {
				return true
			}
			if w.ID == "split-column" {
				if rawChildren, ok := w.Options["widgets"].([]interface{}); ok {
					var childConfigs []WidgetConfig
					for _, raw := range rawChildren {
						m, ok := raw.(map[string]interface{})
						if !ok {
							continue
						}
						cid, _ := m["id"].(string)
						if cid == "" {
							cid, _ = m["type"].(string)
						}
						if cid == "" {
							continue
						}
						childConfigs = append(childConfigs, WidgetConfig{ID: cid})
					}
					if walk(childConfigs) {
						return true
					}
				}
			}
		}
		return false
	}
	return walk(cfgs)
}

// ── serveIndex ───────────────────────────────────────────────
func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Snapshot of the current config; hot-reloaded changes land here on
	// the next render after watchConfig swaps the pointer.
	cfg := getConfig()

	pageCache.RLock()
	entry, found := pageCache.data["/"]
	pageCache.RUnlock()
	if found && time.Now().Before(entry.expiresAt) {
		serveHTML(w, r, entry.data)
		return
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	registry := widgets.Registry()
	ctx := widgetRenderContext(rng, nil)

	// ── Profile image ──
	// Picked once per page; carried on the base ctx so the profile
	// widget renders the same image in the header or in a body column.
	ctx.ProfileImageURL = pickImage(appCfg.profileDir, "/api/images/profile/", rng)

	// ── Render widgets and collect HTML ──
	widgetMap := make(map[string]template.HTML)
	var allConfigs []WidgetConfig
	widgetIndex := 0

	for _, col := range cfg.Columns {
		for _, wCfg := range col.Widgets {
			allConfigs = append(allConfigs, wCfg)

			widget, ok := registry[wCfg.ID]
			if !ok {
				log.Printf("serveIndex: unknown widget %q in config — skipping", wCfg.ID)
				widgetIndex++
				continue
			}
			key := fmt.Sprintf("%s-%d", wCfg.ID, widgetIndex)
			widgetIndex++

			ctxWithOpts := ctx
			ctxWithOpts.Options = wCfg.Options
			html, err := widget.Render(ctxWithOpts)
			if err != nil {
				log.Printf("serveIndex: widget %q render error: %v", wCfg.ID, err)
				continue
			}
			widgetMap[key] = html
		}
	}

	// ── Collect active widget IDs for loading their JS ──
	// Header widgets can also carry client JS (e.g. a body widget placed
	// in the header), so walk header + column configs together.
	combinedConfigs := append(append([]WidgetConfig{}, cfg.Header...), allConfigs...)
	activeWidgetIDs := collectActiveWidgetIDs(combinedConfigs)

	// ── Columns for template ──
	type tmplWidget struct{ ID, Key string }
	type tmplColumn struct {
		Widgets []tmplWidget
	}
	var tmplCols []tmplColumn
	var gridParts []string
	widgetIndex = 0
	for _, col := range cfg.Columns {
		tc := tmplColumn{}
		for _, w := range col.Widgets {
			key := fmt.Sprintf("%s-%d", w.ID, widgetIndex)
			widgetIndex++
			tc.Widgets = append(tc.Widgets, tmplWidget{ID: w.ID, Key: key})
		}
		tmplCols = append(tmplCols, tc)

		switch col.Size {
		case "small":
			gridParts = append(gridParts, "250px")
		default:
			gridParts = append(gridParts, "1fr")
		}
	}
	gridCols := strings.Join(gridParts, " ")

	// ── Fonts ──
	fontLatinMap := map[string]string{
		"inter": "'Inter', sans-serif",
	}
	fontJpMap := map[string]string{
		"dotgothic16": "'DotGothic16', monospace",
	}
	fontClockMap := map[string]string{
		"orbitron": "'Orbitron', monospace",
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
	} else if cfg.FontLatin == "" {
		fontPixel = "'Inter', 'DotGothic16', monospace"
	}
	fontDoto := fontClockMap[cfg.FontClock]
	fontClass := "font-" + cfg.FontClock

	clockClass := ""
	switch cfg.ClockTheme {
	case "light":
		clockClass = "clock-force-light"
	case "dark":
		clockClass = "clock-force-dark"
	}

	// ── Background ──
	bgImageURL := pickImage(appCfg.bgDir, "/api/images/bg/", rng)
	bgHTML := template.HTML(`<div id="bg"></div>`)
	if bgImageURL != "" {
		cls := ""
		if cfg.BGBlur {
			cls = ` class="blurred"`
		}
		bgHTML = template.HTML(`<div id="bg"` + cls + `><img src="` + bgImageURL + `" alt=""></div>`)
	}

	// ── Footer items (live values; the footer polls /api/footer-stats
	// via htmx every 2s to keep them fresh) ──
	footerHTML := buildFooterHTML(cfg.FooterItems, currentSystemStats())

	// ── Header widgets ──
	// The header strip is configured like a column: an ordered widget
	// list (config's `header` section, default: clock, identity,
	// profile). Header widgets render chrome-less via RenderHeader;
	// a body widget placed in the header renders with its full chrome.
	headerConfigs := cfg.Header
	if len(headerConfigs) == 0 {
		headerConfigs = defaultConfig().Header
	}
	var headerParts []string
	searchInHeader := false
	for _, hcfg := range headerConfigs {
		if hcfg.ID == "search" {
			searchInHeader = true
		}
		widget, ok := registry[hcfg.ID]
		if !ok {
			log.Printf("serveIndex: unknown header widget %q in config — skipping", hcfg.ID)
			continue
		}
		hctx := ctx
		hctx.Options = hcfg.Options
		var html template.HTML
		var err error
		if hw, ok := widget.(widgets.HeaderWidget); ok {
			html, err = hw.RenderHeader(hctx)
		} else {
			html, err = widget.Render(hctx)
		}
		if err != nil {
			log.Printf("serveIndex: header widget %q render error: %v", hcfg.ID, err)
			continue
		}
		headerParts = append(headerParts,
			`<div class="header-widget" data-widget="`+hcfg.ID+`">`+string(html)+`</div>`)
	}
	headerHTML := template.HTML(strings.Join(headerParts, ""))

	// ── Search strip ──
	// The search widget's classic home is a full-width strip below the
	// header. It renders there only when search isn't explicitly placed
	// in the header or in a column.
	searchInColumns := false
	for _, col := range cfg.Columns {
		if widgetListContains(col.Widgets, "search") {
			searchInColumns = true
			break
		}
	}
	var searchStripHTML template.HTML
	if !searchInHeader && !searchInColumns {
		if sw, ok := registry["search"]; ok {
			if hw, ok := sw.(widgets.HeaderWidget); ok {
				if html, err := hw.RenderHeader(ctx); err == nil {
					searchStripHTML = html
				}
			}
		}
	}

	// ── Critical style ──
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

	// ── Theme CSS — load the file declared in config ──
	var themeCSSLinks strings.Builder
	if cfg.ThemeFile != "" {
		// Verify the file exists before linking
		if _, err := os.Stat(filepath.Join(appCfg.themesDir, cfg.ThemeFile)); err == nil {
			themeCSSLinks.WriteString(fmt.Sprintf(`<link rel="stylesheet" href="/api/themes/%s">`+"\n", cfg.ThemeFile))
		} else {
			log.Printf("serveIndex: theme_file %q not found in %s", cfg.ThemeFile, appCfg.themesDir)
		}
	}

	// ── JS seed (data the server-rendered widgets no longer need — e.g.
	// bookmarks/quick/recent — now lives in each widget's data-state) ──
	// Currently empty; kept as a hook for future client-side seeds.
	initialData := map[string]interface{}{}
	jsonInitial, _ := json.Marshal(initialData)

	pageData := struct {
		BodyClass       string
		BodyData        template.HTMLAttr
		CriticalStyle   template.CSS
		FontStyle       template.CSS
		BgHTML          template.HTML
		UILang          string
		TitleLang       string
		HeaderHTML      template.HTML
		SearchStripHTML template.HTML
		Widgets         map[string]template.HTML
		Columns         []tmplColumn
		GridCols        string
		InitialData     template.JS
		ActiveWidgetIDs []string
		ThemeCSSLinks   template.HTML
		FooterHTML      template.HTML
	}{
		BodyClass: strings.TrimSpace("theme-" + cfg.Theme + " " + fontClass + " " + clockClass),
		BodyData: template.HTMLAttr(fmt.Sprintf(
			`data-clock-format="%s" data-clock-seconds="%v" data-ui-lang="%s"`,
			cfg.ClockFormat, cfg.ClockSeconds, cfg.UILang,
		)),
		CriticalStyle:   criticalStyle,
		FontStyle:       fontStyle,
		BgHTML:          bgHTML,
		UILang:          cfg.UILang,
		TitleLang:       cfg.TitleLang,
		HeaderHTML:      headerHTML,
		SearchStripHTML: searchStripHTML,
		Widgets:         widgetMap,
		Columns:         tmplCols,
		GridCols:        gridCols,
		InitialData:     template.JS(jsonInitial),
		ActiveWidgetIDs: activeWidgetIDs,
		ThemeCSSLinks:   template.HTML(themeCSSLinks.String()),
		FooterHTML:      template.HTML(footerHTML),
	}

	var buf bytes.Buffer
	if err := indexTmpl.Execute(&buf, pageData); err != nil {
		http.Error(w, "render error: "+err.Error(), 500)
		return
	}
	html := buf.Bytes()

	pageCache.Lock()
	pageCache.data["/"] = &cacheEntry{data: html, expiresAt: time.Now().Add(cacheTTL)}
	pageCache.Unlock()

	serveHTML(w, r, html)
}

// ── serveHTML ──────────────────────────────────────────────────
// Writes the index page with an ETag and revalidation semantics: the
// browser always revalidates (no-cache) but a tab opened within the
// cache TTL — or whenever the page bytes are unchanged — gets a 304
// instead of re-downloading the full HTML.
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

// ── cacheControl ───────────────────────────────────────────────
// cacheControl marks a handler's responses cacheable with a fixed
// Cache-Control value.
func cacheControl(value string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", value)
		next.ServeHTTP(w, r)
	})
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
