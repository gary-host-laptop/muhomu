package main

import (
	"bufio"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"muhomu/widgets"
)

// ── fetchUserData ─────────────────────────────────────────────
// Fetches only user-produced content from the db concurrently.
// Configuration is read from cfg, not here.
func fetchUserData() (map[string]interface{}, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	result := make(map[string]interface{})

	fetches := []struct {
		key string
		fn  func() (interface{}, error)
	}{
		{"notes",     func() (interface{}, error) { return getNotes() }},
		{"bookmarks", func() (interface{}, error) { return getBookmarks() }},
		{"quick",     func() (interface{}, error) { return getQuickAccess() }},
		{"recent",    func() (interface{}, error) { return getRecent() }},
		{"quotes",    func() (interface{}, error) { return getQuotes() }},
	}

	for _, f := range fetches {
		wg.Add(1)
		go func(key string, fn func() (interface{}, error)) {
			defer wg.Done()
			val, err := fn()
			if err != nil {
				log.Printf("fetchUserData: error fetching %s: %v", key, err)
				return
			}
			mu.Lock()
			result[key] = val
			mu.Unlock()
		}(f.key, f.fn)
	}

	wg.Wait()
	return result, nil
}

// ── system stats ──────────────────────────────────────────────
var (
	prevNetRx, prevNetTx uint64
	prevNetTime          time.Time
	prevIdle, prevTotal  uint64
)

type statsPoint struct {
	CPU float64 `json:"cpu"`
	RAM float64 `json:"ram"`
}

var (
	statsMu      sync.RWMutex
	statsCurrent struct {
		cpu, ram, rx, tx float64
	}
	statsHistory []statsPoint
	statsMax     = 30
)

// startStatsCollector launches a background goroutine that polls
// /proc files every 2 seconds and keeps a ring buffer of history.
func startStatsCollector() {
	go func() {
		for {
			cpu := getCPUPercent()
			ram := getRAMPercent()
			rx, tx := getNetSpeeds()

			statsMu.Lock()
			statsCurrent.cpu = cpu
			statsCurrent.ram = ram
			statsCurrent.rx = rx
			statsCurrent.tx = tx
			statsHistory = append(statsHistory, statsPoint{CPU: cpu, RAM: ram})
			if len(statsHistory) > statsMax {
				statsHistory = statsHistory[len(statsHistory)-statsMax:]
			}
			statsMu.Unlock()

			time.Sleep(2 * time.Second)
		}
	}()
}

func getCPUPercent() float64 {
	f, _ := os.Open("/proc/stat")
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	parts := strings.Fields(scanner.Text())
	if len(parts) < 8 {
		return 0
	}
	user, _    := strconv.ParseUint(parts[1], 10, 64)
	nice, _    := strconv.ParseUint(parts[2], 10, 64)
	system, _  := strconv.ParseUint(parts[3], 10, 64)
	idle, _    := strconv.ParseUint(parts[4], 10, 64)
	iowait, _  := strconv.ParseUint(parts[5], 10, 64)
	irq, _     := strconv.ParseUint(parts[6], 10, 64)
	softirq, _ := strconv.ParseUint(parts[7], 10, 64)

	total := user + nice + system + idle + iowait + irq + softirq
	if prevTotal == 0 {
		prevIdle = idle; prevTotal = total
		return 0
	}
	deltaTotal := float64(total - prevTotal)
	deltaIdle  := float64(idle - prevIdle)
	cpu := 100 * (deltaTotal - deltaIdle) / deltaTotal
	prevIdle = idle; prevTotal = total
	return cpu
}

func getRAMPercent() float64 {
	f, _ := os.Open("/proc/meminfo")
	defer f.Close()
	scanner := bufio.NewScanner(f)
	total, available := uint64(0), uint64(0)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			total, _ = strconv.ParseUint(strings.Fields(line)[1], 10, 64)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			available, _ = strconv.ParseUint(strings.Fields(line)[1], 10, 64)
		}
	}
	if total == 0 {
		return 0
	}
	return 100 * float64(total-available) / float64(total)
}

func getNetSpeeds() (rx, tx float64) {
	f, _ := os.Open("/proc/net/dev")
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var rxBytes, txBytes uint64
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, ":") && !strings.Contains(line, "lo") {
			parts := strings.Fields(line)
			if len(parts) >= 10 {
				rxBytes, _ = strconv.ParseUint(parts[1], 10, 64)
				txBytes, _  = strconv.ParseUint(parts[9], 10, 64)
				break
			}
		}
	}
	now := time.Now()
	if prevNetTime.IsZero() {
		prevNetRx = rxBytes; prevNetTx = txBytes; prevNetTime = now
		return 0, 0
	}
	dt := now.Sub(prevNetTime).Seconds()
	if dt == 0 {
		return 0, 0
	}
	rx = float64(rxBytes-prevNetRx) / dt / 1024
	tx = float64(txBytes-prevNetTx) / dt / 1024
	prevNetRx = rxBytes; prevNetTx = txBytes; prevNetTime = now
	return rx, tx
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	statsMu.RLock()
	cpu := statsCurrent.cpu
	ram := statsCurrent.ram
	rx := statsCurrent.rx
	tx := statsCurrent.tx
	history := make([]statsPoint, len(statsHistory))
	copy(history, statsHistory)
	statsMu.RUnlock()

	jsonOK(w, map[string]interface{}{
		"cpu": cpu, "ram": ram, "rx": rx, "tx": tx,
		"history": history,
	})
}

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonErr(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// ── GET /api/data ─────────────────────────────────────────────
// Returns only user-produced data. Configuration is in config.yaml
// and is not served through this endpoint.

func handleGetData(w http.ResponseWriter, r *http.Request) {
	data, err := fetchUserData()
	if err != nil {
		jsonErr(w, "failed to fetch data", 500)
		return
	}
	jsonOK(w, map[string]interface{}{
		"nt_notes":     data["notes"],
		"nt_bookmarks": data["bookmarks"],
		"nt_quick":     data["quick"],
		"nt_recent":    data["recent"],
		"nt_quotes":    data["quotes"],
	})
}

// ── POST /api/data ────────────────────────────────────────────
// Accepts mutations to user-produced data only.

func handlePostData(w http.ResponseWriter, r *http.Request) {
	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, "invalid json", 400)
		return
	}

	for key, raw := range body {
		switch key {
		case "nt_notes":
			var content string
			if err := json.Unmarshal(raw, &content); err != nil {
				log.Printf("handlePostData: unmarshal nt_notes: %v", err)
			} else {
				setNotes(content)
			}

		case "nt_bookmarks":
			var folders []BookmarkFolder
			if err := json.Unmarshal(raw, &folders); err != nil {
				log.Printf("handlePostData: unmarshal nt_bookmarks: %v | raw: %.200s", err, string(raw))
			} else {
				go func(f []BookmarkFolder) {
					dir := appCfg.faviconDir
					changed := false
					for fi, folder := range f {
						for li, link := range folder.Links {
							local := cacheFaviconForLink(link.Label, link.URL, link.Fav, dir)
							if local != link.Fav {
								f[fi].Links[li].Fav = local
								changed = true
							}
						}
					}
					if err := setBookmarks(f); err != nil {
						log.Printf("handlePostData: setBookmarks: %v", err)
					} else if changed {
						invalidateCache()
					}
				}(folders)
				log.Printf("handlePostData: saving %d bookmark folders", len(folders))
			}

		case "nt_quick":
			var items []QuickItem
			if err := json.Unmarshal(raw, &items); err != nil {
				log.Printf("handlePostData: unmarshal nt_quick: %v", err)
			} else {
				go func(it []QuickItem) {
					dir := appCfg.faviconDir
					changed := false
					for i, item := range it {
						local := cacheFaviconForLink(item.Label, item.URL, item.Favicon, dir)
						if local != item.Favicon {
							it[i].Favicon = local
							changed = true
						}
					}
					if err := setQuickAccess(it); err != nil {
						log.Printf("handlePostData: setQuickAccess: %v", err)
					} else if changed {
						invalidateCache()
					}
				}(items)
			}

		case "nt_recent":
			var items []RecentItem
			if err := json.Unmarshal(raw, &items); err != nil {
				log.Printf("handlePostData: unmarshal nt_recent: %v", err)
			} else {
				setRecent(items)
			}

		default:
			// silently ignore — no longer accept arbitrary kv settings
			log.Printf("handlePostData: ignoring unknown key %q (config.yaml owns this)", key)
		}
	}

	invalidateCache()
	jsonOK(w, map[string]string{"ok": "true"})
}

// ── GET /api/bookmarks/search?q= ──────────────────────────────

func handleSearchBookmarks(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		jsonOK(w, []interface{}{})
		return
	}
	results, err := searchBookmarks(q)
	if err != nil {
		jsonErr(w, "search error", 500)
		return
	}
	if results == nil {
		results = []BookmarkLink{}
	}
	jsonOK(w, results)
}

// ── GET /api/weather ──────────────────────────────────────────

func handleWeather(w http.ResponseWriter, r *http.Request) {
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

// ── GET /api/widget-images/next ───────────────────────────────
// Returns the next image alphabetically, wrapping around at the end.

func handleWidgetImageNext(w http.ResponseWriter, r *http.Request) {
	files := listImagesDir(appCfg.widgetImagesDir)
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

// ── GET /api/quotes ───────────────────────────────────────────

func handleGetQuotes(w http.ResponseWriter, r *http.Request) {
	quotes, err := getQuotes()
	if err != nil {
		jsonErr(w, "db error", 500)
		return
	}
	if quotes == nil {
		quotes = []Quote{}
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
	record, err := addQuote(q.Text, q.Author)
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
	if err := deleteQuote(id); err != nil {
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
	var candidates []Quote
	if len(quotes) > 0 {
		candidates = make([]Quote, len(quotes))
		for i, q := range quotes {
			candidates[i] = Quote{Text: q.Text, Author: q.Author}
		}
	} else {
		var err error
		candidates, err = getQuotes()
		if err != nil || len(candidates) == 0 {
			jsonErr(w, "no quotes available", 404)
			return
		}
	}

	var pick Quote
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


