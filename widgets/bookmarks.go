package widgets

import (
	"fmt"
	"html/template"
	"strings"
)

type BookmarksWidget struct{}

func (w *BookmarksWidget) ID() string { return "bookmarks" }

func (w *BookmarksWidget) Render(ctx RenderContext) (template.HTML, error) {
	folders, err := dbBookmarks(ctx.DB)
	if err != nil {
		return "", err
	}
	return wrap(ctx, "bookmarks", "ブックマーク",
		`<button class="wt-btn" id="bm-add-folder-btn"><i class="ph-light ph-plus"></i></button>`,
		string(renderBookmarks(folders))), nil
}

func renderBookmarks(folders []BookmarkFolder) template.HTML {
	var sb strings.Builder
	sb.WriteString(`<div class="widget-body"><div class="folder-list" id="folder-list">`)
	for fi, folder := range folders {
		fmt.Fprintf(&sb, `<details class="folder" draggable="true" data-drag-index="%d">
  <summary class="folder-head">
    <span class="folder-icon">◈</span>
    %s
    <span class="folder-head-btns">
      <button class="folder-head-btn"><i class="ph-light ph-pencil-simple"></i></button>
      <button class="folder-head-btn"><i class="ph-light ph-plus"></i></button>
    </span>
  </summary>
  <div class="folder-links grid">`, fi, htmlEscape(folder.Folder))
		for li, link := range folder.Links {
			fav := link.Fav
			if fav == "" {
				fav = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='32' height='32'%3E%3Crect width='32' height='32' fill='%23333'/%3E%3C/svg%3E"
			}
			fmt.Fprintf(&sb,
				`<a href="%s" class="fav-tile" draggable="true" data-drag-index="%d">
  <img class="fav" src="%s" alt="">
  <span>%s</span>
  <button class="tile-edit"><i class="ph-light ph-pencil-simple"></i></button>
</a>`, link.URL, li, fav, htmlEscape(link.Label))
		}
		sb.WriteString(`</div></details>`)
	}
	sb.WriteString(`</div></div>`)
	return template.HTML(sb.String())
}

func (w *BookmarksWidget) Script() string {
	return `(function(){
  if(!document.getElementById("folder-list"))return;
  let _data=(window.__INITIAL_DATA__&&window.__INITIAL_DATA__.nt_bookmarks)||[];
  async function save(){await Store.set("nt_bookmarks",_data);}
  function load(){
    const list=document.getElementById("folder-list");
    const open=new Set();list.querySelectorAll("details.folder").forEach((el,i)=>{if(el.open)open.add(i);});
    list.innerHTML="";
    _data.forEach((folder,fi)=>{
      const details=document.createElement("details");details.className="folder";details.draggable=true;details.dataset.dragIndex=fi;
      const summary=document.createElement("summary");summary.className="folder-head";
      const icon=document.createElement("span");icon.className="folder-icon";icon.textContent="◈";
      summary.appendChild(icon);summary.appendChild(document.createTextNode(" "+folder.folder));
      const hbtns=document.createElement("span");hbtns.className="folder-head-btns";
      const addBtn=document.createElement("button");addBtn.className="folder-head-btn";addBtn.innerHTML='<i class="ph-light ph-plus"></i>';
      addBtn.addEventListener("click",async e=>{
        e.preventDefault();e.stopPropagation();
        const r=await linkPrompt();if(!r||r==="delete")return;
        _data[fi].links.push({label:r.label,url:r.url,fav:r.fav});await save();load();
      });
      const editBtn=document.createElement("button");editBtn.className="folder-head-btn";editBtn.innerHTML='<i class="ph-light ph-pencil-simple"></i>';
      editBtn.addEventListener("click",async e=>{
        e.preventDefault();e.stopPropagation();
        const r=await miniPrompt([{key:"name",placeholder:"folder name"}],{name:_data[fi].folder},{deletable:true});
        if(!r)return;
        if(r==="delete")_data.splice(fi,1);else if(r.name)_data[fi].folder=r.name;
        await save();load();
      });
      hbtns.appendChild(editBtn);hbtns.appendChild(addBtn);summary.appendChild(hbtns);details.appendChild(summary);
      const linksDiv=document.createElement("div");linksDiv.className="folder-links grid";
      folder.links.forEach((link,li)=>{
        const a=document.createElement("a");a.href=link.url;a.className="fav-tile";a.draggable=true;a.dataset.dragIndex=li;
        const img=document.createElement("img");img.className="fav";img.src=link.fav||guessFavicon(link.url);img.alt="";
        img.addEventListener("error",()=>{const l=document.createElement("div");l.className="fav-letter";l.textContent=(link.label||"?")[0];img.replaceWith(l);});
        const lbl=document.createElement("span");lbl.textContent=link.label;
        const edit=document.createElement("button");edit.className="tile-edit";edit.innerHTML='<i class="ph-light ph-pencil-simple"></i>';
        edit.addEventListener("click",async e=>{
          e.preventDefault();e.stopImmediatePropagation();
          const cur=_data[fi].links[li];
          const r=await linkPrompt({label:cur.label,url:cur.url,fav:cur.fav});
          if(!r)return;
          if(r==="delete")_data[fi].links.splice(li,1);else _data[fi].links[li]={label:r.label,url:r.url,fav:r.fav};
          await save();load();
        });
        edit.addEventListener("mousedown",e=>e.stopImmediatePropagation());
        a.appendChild(img);a.appendChild(lbl);a.appendChild(edit);linksDiv.appendChild(a);
      });
      details.appendChild(linksDiv);
      enableDragSort(linksDiv,"a[data-drag-index]",()=>_data[fi].links,(arr)=>{_data[fi].links=arr;save();},load);
      if(open.has(fi))details.open=true;
      list.appendChild(details);
    });
    enableDragSort(list,"details[data-drag-index]",()=>_data,(arr)=>{_data=arr;save();},load);
  }
  document.getElementById("bm-add-folder-btn")?.addEventListener("click",async()=>{
    const r=await miniPrompt([{key:"name",placeholder:"folder name"}]);
    if(!r||!r.name)return;_data.push({folder:r.name,links:[]});await save();load();
  });
  load();
})();`
}
