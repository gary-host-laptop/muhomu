package main

import (
	"net/http"
	"time"
)

// ── GET /api/weather ──────────────────────────────────────────

func handleWeather(w http.ResponseWriter, r *http.Request) {
	cfg := getConfig()
	lat := r.URL.Query().Get("lat")
	lon := r.URL.Query().Get("lon")
	if lat == "" || lon == "" {
		// fall back to config location
		lat = cfg.Location.Lat
		lon = cfg.Location.Lon
	}
	if lat == "" || lon == "" {
		jsonErr(w, "no location configured", 400)
		return
	}
	url := "https://api.open-meteo.com/v1/forecast?latitude=" + lat +
		"&longitude=" + lon + "&current_weather=true"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		jsonErr(w, "weather fetch failed", 502)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}
