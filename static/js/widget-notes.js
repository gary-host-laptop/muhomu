/* Notes widget — auto-saving textarea, saves via Store.set on input */
(function(){
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
})();
