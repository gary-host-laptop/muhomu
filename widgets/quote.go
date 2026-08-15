package widgets

import (
	"fmt"
	"html/template"
	"math/rand"
	"net/url"
)

type QuoteWidget struct{}

func (w *QuoteWidget) ID() string { return "quote" }

// pickQuote returns a random quote, avoiding current when possible so
// the "next" button never repeats the quote that is on screen.
func pickQuote(pool []Quote, current string, rng *rand.Rand) Quote {
	if len(pool) == 0 {
		return Quote{}
	}
	if current == "" || len(pool) == 1 {
		return pool[rng.Intn(len(pool))]
	}
	for i := 0; i < 20; i++ {
		q := pool[rng.Intn(len(pool))]
		if q.Text != current {
			return q
		}
	}
	// fallback: the next one in order
	for i, q := range pool {
		if q.Text == current {
			return pool[(i+1)%len(pool)]
		}
	}
	return pool[0]
}

func (w *QuoteWidget) Render(ctx RenderContext) (template.HTML, error) {
	current := ctx.Query["current"]

	var q Quote
	if quotes, err := dbQuotes(ctx.DB); err == nil && len(quotes) > 0 {
		q = pickQuote(quotes, current, ctx.RNG)
	} else if len(ctx.ConfigQuotes) > 0 {
		pool := make([]Quote, len(ctx.ConfigQuotes))
		for i, cq := range ctx.ConfigQuotes {
			pool[i] = Quote{Text: cq.Text, Author: cq.Author}
		}
		q = pickQuote(pool, current, ctx.RNG)
	}

	qText := q.Text
	if qText == "" {
		qText = "No quotes yet"
	}

	// The next button re-renders this widget via htmx. It carries the
	// current quote text as ?current= so the server avoids repeating it.
	btn := fmt.Sprintf(
		`<button class="wt-act" hx-get="/api/widget/quote?current=%s" hx-target="closest .widget" hx-swap="outerHTML" title="next quote"><i class="ph-light ph-caret-right"></i></button>`,
		url.QueryEscape(qText),
	)
	inner := fmt.Sprintf(
		`<div class="widget-body" style="padding:8px"><blockquote class="quote-block">
  <p>%s</p>
  <footer>%s</footer>
</blockquote></div>`,
		htmlEscape(qText), htmlEscape(q.Author),
	)
	return wrap(ctx, "quote", "名言", btn, inner), nil
}
