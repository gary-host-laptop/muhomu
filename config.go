package main

import (
	"log"
	"os"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
)

// AppConfig is the full configuration surface. The yaml tags define the
// on-disk format (config.yaml); the json tags exist for the settings API
// round-trip (GET/POST /api/settings), so the settings page can edit the
// file without the server ever hand-writing YAML.
type AppConfig struct {
	Theme        string `yaml:"theme" json:"theme"`
	FontLatin    string `yaml:"font_latin" json:"font_latin"`
	FontJP       string `yaml:"font_jp" json:"font_jp"`
	FontClock    string `yaml:"font_clock" json:"font_clock"`
	ClockFormat  string `yaml:"clock_format" json:"clock_format"`
	ClockSeconds bool   `yaml:"clock_seconds" json:"clock_seconds"`
	ClockTheme   string `yaml:"clock_theme" json:"clock_theme"`
	BGBlur       bool   `yaml:"bg_blur" json:"bg_blur"`
	UILang       string `yaml:"ui_lang" json:"ui_lang"`
	TitleLang    string `yaml:"title_lang" json:"title_lang"`
	Username     string `yaml:"username" json:"username"`
	SearchTarget string `yaml:"search_target" json:"search_target"`
	DefaultLED   string `yaml:"default_led" json:"default_led"`

	Location struct {
		City string `yaml:"city" json:"city"`
		Lat  string `yaml:"lat" json:"lat"`
		Lon  string `yaml:"lon" json:"lon"`
	} `yaml:"location" json:"location"`

	SearchEngines []SearchEngine `yaml:"search_engines" json:"search_engines"`
	JLPTLevel     string         `yaml:"jlpt_level" json:"jlpt_level"`
	FooterItems   []string       `yaml:"footer_items" json:"footer_items"`

	ProfileImagesDir string `yaml:"profile_images_dir" json:"profile_images_dir"`
	BGImagesDir      string `yaml:"bg_images_dir" json:"bg_images_dir"`
	WidgetImagesDir  string `yaml:"widget_images_dir" json:"widget_images_dir"`
	ThemesDir        string `yaml:"themes_dir" json:"themes_dir"`
	ThemeFile        string `yaml:"theme_file" json:"theme_file"`

	// Header is the ordered list of widgets shown in the header strip.
	// It works exactly like a column's widgets list — the header renders
	// them left to right, chrome-less (no title bar / box). The default
	// is clock, identity, profile — the classic header. The search
	// widget can be placed here too; when it is, the full-width search
	// strip below the header is suppressed.
	//
	// Available header widgets: clock, identity, profile, search
	Header []WidgetConfig `yaml:"header" json:"header"`

	// Columns is the authoritative declaration of layout.
	// Each column declares its size and the ordered list of widgets
	// it contains. Order is determined by position in the list —
	// no separate order field needed. Column membership is declared
	// by which column the widget appears under — no col field needed.
	// Both Col and Order on individual widgets were historical residues
	// of the database-driven layout; they are superseded by this form.
	//
	// Available widgets: quick-access, bookmarks, notes, recently-visited,
	// rss, quote, kotoba, system-stats, timer, rain, weather, image, calendar
	Columns []ColumnConfig `yaml:"columns" json:"columns"`
}

type QuoteEntry struct {
	Text   string `yaml:"text" json:"text"`
	Author string `yaml:"author" json:"author"`
}

type SearchEngine struct {
	Name    string `yaml:"name" json:"name"`
	URL     string `yaml:"url" json:"url"`
	Default bool   `yaml:"default" json:"default"`
}

// ColumnConfig declares a column's size and the widgets it contains.
// Size is either "small" (250px fixed) or "full" (remaining space).
type ColumnConfig struct {
	Size    string         `yaml:"size" json:"size"`
	Widgets []WidgetConfig `yaml:"widgets" json:"widgets"`
}

// WidgetConfig identifies a widget within a column.
// Additional fields may be added per-widget in future
// to carry widget-specific configuration.
type WidgetConfig struct {
	ID      string                 `yaml:"id" json:"id"`
	Options map[string]interface{} `yaml:"options" json:"options"`
}

// cfgPtr holds the active configuration. It is loaded once at startup and
// hot-reloaded by watchConfig when config.yaml changes on disk. Readers take
// an immutable snapshot via getConfig — the stored struct is never mutated,
// a reload is a plain pointer swap, so the read path needs no locking.
var cfgPtr atomic.Pointer[AppConfig]

func getConfig() *AppConfig { return cfgPtr.Load() }
func setConfig(c AppConfig) { cfgPtr.Store(&c) }

// quotes holds user-declared quotes loaded from data/quotes.yaml.
var quotes []QuoteEntry

// parseConfigFile reads and parses the config file, returning an error on
// any failure. Used by hot-reload, which must keep the running config
// rather than fall back to defaults.
func parseConfigFile(path string) (AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{}, err
	}
	base := defaultConfig()
	if err := yaml.Unmarshal(data, &base); err != nil {
		return AppConfig{}, err
	}
	return base, nil
}

// watchConfig polls the config file and hot-reloads it on change: the new
// config is atomically swapped in and the page cache invalidated so the
// next render uses the new values. A failed parse (e.g. mid-save) keeps the
// current config and retries on the next tick.
func watchConfig(path string) {
	var lastMod time.Time
	seen := false
	for {
		fi, err := os.Stat(path)
		if err == nil {
			mt := fi.ModTime()
			if seen && !mt.Equal(lastMod) {
				if c, err := parseConfigFile(path); err != nil {
					log.Printf("config: reload of %s failed (%v) — keeping current config", path, err)
				} else {
					setConfig(c)
					invalidateCache()
					total := 0
					for _, col := range c.Columns {
						total += len(col.Widgets)
					}
					log.Printf("config: hot-reloaded %s (%d columns, %d widgets)", path, len(c.Columns), total)
				}
			}
			lastMod = mt
			seen = true
		}
		time.Sleep(2 * time.Second)
	}
}

func loadQuotes(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("quotes: %s not found — no quotes loaded", path)
		} else {
			log.Printf("quotes: error reading %s: %v", path, err)
		}
		return
	}
	var qf struct {
		Quotes []QuoteEntry `yaml:"quotes"`
	}
	if err := yaml.Unmarshal(data, &qf); err != nil {
		log.Printf("quotes: error parsing %s: %v", path, err)
		return
	}
	quotes = qf.Quotes
	log.Printf("quotes: loaded %d from %s", len(quotes), path)
}

func defaultConfig() AppConfig {
	return AppConfig{
		Theme:        "dark",
		FontLatin:    "inter",
		FontJP:       "dotgothic16",
		FontClock:    "orbitron",
		ClockFormat:  "24h",
		ClockSeconds: false,
		ClockTheme:   "theme",
		BGBlur:       true,
		UILang:       "en",
		TitleLang:    "ja",
		SearchTarget: "_self",
		DefaultLED:   "var(--accent)",
		JLPTLevel:    "all",
		SearchEngines: []SearchEngine{
			{Name: "duckduckgo", URL: "https://duckduckgo.com/?q=", Default: true},
			{Name: "youtube", URL: "https://www.youtube.com/results?search_query="},
			{Name: "github", URL: "https://github.com/search?q="},
		},
		Header: []WidgetConfig{
			{ID: "clock"},
			{ID: "identity"},
			{ID: "profile"},
		},
		Columns: []ColumnConfig{
			{Size: "small", Widgets: []WidgetConfig{
				{ID: "quick-access"},
				{ID: "timer"},
				{ID: "rain"},
				{ID: "quote"},
			}},
			{Size: "full", Widgets: []WidgetConfig{
				{ID: "bookmarks"},
				{ID: "notes"},
				{ID: "recently-visited"},
			}},
			{Size: "small", Widgets: []WidgetConfig{
				{ID: "image"},
				{ID: "system-stats"},
				{ID: "kotoba"},
				{ID: "calendar"},
			}},
		},
	}
}

func loadConfig(path string) AppConfig {
	base := defaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("config: %s not found — no widgets will be rendered", path)
			base.Columns = nil
		} else {
			log.Printf("config: error reading %s: %v — no widgets will be rendered", path, err)
			base.Columns = nil
		}
		return base
	}

	if err := yaml.Unmarshal(data, &base); err != nil {
		log.Printf("config: error parsing %s: %v — using defaults", path, err)
		return defaultConfig()
	}

	total := 0
	for _, c := range base.Columns {
		total += len(c.Widgets)
	}
	log.Printf("config: loaded %s (%d columns, %d widgets)", path, len(base.Columns), total)
	return base
}


