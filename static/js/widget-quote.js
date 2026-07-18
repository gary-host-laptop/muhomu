(function(){
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
})();
