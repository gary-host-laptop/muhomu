package main

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// AppConfig is the parsed representation of config.yaml.
// It is read once at startup and treated as immutable thereafter.
// All layout and appearance decisions derive from this struct —
// the database is reserved exclusively for user-produced content
// (bookmarks, quick access, recent, notes, quotes).
type AppConfig struct {
	Theme         string `yaml:"theme"`
	FontLatin     string `yaml:"font_latin"`
	FontJP        string `yaml:"font_jp"`
	FontClock     string `yaml:"font_clock"`
	ClockFormat   string `yaml:"clock_format"`
	ClockSeconds  bool   `yaml:"clock_seconds"`
	ClockTheme    string `yaml:"clock_theme"`
	BGBlur        bool   `yaml:"bg_blur"`
	UILang        string `yaml:"ui_lang"`
	TitleLang     string `yaml:"title_lang"`
	Username      string `yaml:"username"`
	SearchTarget  string `yaml:"search_target"`

	Location struct {
		City string `yaml:"city"`
		Lat  string `yaml:"lat"`
		Lon  string `yaml:"lon"`
	} `yaml:"location"`

	SearchEngines []SearchEngine `yaml:"search_engines"`

	JLPTLevel string `yaml:"jlpt_level"`

	// Image directories — override the data-dir-relative defaults.
	ProfileImagesDir string `yaml:"profile_images_dir"`
	BGImagesDir      string `yaml:"bg_images_dir"`
	WidgetImagesDir  string `yaml:"widget_images_dir"`

	Widgets []WidgetConfig `yaml:"widgets"`
}

type SearchEngine struct {
	Name    string `yaml:"name"`
	URL     string `yaml:"url"`
	Default bool   `yaml:"default"`
}

type WidgetConfig struct {
	ID    string `yaml:"id"`
	Col   string `yaml:"col"`
	Order int    `yaml:"order"`
}

// cfg is the global application configuration, set once at startup.
var cfg AppConfig

// defaultConfig returns a sensible default configuration used when
// config.yaml is absent or partially specified.
func defaultConfig() AppConfig {
	return AppConfig{
		Theme:        "dark",
		FontLatin:    "share-tech-mono",
		FontJP:       "dotgothic16",
		FontClock:    "orbitron",
		ClockFormat:  "24h",
		ClockSeconds: false,
		ClockTheme:   "theme",
		BGBlur:       true,
		UILang:       "en",
		TitleLang:    "ja",
		SearchTarget: "_blank",
		JLPTLevel:    "all",
		SearchEngines: []SearchEngine{
			{Name: "duckduckgo", URL: "https://duckduckgo.com/?q=", Default: true},
			{Name: "youtube",    URL: "https://www.youtube.com/results?search_query="},
			{Name: "github",     URL: "https://github.com/search?q="},
		},
		Widgets: []WidgetConfig{
			{ID: "quick-access",     Col: "left",   Order: 1},
			{ID: "timer",            Col: "left",   Order: 2},
			{ID: "rain",             Col: "left",   Order: 3},
			{ID: "quote",            Col: "left",   Order: 4},
			{ID: "bookmarks",        Col: "center", Order: 1},
			{ID: "notes",            Col: "center", Order: 2},
			{ID: "recently-visited", Col: "center", Order: 3},
			{ID: "image",            Col: "right",  Order: 1},
			{ID: "status",           Col: "right",  Order: 2},
			{ID: "system-stats",     Col: "right",  Order: 3},
			{ID: "kotoba",           Col: "right",  Order: 4},
		},
	}
}

// loadConfig reads config.yaml from the given path. If the file does
// not exist, the default config is returned and a notice is logged.
// Fields not present in the file fall back to their zero values, so
// callers should always start from defaultConfig() and merge.
func loadConfig(path string) AppConfig {
	base := defaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("config: %s not found, using defaults", path)
		} else {
			log.Printf("config: error reading %s: %v — using defaults", path, err)
		}
		return base
	}

	// Unmarshal into base so unset fields keep their defaults.
	if err := yaml.Unmarshal(data, &base); err != nil {
		log.Printf("config: error parsing %s: %v — using defaults", path, err)
		return defaultConfig()
	}

	log.Printf("config: loaded %s (%d widgets)", path, len(base.Widgets))
	return base
}

// activeWidgets returns widgets grouped by column, sorted by order.
// Only widgets present in the config are included — absent widgets
// produce no html, no js scope, no dom nodes.
func activeWidgets() (left, center, right []WidgetConfig) {
	for _, w := range cfg.Widgets {
		switch w.Col {
		case "center":
			center = append(center, w)
		case "right":
			right = append(right, w)
		default:
			left = append(left, w)
		}
	}
	sortWidgets := func(ws []WidgetConfig) {
		for i := 1; i < len(ws); i++ {
			for j := i; j > 0 && ws[j].Order < ws[j-1].Order; j-- {
				ws[j], ws[j-1] = ws[j-1], ws[j]
			}
		}
	}
	sortWidgets(left)
	sortWidgets(center)
	sortWidgets(right)
	return
}

// widgetActive returns true if the given widget id is present in the config.
func widgetActive(id string) bool {
	for _, w := range cfg.Widgets {
		if w.ID == id {
			return true
		}
	}
	return false
}
