package main

import (
	"encoding/json"
	"log"
	"net/http"

	"muhomu/internal/media"
	"muhomu/internal/store"
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
	// Fragment renders may fetch fresh data (the full page render must
	// not — see RenderContext.Fragment).
	ctx.Fragment = true
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
		if err := dataStore.SetNotes(state); err != nil {
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
		var folders []store.BookmarkFolder
		if err := json.Unmarshal([]byte(state), &folders); err != nil {
			jsonErr(w, "invalid state", 400)
			return
		}
		// Persist immediately so the re-render below is consistent, then
		// upgrade favicons to locally cached copies in the background.
		if err := dataStore.SetBookmarks(folders); err != nil {
			log.Printf("handleWidgetPost: setBookmarks: %v", err)
			jsonErr(w, "save failed", 500)
			return
		}
		go func(f []store.BookmarkFolder) {
			dir := appCfg.faviconDir
			changed := false
			for fi := range f {
				for li := range f[fi].Links {
					local := media.CacheFaviconForLink(f[fi].Links[li].Label, f[fi].Links[li].URL, f[fi].Links[li].Fav, dir)
					if local != f[fi].Links[li].Fav {
						f[fi].Links[li].Fav = local
						changed = true
					}
				}
			}
			if changed {
				if err := dataStore.SetBookmarks(f); err != nil {
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
		var items []store.QuickItem
		if err := json.Unmarshal([]byte(state), &items); err != nil {
			jsonErr(w, "invalid state", 400)
			return
		}
		if err := dataStore.SetQuickAccess(items); err != nil {
			log.Printf("handleWidgetPost: setQuickAccess: %v", err)
			jsonErr(w, "save failed", 500)
			return
		}
		go func(it []store.QuickItem) {
			dir := appCfg.faviconDir
			changed := false
			for i := range it {
				local := media.CacheFaviconForLink(it[i].Label, it[i].URL, it[i].Favicon, dir)
				if local != it[i].Favicon {
					it[i].Favicon = local
					changed = true
				}
			}
			if changed {
				if err := dataStore.SetQuickAccess(it); err != nil {
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
		var items []store.RecentItem
		if err := json.Unmarshal([]byte(state), &items); err != nil {
			jsonErr(w, "invalid state", 400)
			return
		}
		if err := dataStore.SetRecent(items); err != nil {
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
