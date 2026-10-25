/* Timer widget — countdown with start/pause/reset, plays alarm.mp3 */
(function(){
  const display=document.getElementById("timer-display");
  if(!display)return;
  const minInput=document.getElementById("timer-min");
  const secInput=document.getElementById("timer-sec");
  const startBtn=document.getElementById("timer-start");
  const resetBtn=document.getElementById("timer-reset");
  // ── alarm volume ─────────────────────────────────────────────
  // The "vol" row mirrors the rain widget's master slider (same markup),
  // and the value is persisted the same way, so the alarm is never louder
  // than the user last asked for.
  const STORAGE_KEY="nt_timer_volume";
  const volSlider=document.getElementById("timer-vol");
  let alarmVol=1;
  try{
    const raw=localStorage.getItem(STORAGE_KEY);
    if(raw!==null){
      const v=parseFloat(raw);
      if(!isNaN(v))alarmVol=Math.min(1,Math.max(0,v));
    }
  }catch(e){}
  if(volSlider)volSlider.value=String(alarmVol);
  let alarm=null;
  function getAlarm(){
    if(!alarm){
      alarm=new Audio("/static/assets/sounds/alarm.mp3");
      alarm.volume=alarmVol;
      alarm.addEventListener("ended",releaseAlarm);
    }
    return alarm;
  }
  // Drops the decoded alarm buffer as soon as it has rung (or is reset),
  // so the ~1 MB PCM doesn't linger for the rest of the session.
  function releaseAlarm(){
    if(!alarm)return;
    alarm.pause();
    alarm.removeAttribute("src");
    alarm.load();
    alarm=null;
  }
  let total=0,remaining=0,interval=null,running=false;
  function pad(n){return String(n).padStart(2,"0");}
  function render(secs){
    const m=Math.floor(secs/60),s=secs%60;
    display.textContent=pad(m)+":"+pad(s);
    display.classList.toggle("urgent",secs<=10&&secs>0);
  }
  function stop(){
    clearInterval(interval);interval=null;running=false;
    startBtn.querySelector("i").className="ph-light ph-play";
    startBtn.classList.remove("active");
    minInput.disabled=false;secInput.disabled=false;
  }
  startBtn.addEventListener("click",()=>{
    if(running){stop();return;}
    const m=parseInt(minInput.value)||0,s=parseInt(secInput.value)||0;
    total=m*60+s;if(total<=0)return;
    remaining=total;render(remaining);
    minInput.disabled=true;secInput.disabled=true;
    startBtn.querySelector("i").className="ph-light ph-pause";
    startBtn.classList.add("active");running=true;
    interval=setInterval(()=>{
      remaining--;render(remaining);
      if(remaining<=0){stop();getAlarm().currentTime=0;getAlarm().play();display.classList.add("urgent");}
    },1000);
  });
  resetBtn.addEventListener("click",()=>{
    stop();releaseAlarm();
    remaining=0;render(0);minInput.value=0;secInput.value=0;
    display.classList.remove("urgent");
  });
  const trashBtn=document.getElementById("timer-trash");
  if(trashBtn)trashBtn.addEventListener("click",()=>resetBtn.click());
  if(volSlider)volSlider.addEventListener("input",(e)=>{
    alarmVol=parseFloat(e.target.value);
    if(alarm)alarm.volume=alarmVol;
    try{localStorage.setItem(STORAGE_KEY,String(alarmVol));}catch(err){}
  });
})();
