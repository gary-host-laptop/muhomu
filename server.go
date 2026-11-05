package main

import (
	"net/http"
)

// newMux wires every route. Feature handlers live in their own files;
// this function is the single place that declares the HTTP surface.
func newMux() *http.ServeMux {
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

	fs := http.FileServer(http.Dir(appCfg.staticDir))
	mux.Handle("/static/", http.StripPrefix("/static/", setContentType(fs)))
	mux.HandleFunc("/", serveIndex)

	return mux
}
