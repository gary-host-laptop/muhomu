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
  <p id="quote-text">%s</p>
  <footer id="quote-author">%s</footer>
</blockquote></div>`,
		htmlEscape(qText), htmlEscape(qAuth),
	)
	return wrap(ctx, "quote", "名言",
		`<button class="wt-act" id="quote-next"><i class="ph-light ph-caret-right"></i></button>`,
		inner), nil
}

func (w *QuoteWidget) Script() string {
	return `(function(){
  const textEl=document.getElementById("quote-text");
  const authEl=document.getElementById("quote-author");
  if(!textEl||!authEl)return;
  document.getElementById("quote-next")?.addEventListener("click",async()=>{
    try{
      const r=await fetch("/api/quote/random");
      if(!r.ok)return;
      const q=await r.json();
      textEl.textContent=q.text;authEl.textContent=q.author;
    }catch(e){}
  });
})();`
}
