package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
// Returns all data in one call — mirrors the old Store.getAll() behaviour.

func handleGetData(w http.ResponseWriter, r *http.Request) {
	settings, err := getAllSettings()
	if err != nil {
		jsonErr(w, "db error", 500)
		return
	}

	notes, _ := getNotes()
	bookmarks, _ := getBookmarks()
	quick, _ := getQuickAccess()
	recent, _ := getRecent()
	profileImgs, _ := getImages("profile")
	bgImgs, _ := getImages("bg")

	settings["nt_notes"] = notes
	settings["nt_bookmarks"] = bookmarks
	settings["nt_quick"] = quick
	settings["nt_recent"] = recent
	settings["nt_profile_images"] = profileImgs
	settings["nt_bg_images"] = bgImgs

	jsonOK(w, settings)
}

// ── POST /api/data ─────────────────────────────────────────────────────────────
// Saves a partial or full data object. Handles special keys separately.

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
			if err := json.Unmarshal(raw, &content); err == nil {
				setNotes(content)
			}

		case "nt_bookmarks":
			var folders []BookmarkFolder
			if err := json.Unmarshal(raw, &folders); err == nil {
				setBookmarks(folders)
			}

		case "nt_quick":
			var items []QuickItem
			if err := json.Unmarshal(raw, &items); err == nil {
				setQuickAccess(items)
			}

		case "nt_recent":
			var items []RecentItem
			if err := json.Unmarshal(raw, &items); err == nil {
				setRecent(items)
			}

		case "nt_profile_images", "nt_bg_images":
			// Images are managed via /api/images endpoints, ignore here

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

// ── POST /api/images/upload?type=profile|bg ────────────────────────────────────

func handleImageUpload(w http.ResponseWriter, r *http.Request, imagesDir string) {
	imgType := r.URL.Query().Get("type")
	if imgType != "profile" && imgType != "bg" && imgType != "widget" {
		jsonErr(w, "type must be profile, bg, or widget", 400)
		return
	}

	// Check limits
	existing, _ := getImages(imgType)
	limit := 5
	if imgType == "bg" || imgType == "widget" {
		limit = 3
	}
	if len(existing) >= limit {
		jsonErr(w, fmt.Sprintf("max %d images", limit), 400)
		return
	}

	r.ParseMultipartForm(20 << 20) // 20MB
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
