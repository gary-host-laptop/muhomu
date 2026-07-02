package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ── fetchAllData ──────────────────────────────────────────────────────────────
// Returns typed values for each key so main.go can use them without
// reflection / type-asserting through []interface{}.
func fetchAllData() (map[string]interface{}, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	result := make(map[string]interface{})

	fetches := []struct {
		key string
		fn  func() (interface{}, error)
	}{
		{"settings", func() (interface{}, error) { return getAllSettings() }},
		{"notes", func() (interface{}, error) { return getNotes() }},
		{"bookmarks", func() (interface{}, error) { return getBookmarks() }},
		{"quick", func() (interface{}, error) { return getQuickAccess() }},
		{"recent", func() (interface{}, error) { return getRecent() }},
		{"profile", func() (interface{}, error) { return getImages("profile") }},
		{"bg", func() (interface{}, error) { return getImages("bg") }},
		{"widget", func() (interface{}, error) { return getImages("widget") }},
	}

	for _, f := range fetches {
		wg.Add(1)
		go func(key string, fn func() (interface{}, error)) {
			defer wg.Done()
			val, err := fn()
			if err != nil {
				log.Printf("fetchAllData: error fetching %s: %v", key, err)
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
	prevCPUTime          time.Time
)

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
	now := time.Now()
	if prevTotal == 0 {
		prevIdle = idle
		prevTotal = total
		prevCPUTime = now
		return 0
	}
	deltaTotal := float64(total - prevTotal)
	deltaIdle := float64(idle - prevIdle)
	cpu := 100 * (deltaTotal - deltaIdle) / deltaTotal
	prevIdle = idle
	prevTotal = total
	prevCPUTime = now
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

func handleStats(w http.ResponseWriter, r *http.Request) {
	cpu := getCPUPercent()
	ram := getRAMPercent()
	rx, tx := getNetSpeeds()
	jsonOK(w, map[string]interface{}{"cpu": cpu, "ram": ram, "rx": rx, "tx": tx})
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

// ── GET /api/data ──────────────────────────────────────────────────────────────

func handleGetData(w http.ResponseWriter, r *http.Request) {
	data, err := fetchAllData()
	if err != nil {
		jsonErr(w, "failed to fetch data", 500)
		return
	}
	settings, _ := data["settings"].(map[string]interface{})
	if settings == nil {
		settings = make(map[string]interface{})
	}
	settings["nt_notes"] = data["notes"]
	settings["nt_bookmarks"] = data["bookmarks"]
	settings["nt_quick"] = data["quick"]
	settings["nt_recent"] = data["recent"]
	settings["nt_profile_images"] = data["profile"]
	settings["nt_bg_images"] = data["bg"]
	settings["nt_widget_images"] = data["widget"]
	jsonOK(w, settings)
}

// ── POST /api/data ─────────────────────────────────────────────────────────────
// Always invalidates page cache so next load reflects the change.

func handlePostData(w http.ResponseWriter, r *http.Request) {
	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, "invalid json", 400)
		return
	}

	settingsToSave := map[string]interface{}{}

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
				log.Printf("handlePostData: unmarshal nt_bookmarks: %v | raw: %s", err, string(raw)[:min(200, len(raw))])
			} else {
				// cache favicons in background so the save returns immediately
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
		case "nt_profile_images", "nt_bg_images", "nt_widget_images":
			// managed via /api/images/* endpoints
		default:
			var val interface{}
			if err := json.Unmarshal(raw, &val); err == nil {
				settingsToSave[key] = val
			}
		}
	}

	if len(settingsToSave) > 0 {
		if err := setSettings(settingsToSave); err != nil {
			jsonErr(w, "db error", 500)
			return
		}
	}

	// Invalidate page cache so next GET / reflects the new data
	invalidateCache()

	jsonOK(w, map[string]string{"ok": "true"})
}

// ── GET /api/bookmarks/search?q= ──────────────────────────────────────────────

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

// ── POST /api/images/upload ────────────────────────────────────────────────────

func handleImageUpload(w http.ResponseWriter, r *http.Request, imagesDir string) {
	imgType := r.URL.Query().Get("type")
	if imgType != "profile" && imgType != "bg" && imgType != "widget" {
		jsonErr(w, "type must be profile, bg, or widget", 400)
		return
	}

	existing, _ := getImages(imgType)
	limit := 5
	if imgType == "bg" || imgType == "widget" {
		limit = 3
	}
	if len(existing) >= limit {
		jsonErr(w, fmt.Sprintf("max %d images", limit), 400)
		return
	}

	r.ParseMultipartForm(20 << 20)
	file, header, err := r.FormFile("image")
	if err != nil {
		jsonErr(w, "no file", 400)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
	if !allowed[ext] {
		jsonErr(w, "unsupported file type", 400)
		return
	}

	filename := fmt.Sprintf("%s-%d%s", imgType, time.Now().UnixNano(), ext)
	dst, err := os.Create(filepath.Join(imagesDir, filename))
	if err != nil {
		jsonErr(w, "save error", 500)
		return
	}
	defer dst.Close()
	io.Copy(dst, file)

	record, err := addImage(imgType, filename)
	if err != nil {
		os.Remove(filepath.Join(imagesDir, filename))
		jsonErr(w, "db error", 500)
		return
	}

	invalidateCache()
	jsonOK(w, record)
}

// ── DELETE /api/images/:filename ───────────────────────────────────────────────

func handleImageDelete(w http.ResponseWriter, r *http.Request, imagesDir string) {
	filename := filepath.Base(r.PathValue("filename"))
	if filename == "" || filename == "." {
		jsonErr(w, "invalid filename", 400)
		return
	}
	os.Remove(filepath.Join(imagesDir, filename))
	deleteImage(filename)
	invalidateCache()
	jsonOK(w, map[string]string{"ok": "true"})
}

// ── GET /api/weather ───────────────────────────────────────────────────────────

func handleWeather(w http.ResponseWriter, r *http.Request) {
	lat := r.URL.Query().Get("lat")
	lon := r.URL.Query().Get("lon")
	if lat == "" || lon == "" {
		jsonErr(w, "lat and lon required", 400)
		return
	}
	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%s&longitude=%s&current_weather=true",
		lat, lon,
	)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		jsonErr(w, "weather fetch failed", 502)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, resp.Body)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ── GET /api/widget-images/next?current=filename ──────────────────────────────
// Returns a random image from the widget images directory that is different
// from the currently displayed one (best-effort; falls back to any image).
// Response: {"url": "/api/widget-images/files/foo.jpg", "filename": "foo.jpg"}

func handleWidgetImageNext(w http.ResponseWriter, r *http.Request) {
	dir := appCfg.widgetImagesDir
	files := listWidgetImages(dir)
	if len(files) == 0 {
		jsonErr(w, "no widget images found", 404)
		return
	}

	current := filepath.Base(r.URL.Query().Get("current"))

	// filter out current if we have more than one image
	candidates := files
	if len(files) > 1 {
		candidates = candidates[:0]
		for _, f := range files {
			if f != current {
				candidates = append(candidates, f)
			}
		}
	}

	pick := candidates[rand.Intn(len(candidates))]
	jsonOK(w, map[string]string{
		"url":      "/api/widget-images/files/" + pick,
		"filename": pick,
	})
}
