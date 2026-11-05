package main

import (
	"net/http"
	"path/filepath"
	"sort"

	"muhomu/internal/media"
)

// ── GET /api/widget-images/next ───────────────────────────────
// Returns the next image alphabetically, wrapping around at the end.

func handleWidgetImageNext(w http.ResponseWriter, r *http.Request) {
	files := media.ListImages(appCfg.widgetImagesDir)
	if len(files) == 0 {
		jsonErr(w, "no widget images found", 404)
		return
	}
	sort.Strings(files)
	current := filepath.Base(r.URL.Query().Get("current"))
	pick := files[0]
	if current != "" {
		for i, f := range files {
			if f == current {
				pick = files[(i+1)%len(files)]
				break
			}
		}
	}
	jsonOK(w, map[string]string{
		"url":      "/api/widget-images/files/" + pick,
		"filename": pick,
	})
}
