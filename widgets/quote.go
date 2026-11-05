package widgets

import (
	"html/template"
	"math/rand"
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
	if quotes, err := ctx.Store.Quotes(); err == nil && len(quotes) > 0 {
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
	data := struct {
		Text   string
		Author string
	}{qText, q.Author}

	btn, err := renderWidgetTemplate("quote-actions.tmpl", data)
	if err != nil {
		return "", err
	}
	inner, err := renderWidgetTemplate("quote.tmpl", data)
	if err != nil {
		return "", err
	}
	return wrap(ctx, "quote", "名言", string(btn), string(inner)), nil
}
