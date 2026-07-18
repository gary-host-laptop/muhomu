// Package widgets defines the Widget interface and the registry of all
// available widgets. Each widget is a self-contained unit that owns its
// config schema, data fetching, and html rendering. The coordinator
// (serveIndex) knows nothing about individual widgets — it simply iterates
// the registry, calls Render on each active widget, and places the result
// in the declared column. This reflects the dialectical development of the
// codebase: the contradictions of centralised rendering (one function
// knowing about all widget data needs) are resolved by developing each
// widget's independence, concentrating its concerns in one place.
package widgets

import (
	"database/sql"
	"html/template"
	"math/rand"
)

// ConfigQuote carries a user-defined quote from the YAML config.
type ConfigQuote struct {
	Text   string
	Author string
}

// RenderContext carries the material conditions available to each widget
// at render time. Widgets take only what they need from this context.
type RenderContext struct {
	DB             *sql.DB
	RNG            *rand.Rand
	FaviconDir     string
	ProfileDir     string
	BGDir          string
	WidgetImageDir string
	JLPTLevel      string
	TitleLang      string
	LocationCity   string
	LocationLat    string
	LocationLon    string
	DefaultLED     string
	ConfigQuotes   []ConfigQuote
	Options        map[string]interface{}
}

// Widget is the interface every widget implements. Its two methods
// correspond to the two questions every widget must answer:
// what am i called? and what do i produce?
type Widget interface {
	ID() string
	Render(ctx RenderContext) (template.HTML, error)
}

// Registry returns the map of all available widgets keyed by their id.
// A widget's presence here declares its existence as a possibility;
// its presence in config.yaml declares its existence as an actuality.
func Registry() map[string]Widget {
	all := []Widget{
		&BookmarksWidget{},
		&NotesWidget{},
		&QuickAccessWidget{},
		&RecentlyVisitedWidget{},
		&RSSWidget{},
		&QuoteWidget{},
		&KotobaWidget{},
		&SystemStatsWidget{},

		&ImageWidget{},
		&TimerWidget{},
		&RainWidget{},
		&WeatherWidget{},
		&CalendarWidget{},
		&SplitColumnWidget{},
	}
	m := make(map[string]Widget, len(all))
	for _, w := range all {
		m[w.ID()] = w
	}
	return m
}

// wrap produces the outer widget chrome (title bar + body container)
// around inner html. All widgets share this structure.
// The LED color is determined by (highest priority first):
//  1. Per-widget "led" option in ctx.Options
//  2. Global ctx.DefaultLED (from config's default_led)
//  3. Fallback: var(--accent)
func wrap(ctx RenderContext, id, label, acts, inner string) template.HTML {
	ledColor := ctx.DefaultLED
	if ledColor == "" {
		ledColor = "var(--accent)"
	}
	if c, ok := ctx.Options["led"].(string); ok && c != "" {
		ledColor = c
	}
	return template.HTML(`
<div class="widget" data-widget="` + id + `">
  <div class="widget-title">
    <div class="led" style="--led-color:` + ledColor + `"></div>
    <span class="wt-label" data-widget-label="` + id + `">` + label + `</span>
    ` + acts + `
  </div>
  ` + inner + `
</div>`)
}
