/* Quick access CRUD — state managed in __INITIAL_DATA__, saved via Store.set */
(function(){
  if(!document.querySelector(".quick-links"))return;
  let _data=(window.__INITIAL_DATA__&&window.__INITIAL_DATA__.nt_quick)||[];
  async function save(){await Store.set("nt_quick",_data);}
  function load(){
    const container=document.querySelector(".quick-links");
    if(!container)return;
    container.innerHTML="";
    _data.forEach((item,i)=>{
      const a=document.createElement("a");a.href=item.url;a.draggable=true;a.dataset.dragIndex=i;
      const img=document.createElement("img");img.className="ql-fav";img.src=item.favicon||item.fav||guessFavicon(item.url);img.alt="";
      a.appendChild(img);a.appendChild(document.createTextNode(" "+item.label));
      const grp=document.createElement("span");grp.style.cssText="margin-left:auto;display:none;align-items:center;gap:0;flex-shrink:0;";grp.className="qa-btn-group";
      const edit=document.createElement("button");edit.className="qa-edit";edit.innerHTML='<i class="ph-light ph-pencil-simple"></i>';
      edit.addEventListener("click",async e=>{
        e.preventDefault();e.stopPropagation();
        const cur=_data[i];
        const r=await linkPrompt({label:cur.label,url:cur.url,fav:cur.favicon||cur.fav});
        if(!r)return;
        if(r==="delete")_data.splice(i,1);
        else if(r.label&&r.url)_data[i]={label:r.label,url:r.url,favicon:r.fav||""};
        await save();load();
      });
      grp.appendChild(edit);a.appendChild(grp);container.appendChild(a);
    });
    enableDragSort(container,"a[data-drag-index]",()=>_data,(arr)=>{_data=arr;save();},load);
  }
  document.getElementById("qa-add-btn")?.addEventListener("click",async()=>{
    const r=await linkPrompt();if(!r||!r.label||!r.url)return;
    _data.push({label:r.label,url:r.url,favicon:r.fav||""});await save();load();
  });
  load();
})();
