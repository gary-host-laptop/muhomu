package main

import (
	"bufio"
	"encoding/json"
	"fmt"
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
	// statsMax is the ring size: 30 visible slots + 1 extra point that the
	// system-stats widget renders one slot off-screen right. At each swap
	// the fresh line is then pixel-identical to the previous frame's
	// resting position, so the CSS scroll glide can slide the newest point
	// in from the right while the oldest slides off-screen left — no point
	// is ever dropped at a swap (see statsGraph in widgets/system_stats.go).
	statsMax = 31
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

// ── rss feed warmer ───────────────────────────────────────────
// startRSSWarmer periodically refreshes every configured rss widget's
// feeds (per-feed TTL / ETag rules apply as usual), so articles stay
// fresh even while no browser is open and full-page renders — which
// never touch the network — almost always find warm cache entries.
// The config walk re-reads the current config on every tick, so
// hot-reloaded feed lists are picked up without a restart.

func startRSSWarmer() {
	warmAll := func() {
		for _, wcfg := range flattenWidgetConfigs(allWidgetConfigs()) {
			if wcfg.ID != "rss" {
				continue
			}
			widgets.WarmRSSCache(wcfg.Options, appCfg.faviconDir)
		}
	}

	// Warm immediately at startup rather than after one period.
	warmAll()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		warmAll()
	}
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
	user, _ := strconv.ParseUint(parts[1], 10, 64)
	nice, _ := strconv.ParseUint(parts[2], 10, 64)
	system, _ := strconv.ParseUint(parts[3], 10, 64)
	idle, _ := strconv.ParseUint(parts[4], 10, 64)
	iowait, _ := strconv.ParseUint(parts[5], 10, 64)
	irq, _ := strconv.ParseUint(parts[6], 10, 64)
	softirq, _ := strconv.ParseUint(parts[7], 10, 64)

	total := user + nice + system + idle + iowait + irq + softirq
	if prevTotal == 0 {
		prevIdle = idle
		prevTotal = total
		return 0
	}
	deltaTotal := float64(total - prevTotal)
	deltaIdle := float64(idle - prevIdle)
	cpu := 100 * (deltaTotal - deltaIdle) / deltaTotal
	prevIdle = idle
	prevTotal = total
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
				txBytes, _ = strconv.ParseUint(parts[9], 10, 64)
				break
			}
		}
	}
	now := time.Now()
	if prevNetTime.IsZero() {
		prevNetRx = rxBytes
		prevNetTx = txBytes
		prevNetTime = now
		return 0, 0
	}
	dt := now.Sub(prevNetTime).Seconds()
	if dt == 0 {
		return 0, 0
	}
	rx = float64(rxBytes-prevNetRx) / dt / 1024
	tx = float64(txBytes-prevNetTx) / dt / 1024
	prevNetRx = rxBytes
	prevNetTx = txBytes
	prevNetTime = now
	return rx, tx
}

// currentSystemStats returns a consistent snapshot of the latest stats
// (values + history) under the stats lock, for the system-stats widget.
func currentSystemStats() widgets.SystemStats {
	statsMu.RLock()
	defer statsMu.RUnlock()
	hist := make([]widgets.StatsPoint, len(statsHistory))
	for i, p := range statsHistory {
		hist[i] = widgets.StatsPoint{CPU: p.CPU, RAM: p.RAM}
	}
	return widgets.SystemStats{
		CPU:     statsCurrent.cpu,
		RAM:     statsCurrent.ram,
		RX:      statsCurrent.rx,
		TX:      statsCurrent.tx,
		History: hist,
	}
}

// ── footer stats fragment (htmx) ──────────────────────────────
// The page footer's live stats row (net up/down speeds, cpu, ram)
// polls this endpoint every 2s via hx-get. It renders the same spans
// serveIndex produces but with current values — the footer is fully
// server-rendered and needs no client JS.

func fmtFooterSpeed(kbps float64) string {
	if kbps >= 1024 {
		return fmt.Sprintf("%.1f MB/s", kbps/1024)
	}
	return fmt.Sprintf("%.1f KB/s", kbps)
}

func buildFooterHTML(footerItems []string, s widgets.SystemStats) string {
	var b strings.Builder
	written := 0
	for _, item := range footerItems {
		var stat string
		switch item {
		case "rx":
			stat = fmt.Sprintf(`<span class="fi-item fi-rx" id="fi-rx">⬇ %s</span>`, fmtFooterSpeed(s.RX))
		case "tx":
			stat = fmt.Sprintf(`<span class="fi-item fi-tx" id="fi-tx">⬆ %s</span>`, fmtFooterSpeed(s.TX))
		case "cpu":
			stat = fmt.Sprintf(`<span class="fi-item fi-cpu" id="fi-cpu">◉ cpu %.1f%%</span>`, s.CPU)
		case "ram":
			stat = fmt.Sprintf(`<span class="fi-item fi-ram" id="fi-ram">◉ ram %.0f%%</span>`, s.RAM)
		default:
			continue
		}
		if written > 0 {
			b.WriteString(`<span class="fi-sep">·</span>`)
		}
		b.WriteString(stat)
		written++
	}

	// The version and GitHub link are fixed furniture: they always sit at
	// the far right, regardless of footer_items order or membership.
	b.WriteString(`<span class="fi-right">`)
	b.WriteString(`<span class="fi-version">muhomu v` + version + `</span>`)
	b.WriteString(`<span class="fi-sep">·</span>`)
	b.WriteString(`<a class="fi-item fi-github" href="` + githubRepoURL +
		`" target="_blank" rel="noopener" title="muhomu on GitHub" aria-label="muhomu on GitHub"><i class="ph-light ph-github-logo"></i></a>`)
	b.WriteString(`</span>`)
	return b.String()
}

func handleFooterStats(w http.ResponseWriter, r *http.Request) {
	cfg := getConfig()
	out := buildFooterHTML(cfg.FooterItems, currentSystemStats())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte(out))
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
