package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ── runtime paths ─────────────────────────────────────────────
// Set once at startup from the CLI flags; the settings API needs them
// to know which file to write. Everything else (image dirs, themes dir)
// lives in appCfg.
var (
	configFilePath string
	quotesFilePath string
)

// settingsTmpl is the parsed settings page template.
var settingsTmpl *template.Template

// ── GET /settings ─────────────────────────────────────────────
func handleSettingsPage(w http.ResponseWriter, r *http.Request) {
	if settingsTmpl == nil {
		http.Error(w, "settings template not loaded", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := settingsTmpl.Execute(w, nil); err != nil {
		log.Printf("settings: template error: %v", err)
	}
}

// ── settings API payloads ─────────────────────────────────────
type imgInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type settingsPayload struct {
	Config   AppConfig            `json:"config"`
	Defaults AppConfig            `json:"defaults"`
	Quotes   []QuoteEntry         `json:"quotes"`
	Themes   []string             `json:"themes"`
	Images   map[string][]imgInfo `json:"images"`
}

// listImageDir returns {name, url} entries for a served image directory.
// Returns an empty (non-nil) slice so the JSON payload always carries []
// rather than null.
func listImageDir(dir, urlPrefix string) []imgInfo {
	out := []imgInfo{}
	for _, name := range listImagesDir(dir) {
		out = append(out, imgInfo{Name: name, URL: urlPrefix + name})
	}
	return out
}

// listThemeFiles returns the custom theme CSS files in the themes dir,
// or an empty (non-nil) slice.
func listThemeFiles(dir string) []string {
	files := []string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return files
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".css") {
			files = append(files, e.Name())
		}
	}
	return files
}

// ── GET /api/settings ─────────────────────────────────────────
// Returns the whole configuration surface as JSON: the current config,
// the defaults (for reset), the quotes file, the available themes, and
// the current profile/bg/widget image listings. The settings page is
// purely data-driven from this response.
func handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	cfg := getConfig()
	payload := settingsPayload{
		Config:   *cfg,
		Defaults: defaultConfig(),
		Quotes:   quotes,
		Themes:   listThemeFiles(appCfg.themesDir),
		Images: map[string][]imgInfo{
			"profile": listImageDir(appCfg.profileDir, "/api/images/profile/"),
			"bg":      listImageDir(appCfg.bgDir, "/api/images/bg/"),
			"widget":  listImageDir(appCfg.widgetImagesDir, "/api/widget-images/files/"),
		},
	}
	writeJSON(w, payload)
}

// ── POST /api/settings ────────────────────────────────────────
// Accepts a (possibly partial) AppConfig as JSON, merges it over the
// running config, validates that the result still parses as YAML, and
// writes it to config.yaml atomically. The existing hot-reload then
// picks the file up — but we also apply it immediately (setConfig +
// cache invalidation) so the new tab page reflects the change at once.
func handleSettingsPost(w http.ResponseWriter, r *http.Request) {
	if configFilePath == "" {
		http.Error(w, "config path not set", 500)
		return
	}

	merged := *getConfig() // start from the running config
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(&merged); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), 400)
		return
	}

	// Validation: the written file must always parse back into a config.
	// This is the one hard guarantee — a bad write would brick hot-reload.
	data, err := yaml.Marshal(&merged)
	if err != nil {
		http.Error(w, "config could not be serialized: "+err.Error(), 400)
		return
	}
	var check AppConfig
	if err := yaml.Unmarshal(data, &check); err != nil {
		http.Error(w, "config would not parse: "+err.Error(), 400)
		return
	}

	if err := saveConfigFile(configFilePath, &merged); err != nil {
		log.Printf("settings: save failed: %v", err)
		http.Error(w, "save failed: "+err.Error(), 500)
		return
	}

	setConfig(merged)
	invalidateCache()
	log.Printf("settings: config saved to %s", configFilePath)

	writeJSON(w, map[string]interface{}{"config": merged})
}

// saveConfigFile writes cfg to path atomically (temp file + rename) so
// a crash mid-write can never leave a half-written config.yaml behind.
func saveConfigFile(path string, cfg *AppConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ── POST /api/settings/reset ──────────────────────────────────
// Writes the built-in default config over config.yaml and applies it.
func handleSettingsReset(w http.ResponseWriter, r *http.Request) {
	if configFilePath == "" {
		http.Error(w, "config path not set", 500)
		return
	}
	def := defaultConfig()
	if err := saveConfigFile(configFilePath, &def); err != nil {
		log.Printf("settings: reset failed: %v", err)
		http.Error(w, "reset failed: "+err.Error(), 500)
		return
	}
	setConfig(def)
	invalidateCache()
	log.Printf("settings: config reset to defaults")
	writeJSON(w, map[string]interface{}{"config": def})
}

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
	if !allowedImageExts[strings.ToLower(filepath.Ext(name))] {
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
	os.MkdirAll(appCfg.widgetThumbsDir, 0755)
	if remove {
		os.Remove(filepath.Join(appCfg.widgetThumbsDir, name+".thumb.jpg"))
		return
	}
	ensureWidgetThumb(dir, appCfg.widgetThumbsDir, name)
}

// ── helpers ───────────────────────────────────────────────────
func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("settings: json encode error: %v", err)
	}
}
