(function(){
  const canvas=document.getElementById("stats-canvas");
  if(!canvas)return;
  const ctx=canvas.getContext("2d");
  const history={cpu:[],mem:[]};
  const MAX=30;
  function syncSize(){
    const r=canvas.getBoundingClientRect();
    const w=Math.round(r.width),h=Math.round(r.height);
    if(canvas.width!==w||canvas.height!==h){canvas.width=w;canvas.height=h;}
  }
  const ro=new ResizeObserver(syncSize);ro.observe(canvas);syncSize();
  async function fetchStats(){try{const r=await fetch("/api/stats");if(!r.ok)return null;return r.json();}catch{return null;}}
  function push(arr,val){arr.push(val);if(arr.length>MAX)arr.shift();}
  function drawGraph(data,color,yOff,h){
    if(!data.length)return;
    const w=canvas.width,step=w/(MAX-1);
    ctx.beginPath();ctx.strokeStyle=color;ctx.lineWidth=1.5;
    data.forEach((v,i)=>{const x=i*step,y=yOff+h-(v/100)*h;i===0?ctx.moveTo(x,y):ctx.lineTo(x,y);});
    ctx.stroke();
    ctx.lineTo((data.length-1)*step,yOff+h);ctx.lineTo(0,yOff+h);ctx.closePath();
    ctx.fillStyle=color.replace(")"," , 0.08)").replace("rgb","rgba");ctx.fill();
  }
  function render(stats){
    const w=canvas.width,h=canvas.height;if(!w||!h)return;
    const style=getComputedStyle(document.documentElement);
    const dim=style.getPropertyValue("--dimmer").trim();
    const border=style.getPropertyValue("--border").trim();
    const white=style.getPropertyValue("--white").trim();
    ctx.clearRect(0,0,w,h);
    ctx.strokeStyle=border;ctx.lineWidth=0.5;
    for(let i=0;i<=4;i++){
      const y=Math.round((h/2)*(i/4)),y2=Math.round(h/2+(h/2)*(i/4));
      ctx.beginPath();ctx.moveTo(0,y);ctx.lineTo(w,y);ctx.stroke();
      ctx.beginPath();ctx.moveTo(0,y2);ctx.lineTo(w,y2);ctx.stroke();
    }
    drawGraph(history.cpu,"rgb(138,180,248)",0,h/2-4);
    drawGraph(history.mem,"rgb(144,216,112)",h/2+4,h/2-4);
    ctx.font="9px var(--font-pixel,monospace)";
    ctx.fillStyle=dim;ctx.fillText("CPU",4,12);ctx.fillText("MEM",4,h/2+16);
    if(stats){
      ctx.fillStyle=white;
      ctx.fillText(Math.round(stats.cpu||0)+"%",w-28,12);
      ctx.fillText(Math.round(stats.ram||0)+"%",w-28,h/2+16);
    }
    ctx.strokeStyle=border;ctx.lineWidth=1;
    ctx.beginPath();ctx.moveTo(0,h/2);ctx.lineTo(w,h/2);ctx.stroke();
  }
  async function update(isFirst){syncSize();const stats=await fetchStats();if(stats){if(isFirst&&stats.history&&stats.history.length){history.cpu=stats.history.map(p=>p.cpu);history.mem=stats.history.map(p=>p.ram);}else{push(history.cpu,stats.cpu||0);push(history.mem,stats.ram||0);}}render(stats);}
  update(true);setInterval(()=>update(false),2000);
})();
