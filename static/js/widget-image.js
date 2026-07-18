(function(){
  const btn=document.getElementById("widget-img-next");
  const wrap=document.getElementById("widget-img-wrap");
  if(!btn||!wrap)return;
  let currentFilename=wrap.querySelector("img")?.dataset.filename||"";
  let preloaded=null;
  function setImage(url,filename){
    const img=document.createElement("img");
    img.src=url;img.id="widget-img";img.className="active";img.dataset.filename=filename;
    wrap.innerHTML="";wrap.appendChild(img);currentFilename=filename;
  }
  function preloadNext(after){
    fetch("/api/widget-images/next?current="+encodeURIComponent(after))
      .then(r=>r.json()).then(d=>{if(!d.url)return;const i=new Image();i.src=d.url;preloaded={url:d.url,filename:d.filename};})
      .catch(()=>{});
  }
  btn.addEventListener("click",()=>{
    if(!preloaded){
      fetch("/api/widget-images/next?current="+encodeURIComponent(currentFilename))
        .then(r=>r.json()).then(d=>{if(!d.url)return;setImage(d.url,d.filename);preloadNext(d.filename);})
        .catch(()=>{});
      return;
    }
    setImage(preloaded.url,preloaded.filename);preloaded=null;preloadNext(currentFilename);
  });
  if(currentFilename)preloadNext(currentFilename);
})();
