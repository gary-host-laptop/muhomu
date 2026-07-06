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

func (w *QuoteWidget) Script() string {
	return `(function(){
  const textEl=document.getElementById("quote-text");
  const authEl=document.getElementById("quote-author");
  if(!textEl||!authEl)return;
  let idx=0;
  const currentText=textEl.textContent.trim();
  const found=QUOTES.findIndex(q=>q.text===currentText);
  idx=found!==-1?found:Math.floor(Math.random()*QUOTES.length);
  document.getElementById("quote-next")?.addEventListener("click",()=>{
    let next;
    do{next=Math.floor(Math.random()*QUOTES.length);}while(next===idx&&QUOTES.length>1);
    idx=next;textEl.textContent=QUOTES[idx].text;authEl.textContent=QUOTES[idx].author;
  });
})();`
}
