package main

import (
	"math/rand"
	"net/http"

	"muhomu/widgets"
)

// ── GET /api/kotoba/next ──────────────────────────────────────
// Returns a random Japanese word, optionally filtered by JLPT level.

func handleKotobaNext(w http.ResponseWriter, r *http.Request) {
	words := widgets.AllWords()
	if len(words) == 0 {
		jsonErr(w, "no words available", 500)
		return
	}
	level := r.URL.Query().Get("level")
	var candidates []widgets.Word
	if level == "" || level == "all" {
		candidates = words
	} else {
		for _, w := range words {
			if w.L == level {
				candidates = append(candidates, w)
			}
		}
		if len(candidates) == 0 {
			candidates = words
		}
	}
	pick := candidates[rand.Intn(len(candidates))]
	jsonOK(w, pick)
}
