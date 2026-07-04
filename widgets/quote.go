package widgets

import (
	"fmt"
	"html/template"
)

type QuoteWidget struct{}

func (w *QuoteWidget) ID() string { return "quote" }

func (w *QuoteWidget) Render(ctx RenderContext) (template.HTML, error) {
	quotes, err := dbQuotes(ctx.DB)
	if err != nil || len(quotes) == 0 {
		quotes = []Quote{
			{Text: "In the middle of difficulty lies opportunity.", Author: "Albert Einstein"},
		}
	}
	q := quotes[ctx.RNG.Intn(len(quotes))]
	inner := fmt.Sprintf(
		`<div class="widget-body" style="padding:8px"><blockquote class="quote-block">
  <p id="quote-text">%s</p>
  <footer id="quote-author">%s</footer>
</blockquote></div>`,
		htmlEscape(q.Text), htmlEscape(q.Author),
	)
	return wrap("quote", "blue", "名言",
		`<button class="wt-act" id="quote-next"><i class="ph-light ph-caret-right"></i></button>`,
		inner), nil
}
