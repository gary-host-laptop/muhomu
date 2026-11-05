package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
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

// ── helpers ───────────────────────────────────────────────────
func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("settings: json encode error: %v", err)
	}
}
