package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"muhomu/internal/media"
	"muhomu/widgets"
)

// serveIndex renders the full new-tab page. It walks the configured
// columns, renders every widget through the registry, assembles the page
// template, and caches the result for cacheTTL.
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
	ctx.ProfileImageURL = media.PickImage(appCfg.profileDir, "/api/images/profile/", rng)

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

	fontClass := fontClassFor(cfg)
	clockClass := clockClassFor(cfg)

	// ── Background ──
	bgImageURL := media.PickImage(appCfg.bgDir, "/api/images/bg/", rng)
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

	criticalStyle := buildThemeStyle(cfg)
	fontStyle := buildFontStyle(cfg)
	themeCSSLinks := buildThemeCSSLinks(cfg)

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
		ThemeCSSLinks:   themeCSSLinks,
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
