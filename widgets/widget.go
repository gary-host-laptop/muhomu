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

// StatsPoint is one sample of CPU/RAM percentages.
type StatsPoint struct {
	CPU float64
	RAM float64
}

// SystemStats is a snapshot of the machine's current statistics,
// populated by the coordinator for the system-stats widget.
type SystemStats struct {
	CPU     float64 // percent
	RAM     float64 // percent
	RX      float64 // download KB/s
	TX      float64 // upload KB/s
	History []StatsPoint
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
	// Stats carries the latest system statistics snapshot for the
	// system-stats widget. Set by the coordinator at render time.
	Stats SystemStats
	// Query carries per-request query parameters. It is set by the widget
	// fragment endpoints (htmx) so a widget can vary its render — e.g. the
	// image widget steps sequentially from ?current=, the quote widget
	// avoids repeating the ?current= quote.
	Query map[string]string
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
//
// An optional sixth argument embeds a JSON blob as data-state on the
// widget root, so client JS can read the widget's data model after
// server-side re-renders (the widget remains the single source of truth).
func wrap(ctx RenderContext, id, label, acts, inner string, states ...string) template.HTML {
	ledColor := ctx.DefaultLED
	if ledColor == "" {
		ledColor = "var(--accent)"
	}
	if c, ok := ctx.Options["led"].(string); ok && c != "" {
		ledColor = c
	}
	stateAttr := ""
	if len(states) > 0 && states[0] != "" {
		stateAttr = ` data-state="` + htmlEscape(states[0]) + `"`
	}
	return template.HTML(`
<div class="widget" data-widget="` + id + `"` + stateAttr + `>
  <div class="widget-title">
    <div class="led" style="--led-color:` + ledColor + `"></div>
    <span class="wt-label" data-widget-label="` + id + `">` + label + `</span>
    ` + acts + `
  </div>
  ` + inner + `
</div>`)
}
