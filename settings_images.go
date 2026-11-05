package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"muhomu/internal/media"
)

// listImageDir returns {name, url} entries for a served image directory.
// Returns an empty (non-nil) slice so the JSON payload always carries []
// rather than null.
func listImageDir(dir, urlPrefix string) []imgInfo {
	out := []imgInfo{}
	for _, name := range media.ListImages(dir) {
		out = append(out, imgInfo{Name: name, URL: urlPrefix + name})
	}
	return out
}

// ── image upload / delete ─────────────────────────────────────
// imageTypeDir maps the API's image "type" to the runtime directory and
// the URL prefix used to serve it.
func imageTypeDir(kind string) (dir, urlPrefix string, ok bool) {
	switch kind {
	case "profile":
		return appCfg.profileDir, "/api/images/profile/", true
	case "bg":
		return appCfg.bgDir, "/api/images/bg/", true
	case "widget":
		return appCfg.widgetImagesDir, "/api/widget-images/files/", true
	}
	return "", "", false
}

const maxImageBytes = 20 << 20 // 20 MB

// POST /api/settings/images?type=profile|bg|widget — multipart
// form field "file". Writes into the corresponding directory.
func handleImageUpload(w http.ResponseWriter, r *http.Request) {
	dir, _, ok := imageTypeDir(r.URL.Query().Get("type"))
	if !ok {
		http.Error(w, "unknown image type", 400)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes)
	if err := r.ParseMultipartForm(maxImageBytes); err != nil {
		http.Error(w, "upload too large or malformed: "+err.Error(), 400)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file: "+err.Error(), 400)
		return
	}
	defer file.Close()

	name := filepath.Base(header.Filename) // strip any path components
	if name == "" || name == "." {
		http.Error(w, "invalid filename", 400)
		return
	}
	if !media.IsAllowedImage(name) {
		http.Error(w, "unsupported image type", 400)
		return
	}

	dst, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		log.Printf("settings: image upload failed: %v", err)
		http.Error(w, "write failed: "+err.Error(), 500)
		return
	}
	defer dst.Close()
	if _, err := dst.ReadFrom(file); err != nil {
		log.Printf("settings: image upload failed: %v", err)
		http.Error(w, "write failed: "+err.Error(), 500)
		return
	}

	// Uploaded images are content-addressed by filename and picked
	// randomly per page render — invalidate the page cache so a new
	// image can show up on the next load.
	invalidateCache()
	widgetThumbsUpdate(dir, name, false)
	writeJSON(w, map[string]interface{}{"ok": true, "name": name})
}

// DELETE /api/settings/images?type=profile|bg|widget&name=file.png
func handleImageDelete(w http.ResponseWriter, r *http.Request) {
	dir, _, ok := imageTypeDir(r.URL.Query().Get("type"))
	if !ok {
		http.Error(w, "unknown image type", 400)
		return
	}
	name := filepath.Base(r.URL.Query().Get("name"))
	if name == "" || name == "." || name == ".." {
		http.Error(w, "invalid name", 400)
		return
	}
	if err := os.Remove(filepath.Join(dir, name)); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "not found", 404)
			return
		}
		log.Printf("settings: image delete failed: %v", err)
		http.Error(w, "delete failed: "+err.Error(), 500)
		return
	}
	invalidateCache()
	widgetThumbsUpdate(dir, name, true)
	writeJSON(w, map[string]interface{}{"ok": true})
}

// widgetThumbsUpdate keeps the image-widget thumbnail dir in sync with
// its originals: generates a thumb after an upload, removes it on
// delete. No-op for the profile and bg dirs, which have no thumbs.
func widgetThumbsUpdate(dir, name string, remove bool) {
	if dir != appCfg.widgetImagesDir {
		return
	}
	media.UpdateThumb(dir, appCfg.widgetThumbsDir, name, remove)
}
