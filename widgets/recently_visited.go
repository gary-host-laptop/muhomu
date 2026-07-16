package widgets

import (
	"fmt"
	"html/template"
	"strings"
)

type RecentlyVisitedWidget struct{}

func (w *RecentlyVisitedWidget) ID() string { return "recently-visited" }

func (w *RecentlyVisitedWidget) Render(ctx RenderContext) (template.HTML, error) {
	items, err := dbRecent(ctx.DB)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString(`<div class="widget-body"><div class="recent-grid" id="recent-grid">`)
	for _, item := range items {
		fmt.Fprintf(&sb,
			`<div class="recent-tile">
  <div class="rt-name">%s</div>
  <div class="rt-url">%s</div>
  <button class="recent-x"><i class="ph-light ph-x"></i></button>
</div>`, htmlEscape(item.Name), htmlEscape(item.URL))
	}
	sb.WriteString(`</div>
<div class="recent-quick-add">
  <input type="text" id="rqa-name" placeholder="label" autocomplete="off" />
  <input type="text" id="rqa-url" placeholder="https://..." autocomplete="off" />
  <button id="rqa-btn"><span>+</span></button>
</div></div>`)
	return wrap("recently-visited", "pink", "後で見る", "", sb.String()), nil
}

func (w *RecentlyVisitedWidget) Script() string {
	return `(function(){
  if(!document.getElementById("recent-grid"))return;
  let _data=(window.__INITIAL_DATA__&&window.__INITIAL_DATA__.nt_recent)||[];
  async function save(){await Store.set("nt_recent",_data);}
  function load(){
    const grid=document.getElementById("recent-grid");
    grid.innerHTML="";
    _data.forEach((item,i)=>{
      const tile=document.createElement("div");tile.className="recent-tile";
      const name=document.createElement("div");name.className="rt-name";name.textContent=item.name;
      const url=document.createElement("div");url.className="rt-url";url.textContent=item.url;
      tile.appendChild(name);tile.appendChild(url);
      tile.addEventListener("click",()=>{window.location.href=item.url;});
      const x=document.createElement("button");x.className="recent-x";x.innerHTML='<i class="ph-light ph-x"></i>';
      x.addEventListener("click",async e=>{e.stopPropagation();_data.splice(i,1);await save();load();});
      tile.appendChild(x);grid.appendChild(tile);
    });
  }
  async function quickAdd(){
    const name=document.getElementById("rqa-name").value.trim();
    const url=document.getElementById("rqa-url").value.trim();
    if(!name||!url)return;
    _data.unshift({name,url});if(_data.length>8)_data.pop();
    await save();
    document.getElementById("rqa-name").value="";document.getElementById("rqa-url").value="";
    load();
  }
  document.getElementById("rqa-btn")?.addEventListener("click",quickAdd);
  document.getElementById("rqa-url")?.addEventListener("keydown",e=>{if(e.key==="Enter")quickAdd();});
  load();
})();`
}
