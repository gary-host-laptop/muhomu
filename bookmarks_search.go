package main

import (
	"net/http"
	"strings"

	"muhomu/internal/store"
)

// ── GET /api/bookmarks/search?q= ──────────────────────────────

func handleSearchBookmarks(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		jsonOK(w, []interface{}{})
		return
	}
	results, err := dataStore.SearchBookmarks(q)
	if err != nil {
		jsonErr(w, "search error", 500)
		return
	}
	if results == nil {
		results = []store.BookmarkLink{}
	}
	jsonOK(w, results)
}
