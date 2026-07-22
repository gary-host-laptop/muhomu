package widgets

import (
	"fmt"
	"html/template"
)

type QuoteWidget struct{}

func (w *QuoteWidget) ID() string { return "quote" }

func (w *QuoteWidget) Render(ctx RenderContext) (template.HTML, error) {
	qText := "No quotes yet"
	qAuth := ""
	// Prefer DB quotes (user-added via API), fall back to config quotes
	quotes, err := dbQuotes(ctx.DB)
	if err == nil && len(quotes) > 0 {
		q := quotes[ctx.RNG.Intn(len(quotes))]
		qText = q.Text
		qAuth = q.Author
	} else if len(ctx.ConfigQuotes) > 0 {
		q := ctx.ConfigQuotes[ctx.RNG.Intn(len(ctx.ConfigQuotes))]
		qText = q.Text
		qAuth = q.Author
	}
	inner := fmt.Sprintf(
		`<div class="widget-body" style="padding:8px"><blockquote class="quote-block">
  <p data-fetch-key="text">%s</p>
  <footer data-fetch-key="author">%s</footer>
</blockquote></div>`,
		htmlEscape(qText), htmlEscape(qAuth),
	)
	return wrap(ctx, "quote", "名言",
		`<button class="wt-act" data-preload="/api/quote/random" data-preload-render="render_quote" data-preload-init="quote_preload_init"><i class="ph-light ph-caret-right"></i></button>`,
		inner), nil
}


