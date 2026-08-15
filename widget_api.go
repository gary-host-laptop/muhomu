package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"

	"muhomu/widgets"
)

// ── widget fragment endpoints (htmx) ───────────────────────────
// These power htmx swaps:
//   - GET /api/widget/{id}  renders a widget fresh from server state
//     (used by the quote/kotoba/image "next" buttons via hx-get, and by
//     bookmarks' folder navigation via ?folder=N).
//   - POST /api/widget/{id} persists the widget's full state (sent as a
//     form field "state" containing the JSON data model) and returns the
//     re-rendered widget. Used by the bookmarks / quick-access /
//     recently-visited widgets after client-side mutations.
//
// Both return exactly the HTML a full page render would produce, so the
// Go widget renderer is the single source of truth for widget markup.

// allWidgetConfigs flattens the configured columns into a widget list.
func allWidgetConfigs() []WidgetConfig {
	var out []WidgetConfig
	for _, col := range cfg.Columns {
		out = append(out, col.Widgets...)
	}
	return out
}

// widgetConfigFor returns the first WidgetConfig matching id, recursing
// into split-column children as well.
func widgetConfigFor(id string) (WidgetConfig, bool) {
	var find func(cfgs []WidgetConfig) (WidgetConfig, bool)
	find = func(cfgs []WidgetConfig) (WidgetConfig, bool) {
		for _, w := range cfgs {
			if w.ID == id {
				return w, true
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
						opts, _ := m["options"].(map[string]interface{})
						if opts == nil {
							opts = make(map[string]interface{})
						}
						for k, v := range m {
							if k != "id" && k != "type" && k != "options" {
								opts[k] = v
							}
						}
						childConfigs = append(childConfigs, WidgetConfig{ID: cid, Options: opts})
					}
					if found, ok := find(childConfigs); ok {
						return found, true
					}
				}
			}
		}
		return WidgetConfig{}, false
	}
	return find(allWidgetConfigs())
}

// widgetRenderContext builds the RenderContext used both by serveIndex
// and by the widget fragment endpoints. rng may be nil to allocate a
// fresh one.
func widgetRenderContext(rng *rand.Rand, opts map[string]interface{}) widgets.RenderContext {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	configQuotes := make([]widgets.ConfigQuote, len(quotes))
	for i, q := range quotes {
		configQuotes[i] = widgets.ConfigQuote{Text: q.Text, Author: q.Author}
	}
	return widgets.RenderContext{
		DB:             db,
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

// handleWidgetGet renders one widget from server state, honoring the
// query params a widget needs (e.g. ?current= for quote/image).
func handleWidgetGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	wcfg, ok := widgetConfigFor(id)
	if !ok {
		jsonErr(w, "widget not configured", 404)
		return
	}
	ctx := widgetRenderContext(nil, wcfg.Options)
	ctx.Query = map[string]string{}
	for k := range r.URL.Query() {
		ctx.Query[k] = r.URL.Query().Get(k)
	}
	writeWidget(w, id, ctx)
}

// handleWidgetPost persists a widget's full state (form field "state",
// sent by the client via htmx.ajax) and returns the re-rendered widget.
func handleWidgetPost(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	wcfg, ok := widgetConfigFor(id)
	if !ok {
		jsonErr(w, "widget not configured", 404)
		return
	}
	state := r.FormValue("state")

	switch id {
	case "notes":
		// notes is the one widget where an empty state is legitimate
		// (clearing the note). The client uses hx-swap="none", so no
		// re-render is needed — 204 is enough.
		if err := setNotes(state); err != nil {
			log.Printf("handleWidgetPost: setNotes: %v", err)
			jsonErr(w, "save failed", 500)
			return
		}
		invalidateCache()
		w.WriteHeader(http.StatusNoContent)
		return

	case "bookmarks":
		if state == "" {
			jsonErr(w, "missing state", 400)
			return
		}
		var folders []BookmarkFolder
		if err := json.Unmarshal([]byte(state), &folders); err != nil {
			jsonErr(w, "invalid state", 400)
			return
		}
		// Persist immediately so the re-render below is consistent, then
		// upgrade favicons to locally cached copies in the background.
		if err := setBookmarks(folders); err != nil {
			log.Printf("handleWidgetPost: setBookmarks: %v", err)
			jsonErr(w, "save failed", 500)
			return
		}
		go func(f []BookmarkFolder) {
			dir := appCfg.faviconDir
			changed := false
			for fi := range f {
				for li := range f[fi].Links {
					local := cacheFaviconForLink(f[fi].Links[li].Label, f[fi].Links[li].URL, f[fi].Links[li].Fav, dir)
					if local != f[fi].Links[li].Fav {
						f[fi].Links[li].Fav = local
						changed = true
					}
				}
			}
			if changed {
				if err := setBookmarks(f); err != nil {
					log.Printf("handleWidgetPost: favicon re-save: %v", err)
				} else {
					invalidateCache()
				}
			}
		}(folders)

	case "quick-access":
		if state == "" {
			jsonErr(w, "missing state", 400)
			return
		}
		var items []QuickItem
		if err := json.Unmarshal([]byte(state), &items); err != nil {
			jsonErr(w, "invalid state", 400)
			return
		}
		if err := setQuickAccess(items); err != nil {
			log.Printf("handleWidgetPost: setQuickAccess: %v", err)
			jsonErr(w, "save failed", 500)
			return
		}
		go func(it []QuickItem) {
			dir := appCfg.faviconDir
			changed := false
			for i := range it {
				local := cacheFaviconForLink(it[i].Label, it[i].URL, it[i].Favicon, dir)
				if local != it[i].Favicon {
					it[i].Favicon = local
					changed = true
				}
			}
			if changed {
				if err := setQuickAccess(it); err != nil {
					log.Printf("handleWidgetPost: favicon re-save: %v", err)
				} else {
					invalidateCache()
				}
			}
		}(items)

	case "recently-visited":
		if state == "" {
			jsonErr(w, "missing state", 400)
			return
		}
		var items []RecentItem
		if err := json.Unmarshal([]byte(state), &items); err != nil {
			jsonErr(w, "invalid state", 400)
			return
		}
		if err := setRecent(items); err != nil {
			log.Printf("handleWidgetPost: setRecent: %v", err)
			jsonErr(w, "save failed", 500)
			return
		}

	default:
		jsonErr(w, "widget is read-only", 400)
		return
	}

	invalidateCache()
	ctx := widgetRenderContext(nil, wcfg.Options)
	// Re-render in the same folder view the client was in, so CRUD
	// operations performed from inside a folder keep the user there.
	if id == "bookmarks" {
		ctx.Query = map[string]string{"folder": r.FormValue("folder")}
	}
	writeWidget(w, id, ctx)
}
