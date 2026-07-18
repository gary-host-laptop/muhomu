(function(){
  const charEl=document.getElementById("kanji-char");
  const readEl=document.getElementById("kanji-reading");
  const meanEl=document.getElementById("kanji-meaning");
  const levelEl=document.getElementById("kanji-level");
  if(!charEl)return;
  function renderWord(w){
    charEl.innerHTML="";
    const a=document.createElement("a");
    a.href="https://jisho.org/search/"+encodeURIComponent(w.k)+"%20%23kanji";
    a.style.cssText="color:inherit;text-decoration:none;";a.textContent=w.k;
    charEl.appendChild(a);
    if(readEl)readEl.textContent=w.r;
    if(meanEl)meanEl.textContent=w.m;
    if(levelEl){levelEl.textContent=w.l.toUpperCase();levelEl.className="kanji-level "+w.l;}
  }
  document.getElementById("kanji-next")?.addEventListener("click",async()=>{
    try{
      const r=await fetch("/api/kotoba/next");
      const w=await r.json();
      renderWord(w);
    }catch(e){}
  });
})();
