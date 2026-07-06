package widgets

import (
	"fmt"
	"html/template"
)

// words is a fallback set used for the initial ssr render.
// The full word list lives in static/data/words.js and is used
// by the client-side js for the "next" button with jlpt filtering.
var words = []Word{
	{K: "語",   R: "ご",         M: "word, language", L: "n5"},
	{K: "勉強", R: "べんきょう", M: "study",           L: "n5"},
	{K: "漢字", R: "かんじ",     M: "kanji",           L: "n5"},
	{K: "時間", R: "じかん",     M: "time",            L: "n5"},
	{K: "友達", R: "ともだち",   M: "friend",          L: "n5"},
	{K: "言葉", R: "ことば",     M: "word, language",  L: "n5"},
	{K: "本",   R: "ほん",       M: "book",            L: "n5"},
	{K: "水",   R: "みず",       M: "water",           L: "n5"},
	{K: "空",   R: "そら",       M: "sky",             L: "n5"},
	{K: "夢",   R: "ゆめ",       M: "dream",           L: "n4"},
}

type KotobaWidget struct{}

func (w *KotobaWidget) ID() string { return "kotoba" }

func (w *KotobaWidget) Render(ctx RenderContext) (template.HTML, error) {
	word := words[ctx.RNG.Intn(len(words))]
	inner := fmt.Sprintf(`
<div class="kanji-block">
  <div class="kanji-char" id="kanji-char">
    <a href="https://jisho.org/search/%s%%20%%23kanji" target="_blank"
       style="color:inherit;text-decoration:none;">%s</a>
  </div>
  <div class="kanji-reading" id="kanji-reading">%s</div>
  <div class="kanji-divider"></div>
  <div class="kanji-meta">
    <span class="kanji-level %s" id="kanji-level">%s</span>
  </div>
  <div class="kanji-meaning" id="kanji-meaning">%s</div>
</div>`, word.K, word.K, word.R, word.L, word.L, word.M)
	return wrap("kotoba", "pink", "言葉",
		`<button class="wt-act" id="kanji-next"><i class="ph-light ph-caret-right"></i></button>`,
		inner), nil
}

func (w *KotobaWidget) Script() string {
	return `(function(){
  const charEl=document.getElementById("kanji-char");
  if(!charEl)return;
  function renderWord(w){
    charEl.innerHTML="";
    const a=document.createElement("a");
    a.href="https://jisho.org/search/"+encodeURIComponent(w.k)+"%20%23kanji";
    a.target="_blank";a.style.cssText="color:inherit;text-decoration:none;";a.textContent=w.k;
    charEl.appendChild(a);
    const r=document.getElementById("kanji-reading");if(r)r.textContent=w.r;
    const m=document.getElementById("kanji-meaning");if(m)m.textContent=w.m;
    const l=document.getElementById("kanji-level");if(l){l.textContent=w.l.toUpperCase();l.className="kanji-level "+w.l;}
  }
  let idx=0;
  const cur=charEl.textContent.trim();
  const found=WORDS.findIndex(w=>w.k===cur);
  idx=found!==-1?found:Math.floor(Math.random()*WORDS.length);
  document.getElementById("kanji-next")?.addEventListener("click",()=>{
    let next;
    do{next=Math.floor(Math.random()*WORDS.length);}while(next===idx&&WORDS.length>1);
    idx=next;renderWord(WORDS[idx]);
  });
})();`
}
