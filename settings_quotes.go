package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ── POST /api/settings/quotes ─────────────────────────────────
// Saves the quotes file (data/quotes.yaml) from a {quotes: [...]}
// payload and updates the in-memory list so the quote widget sees the
// change immediately.
func handleQuotesSave(w http.ResponseWriter, r *http.Request) {
	if quotesFilePath == "" {
		http.Error(w, "quotes path not set", 500)
		return
	}
	var body struct {
		Quotes []QuoteEntry `json:"quotes"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), 400)
		return
	}
	for _, q := range body.Quotes {
		if strings.TrimSpace(q.Text) == "" {
			http.Error(w, "quotes cannot contain empty text", 400)
			return
		}
	}
	out := struct {
		Quotes []QuoteEntry `yaml:"quotes"`
	}{Quotes: body.Quotes}
	data, err := yaml.Marshal(&out)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if err := os.WriteFile(quotesFilePath, data, 0644); err != nil {
		log.Printf("settings: quotes save failed: %v", err)
		http.Error(w, "save failed: "+err.Error(), 500)
		return
	}
	quotes = body.Quotes
	invalidateCache()
	writeJSON(w, map[string]interface{}{"quotes": quotes})
}
