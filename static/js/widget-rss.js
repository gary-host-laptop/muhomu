(function(){
  document.querySelectorAll(".rss-nav-btn").forEach(function(btn){
    btn.addEventListener("click",function(){
      var wrap=this.closest(".rss-body");
      if(!wrap)return;
      var container=wrap.querySelector(".rss-list, .rss-horizontal");
      if(!container)return;
      var pages=container.querySelectorAll(".rss-page");
      var current=container.querySelector(".rss-page.active");
      if(!current)return;
      var curIdx=parseInt(current.dataset.rssPage);
      var dir=this.dataset.rssDir;
      var nextIdx=dir==="next"?curIdx+1:curIdx-1;
      if(nextIdx<0||nextIdx>=pages.length)return;
      current.classList.remove("active");
      pages[nextIdx].classList.add("active");
      var ind=wrap.querySelector(".rss-page-indicator");
      if(ind)ind.textContent=(nextIdx+1)+" / "+pages.length;
      var prev=wrap.querySelector('[data-rss-dir="prev"]');
      if(prev)prev.disabled=(nextIdx===0);
    });
  });
})();
