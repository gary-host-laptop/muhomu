(function(){
  if(!document.getElementById("vol-rain"))return;
  const tracks={
    rain:{file:"/static/assets/sounds/heavy-rain.mp3",loop:true,id:"vol-rain",audio:null},
    wind:{file:"/static/assets/sounds/wind.mp3",loop:true,id:"vol-wind",audio:null},
    thunder:{file:"/static/assets/sounds/thunder.mp3",loop:false,id:"vol-thunder",audio:null},
  };
  function initAudio(){Object.values(tracks).forEach(t=>{
    if(t.audio)return;
    t.audio=new Audio(t.file);t.audio.loop=t.loop;
    t.audio.volume=parseFloat(document.getElementById(t.id).value);
  });}
  let playing=false,thunderTimer=null;
  function scheduleThunder(){
    thunderTimer=setTimeout(()=>{
      if(!playing)return;
      const a=tracks.thunder.audio;a.currentTime=0;a.play();scheduleThunder();
    },20000+Math.random()*40000);
  }
  const btn=document.getElementById("rain-btn");
  btn.addEventListener("click",()=>{
    initAudio();
    if(!playing){
      Promise.all([tracks.rain.audio.play(),tracks.wind.audio.play()])
        .then(()=>{scheduleThunder();btn.querySelector("i").className="ph-light ph-stop";btn.classList.add("playing");playing=true;})
        .catch(()=>{btn.querySelector("i").className="ph-light ph-stop";btn.classList.add("playing");playing=true;});
    }else{
      tracks.rain.audio.pause();tracks.wind.audio.pause();tracks.thunder.audio.pause();
      clearTimeout(thunderTimer);btn.querySelector("i").className="ph-light ph-play";btn.classList.remove("playing");playing=false;
    }
  });
  let masterVol=1;
  document.getElementById("vol-master")?.addEventListener("input",e=>{
    masterVol=parseFloat(e.target.value);
    Object.values(tracks).forEach(t=>{if(!t.audio)return;t.audio.volume=parseFloat(document.getElementById(t.id).value)*masterVol;});
  });
  Object.values(tracks).forEach(t=>{
    document.getElementById(t.id).addEventListener("input",e=>{if(!t.audio)return;t.audio.volume=parseFloat(e.target.value)*masterVol;});
  });
})();
