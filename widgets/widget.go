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
	Options        map[string]interface{}
}

// Widget is the interface every widget implements. Its two methods
// correspond to the two questions every widget must answer:
// what am i called? and what do i produce?
type Widget interface {
	ID() string
	Render(ctx RenderContext) (template.HTML, error)
}

// Scriptable is an optional interface a widget may implement if it
// requires client-side javascript for interactivity. The Script()
// method returns a self-contained js string — no module system,
// no imports, just vanilla js that will be injected into the page
// only when this widget is active. Widgets with no client-side
// behaviour (calendar, status) do not implement this interface.
type Scriptable interface {
	Script() string
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
		&QuoteWidget{},
		&KotobaWidget{},
		&SystemStatsWidget{},
		&StatusWidget{},
		&ImageWidget{},
		&TimerWidget{},
		&RainWidget{},
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
func wrap(id, barColor, label, acts, inner string) template.HTML {
	return template.HTML(`
<div class="widget" data-widget="` + id + `">
  <div class="widget-title">
    <div class="wt-bar ` + barColor + `"></div>
    <span class="wt-label" data-widget-label="` + id + `">` + label + `</span>
    ` + acts + `
  </div>
  ` + inner + `
</div>`)
}
