package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"muhomu/internal/store"
)

// ── GET /api/quotes ───────────────────────────────────────────

func handleGetQuotes(w http.ResponseWriter, r *http.Request) {
	quotes, err := dataStore.Quotes()
	if err != nil {
		jsonErr(w, "db error", 500)
		return
	}
	if quotes == nil {
		quotes = []store.Quote{}
	}
	jsonOK(w, quotes)
}

// ── POST /api/quotes ──────────────────────────────────────────

func handleAddQuote(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Text   string `json:"text"`
		Author string `json:"author"`
	}
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil || q.Text == "" {
		jsonErr(w, "text required", 400)
		return
	}
	record, err := dataStore.AddQuote(q.Text, q.Author)
	if err != nil {
		jsonErr(w, "db error", 500)
		return
	}
	invalidateCache()
	jsonOK(w, record)
}

// ── DELETE /api/quotes/{id} ───────────────────────────────────

func handleDeleteQuote(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		jsonErr(w, "invalid id", 400)
		return
	}
	if err := dataStore.DeleteQuote(id); err != nil {
		jsonErr(w, "db error", 500)
		return
	}
	invalidateCache()
	jsonOK(w, map[string]string{"ok": "true"})
}

// ── GET /api/quote/random ─────────────────────────────────────
// Returns a random quote from data/quotes.yaml. Falls back to
// DB quotes if the file has none. Pass ?current=text to guarantee
// a different quote is returned.

func handleQuoteRandom(w http.ResponseWriter, r *http.Request) {
	current := strings.TrimSpace(r.URL.Query().Get("current"))

	// Prefer YAML quotes, fall back to DB quotes
	var candidates []store.Quote
	if len(quotes) > 0 {
		candidates = make([]store.Quote, len(quotes))
		for i, q := range quotes {
			candidates[i] = store.Quote{Text: q.Text, Author: q.Author}
		}
	} else {
		var err error
		candidates, err = dataStore.Quotes()
		if err != nil || len(candidates) == 0 {
			jsonErr(w, "no quotes available", 404)
			return
		}
	}

	var pick store.Quote
	if current == "" || len(candidates) <= 1 {
		pick = candidates[rand.Intn(len(candidates))]
	} else {
		// Keep picking until we get a different quote
		for i := 0; i < 20; i++ {
			q := candidates[rand.Intn(len(candidates))]
			if q.Text != current {
				pick = q
				break
			}
		}
		if pick.Text == "" {
			// Fallback: return the next one in order
			for i, q := range candidates {
				if q.Text == current {
					pick = candidates[(i+1)%len(candidates)]
					break
				}
			}
			if pick.Text == "" {
				pick = candidates[0]
			}
		}
	}

	jsonOK(w, pick)
}
