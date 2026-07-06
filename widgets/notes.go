package widgets

import (
	"fmt"
	"html/template"
)

type NotesWidget struct{}

func (w *NotesWidget) ID() string { return "notes" }

func (w *NotesWidget) Render(ctx RenderContext) (template.HTML, error) {
	content, err := dbNotes(ctx.DB)
	if err != nil {
		content = ""
	}
	inner := fmt.Sprintf(
		`<div class="widget-body"><textarea id="notes" placeholder="// type here. auto-saves.">%s</textarea></div>`,
		htmlEscape(content),
	)
	return wrap("notes", "green", "メモ帳",
		`<button class="wt-act" id="notes-clear"><i class="ph-light ph-trash"></i></button>`,
		inner), nil
}

func (w *NotesWidget) Script() string {
	return `(function(){
  const notesEl=document.getElementById("notes");
  if(!notesEl)return;
  let saveTimer=null;
  notesEl.addEventListener("input",()=>{
    clearTimeout(saveTimer);
    saveTimer=setTimeout(()=>Store.set("nt_notes",notesEl.value),800);
  });
  document.getElementById("notes-clear")?.addEventListener("click",async()=>{
    if(confirm("clear notes?")){clearTimeout(saveTimer);notesEl.value="";await Store.set("nt_notes","");}
  });
  window.addEventListener("beforeunload",()=>{
    if(!saveTimer)return;
    clearTimeout(saveTimer);saveTimer=null;
    navigator.sendBeacon("/api/data",new Blob([JSON.stringify({nt_notes:notesEl.value})],{type:"application/json"}));
  });
})();`
}
