/* Recently visited — list of saved links with quick-add form */
(function(){
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
})();
