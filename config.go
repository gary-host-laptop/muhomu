package main

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Theme        string `yaml:"theme"`
	FontLatin    string `yaml:"font_latin"`
	FontJP       string `yaml:"font_jp"`
	FontClock    string `yaml:"font_clock"`
	ClockFormat  string `yaml:"clock_format"`
	ClockSeconds bool   `yaml:"clock_seconds"`
	ClockTheme   string `yaml:"clock_theme"`
	BGBlur       bool   `yaml:"bg_blur"`
	UILang       string `yaml:"ui_lang"`
	TitleLang    string `yaml:"title_lang"`
	Username     string `yaml:"username"`
	SearchTarget string `yaml:"search_target"`

	Location struct {
		City string `yaml:"city"`
		Lat  string `yaml:"lat"`
		Lon  string `yaml:"lon"`
	} `yaml:"location"`

	SearchEngines []SearchEngine `yaml:"search_engines"`
	JLPTLevel     string         `yaml:"jlpt_level"`

	ProfileImagesDir string `yaml:"profile_images_dir"`
	BGImagesDir      string `yaml:"bg_images_dir"`
	WidgetImagesDir  string `yaml:"widget_images_dir"`

	// Columns is the authoritative declaration of layout.
	// Each column declares its size and the ordered list of widgets
	// it contains. Order is determined by position in the list —
	// no separate order field needed. Column membership is declared
	// by which column the widget appears under — no col field needed.
	// Both Col and Order on individual widgets were historical residues
	// of the database-driven layout; they are superseded by this form.
	Columns []ColumnConfig `yaml:"columns"`
}

type SearchEngine struct {
	Name    string `yaml:"name"`
	URL     string `yaml:"url"`
	Default bool   `yaml:"default"`
}

// ColumnConfig declares a column's size and the widgets it contains.
// Size is either "small" (250px fixed) or "full" (remaining space).
type ColumnConfig struct {
	Size    string         `yaml:"size"`
	Widgets []WidgetConfig `yaml:"widgets"`
}

// WidgetConfig identifies a widget within a column.
// Additional fields may be added per-widget in future
// to carry widget-specific configuration.
type WidgetConfig struct {
	ID      string                 `yaml:"id"`
	Options map[string]interface{} `yaml:"options"`
}

var cfg AppConfig

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
			{Name: "youtube", URL: "https://www.youtube.com/results?search_query="},
			{Name: "github", URL: "https://github.com/search?q="},
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
			log.Printf("config: %s not found, using defaults", path)
		} else {
			log.Printf("config: error reading %s: %v — using defaults", path, err)
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

// allWidgetIDs returns a flat list of all widget ids across all columns,
// in declaration order. Used to build the widget map and script list.
func allWidgetIDs() []string {
	var ids []string
	for _, col := range cfg.Columns {
		for _, w := range col.Widgets {
			ids = append(ids, w.ID)
		}
	}
	return ids
}

// widgetActive returns true if the given id appears in any column.
func widgetActive(id string) bool {
	for _, col := range cfg.Columns {
		for _, w := range col.Widgets {
			if w.ID == id {
				return true
			}
		}
	}
	return false
}
