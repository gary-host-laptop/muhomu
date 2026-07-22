/* Rain sound player — volume sliders for rain, wind, thunder tracks */
(function(){
  if(!document.getElementById("vol-rain"))return;
  var STORAGE_KEY="nt_rain_volumes";
  var tracks={
    rain:{file:"/static/assets/sounds/heavy-rain.mp3",loop:true,id:"vol-rain",audio:null},
    wind:{file:"/static/assets/sounds/wind.mp3",loop:true,id:"vol-wind",audio:null},
    thunder:{file:"/static/assets/sounds/thunder.mp3",loop:false,id:"vol-thunder",audio:null},
  };
  function initAudio(){Object.values(tracks).forEach(t=>{
    if(t.audio)return;
    t.audio=new Audio(t.file);t.audio.loop=t.loop;
    t.audio.volume=parseFloat(document.getElementById(t.id).value)*masterVol;
  });}
  var playing=false,thunderTimer=null;
  function scheduleThunder(){
    thunderTimer=setTimeout(function(){
      if(!playing)return;
      var a=tracks.thunder.audio;a.currentTime=0;a.play();scheduleThunder();
    },20000+Math.random()*40000);
  }
  function saveVolumes(){
    var data={master:masterVol};
    Object.keys(tracks).forEach(function(k){data[k]=parseFloat(document.getElementById(tracks[k].id).value);});
    try{localStorage.setItem(STORAGE_KEY,JSON.stringify(data));}catch(e){}
  }
  function loadVolumes(){
    var raw;
    try{raw=localStorage.getItem(STORAGE_KEY);}catch(e){}
    if(!raw)return;
    var data;
    try{data=JSON.parse(raw);}catch(e){return;}
    if(data.master!==undefined){
      masterVol=data.master;
      var mEl=document.getElementById("vol-master");
      if(mEl)mEl.value=data.master;
    }
    Object.keys(tracks).forEach(function(k){
      if(data[k]===undefined)return;
      var el=document.getElementById(tracks[k].id);
      if(el)el.value=data[k];
    });
  }
  var masterVol=1;
  loadVolumes();
  var btn=document.getElementById("rain-btn");
  btn.addEventListener("click",function(){
    initAudio();
    if(!playing){
      Promise.all([tracks.rain.audio.play(),tracks.wind.audio.play()])
        .then(function(){scheduleThunder();btn.querySelector("i").className="ph-light ph-stop";btn.classList.add("playing");playing=true;})
        .catch(function(){btn.querySelector("i").className="ph-light ph-stop";btn.classList.add("playing");playing=true;});
    }else{
      tracks.rain.audio.pause();tracks.wind.audio.pause();tracks.thunder.audio.pause();
      clearTimeout(thunderTimer);btn.querySelector("i").className="ph-light ph-play";btn.classList.remove("playing");playing=false;
    }
  });
  document.getElementById("vol-master")?.addEventListener("input",function(e){
    masterVol=parseFloat(e.target.value);
    Object.values(tracks).forEach(function(t){if(!t.audio)return;t.audio.volume=parseFloat(document.getElementById(t.id).value)*masterVol;});
    saveVolumes();
  });
  Object.values(tracks).forEach(function(t){
    document.getElementById(t.id).addEventListener("input",function(e){if(!t.audio)return;t.audio.volume=parseFloat(e.target.value)*masterVol;saveVolumes();});
  });
})();
