package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"muhomu/widgets"
)

// widgetRenderContext builds the RenderContext used both by serveIndex
// and by the widget fragment endpoints. rng may be nil to allocate a
// fresh one.
func widgetRenderContext(rng *rand.Rand, opts map[string]interface{}) widgets.RenderContext {
	cfg := getConfig()
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	configQuotes := make([]widgets.ConfigQuote, len(quotes))
	for i, q := range quotes {
		configQuotes[i] = widgets.ConfigQuote{Text: q.Text, Author: q.Author}
	}
	searchEngines := make([]widgets.SearchEngine, len(cfg.SearchEngines))
	for i, e := range cfg.SearchEngines {
		searchEngines[i] = widgets.SearchEngine{Name: e.Name, URL: e.URL, Default: e.Default}
	}
	return widgets.RenderContext{
		Store:          dataStore,
		RNG:            rng,
		FaviconDir:     appCfg.faviconDir,
		ProfileDir:     appCfg.profileDir,
		BGDir:          appCfg.bgDir,
		WidgetImageDir: appCfg.widgetImagesDir,
		JLPTLevel:      cfg.JLPTLevel,
		TitleLang:      cfg.TitleLang,
		LocationCity:   cfg.Location.City,
		LocationLat:    cfg.Location.Lat,
		LocationLon:    cfg.Location.Lon,
		DefaultLED:     cfg.DefaultLED,
		ConfigQuotes:   configQuotes,
		Username:       cfg.Username,
		SearchEngines:  searchEngines,
		Options:        opts,
		Stats:          currentSystemStats(),
	}
}

// writeWidget renders and writes a single widget's HTML.
func writeWidget(w http.ResponseWriter, id string, ctx widgets.RenderContext) {
	widget, ok := widgets.Registry()[id]
	if !ok {
		jsonErr(w, "unknown widget", 404)
		return
	}
	html, err := widget.Render(ctx)
	if err != nil {
		log.Printf("widget %q render error: %v", id, err)
		jsonErr(w, "render error", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte(html))
}
