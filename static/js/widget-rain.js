/* Rain sound player — pure Web Audio synthesis (no MP3 streaming).
   Layers:
     • rain wash   — 30 s seamless pink-noise loop (equal-power crossfade),
                     two detuned sources (stereo width + inaudible loop
                     point), a shared highpass (strips sub-bass boom) and one
                     lowpass per channel whose cutoff wanders via a
                     mean-reverting random walk (no long drifts or "swells").
     • droplets    — a scheduler sprays short band-passed noise grains with
                     random pan/rate/volume for the "patter" texture.
     • wind        — 30 s seamless brown-noise loop through a highpass +
                     slowly-wandering lowpass (a whoosh, not a sub-bass
                     drone).
     • thunder     — three real strike samples, played at random: each strike
                     is a throwaway <audio> element released as soon as it
                     ends, so at most one short sample (~1-4 MB decoded) is
                     ever resident — never a whole long file.
   On stop the AudioContext is closed — footprint while playing is ~14 MB of
   generated buffers plus one thunder strike; ~0 when stopped.        */
(function() {
  var STORAGE_KEY = "nt_rain_volumes";
  var masterVol = 1;
  var playing = false;

  var ctx = null;                 // AudioContext — created in the play gesture
  var masterGain = null;
  var rainGain = null;
  var windGain = null;
  var grainFilter = null;
  var grainBufs = [];

  // filters kept at module scope for the random-walk modulation
  var washFilterL = null, washFilterR = null, windFilter = null;
  var washTarget = 5200, windTarget = 320;
  var lastWashStep = -1, lastWindStep = -1;

  var THUNDER_FILES = [
    "/static/assets/sounds/thunder-1.mp3",
    "/static/assets/sounds/thunder-2.mp3",
    "/static/assets/sounds/thunder-3.mp3"
  ];
  var currentThunder = null;

  var thunderTimer = null;
  var strikeReleaseTimer = null; // per-strike safety release
  var schedulerTimer = null;
  var nextGrainTime = 0;

  // ── volume persistence (unchanged semantics) ──────────────────
  function saveVolumes() {
    var data = { master: masterVol };
    ["rain", "wind", "thunder"].forEach(function(k) {
      data[k] = parseFloat(document.getElementById("vol-" + k).value);
    });
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(data)); } catch (e) {}
  }

  function loadVolumes() {
    var raw;
    try { raw = localStorage.getItem(STORAGE_KEY); } catch (e) {}
    if (!raw) return;
    var data;
    try { data = JSON.parse(raw); } catch (e) { return; }
    if (data.master !== undefined) {
      masterVol = data.master;
      var mEl = document.getElementById("vol-master");
      if (mEl) mEl.value = data.master;
    }
    ["rain", "wind", "thunder"].forEach(function(k) {
      if (data[k] === undefined) return;
      var el = document.getElementById("vol-" + k);
      if (el) el.value = data[k];
    });
  }
  loadVolumes();

  // ── offline buffer generators (run once per context) ──────────
  function normalize(out, peak) {
    var sum = 0, max = 0, i;
    for (i = 0; i < out.length; i++) {
      sum += out[i];
      var a = Math.abs(out[i]);
      if (a > max) max = a;
    }
    var mean = sum / out.length;
    var scale = max > 0 ? peak / max : 1;
    for (i = 0; i < out.length; i++) out[i] = (out[i] - mean) * scale;
    return out;
  }

  // pink noise — Paul Kellet's filter approximation (1/f spectrum,
  // the natural "wash" slope of rain), DC-removed and normalized.
  function pinkNoise(len) {
    var out = new Float32Array(len);
    var b0 = 0, b1 = 0, b2 = 0, b3 = 0, b4 = 0, b5 = 0, b6 = 0;
    for (var n = 0; n < len; n++) {
      var w = Math.random() * 2 - 1;
      b0 = 0.99886 * b0 + w * 0.0555179;
      b1 = 0.99332 * b1 + w * 0.0750759;
      b2 = 0.96900 * b2 + w * 0.1538520;
      b3 = 0.86650 * b3 + w * 0.3104856;
      b4 = 0.55000 * b4 + w * 0.5329522;
      b5 = -0.7616 * b5 - w * 0.0168980;
      out[n] = (b0 + b1 + b2 + b3 + b4 + b5 + b6 + w * 0.5362) * 0.11;
      b6 = w * 0.115926;
    }
    return normalize(out, 0.8);
  }

  // brown noise — integrated white noise (deep rumble for wind).
  function brownNoise(len) {
    var out = new Float32Array(len);
    var last = 0;
    for (var n = 0; n < len; n++) {
      last = (last + 0.02 * (Math.random() * 2 - 1)) / 1.02;
      out[n] = last;
    }
    return normalize(out, 0.9);
  }

  // equal-power crossfade of the loop tail into the head: the seam is
  // click-free AND the level stays constant through the crossfade (a plain
  // power crossfade would dip ~3 dB for the first 10% of the loop). The
  // fade is kept short (5%) because the wrap jump scales with its length
  // for low-frequency-heavy noise (measured: brown 4.1x -> 1.4x).
  function seamlessLoop(d, len) {
    var fade = Math.floor(len * 0.05);
    if (fade <= 0 || fade >= len) return;
    for (var i = 0; i < fade; i++) {
      var th = (i / fade) * (Math.PI / 2);
      d[i] = d[i] * Math.sin(th) + d[len - fade + i] * Math.cos(th);
    }
  }

  // a single droplet grain: ~70 ms of noise with a baked attack/decay
  // envelope so grains never click regardless of scheduling.
  function makeGrain(ctx) {
    var len = Math.floor(ctx.sampleRate * 0.07);
    var buf = ctx.createBuffer(1, len, ctx.sampleRate);
    var d = buf.getChannelData(0);
    for (var i = 0; i < len; i++) d[i] = Math.random() * 2 - 1;
    var atk = Math.floor(len * 0.06);
    for (i = 0; i < atk; i++) d[i] *= i / atk;
    for (i = atk; i < len; i++) d[i] *= 1 - (i - atk) / (len - atk);
    return buf;
  }

  function makePan(ctx, value) {
    if (!ctx.createStereoPanner) return null; // old browsers: just mono
    var p = ctx.createStereoPanner();
    p.pan.value = value;
    return p;
  }

  function makeBiquad(ctx, type, f0, Q) {
    var f = ctx.createBiquadFilter();
    f.type = type;
    f.frequency.value = f0;
    f.Q.value = Q;
    return f;
  }

  function route(src, pan, target) {
    if (pan) { src.connect(pan); pan.connect(target); }
    else { src.connect(target); }
  }

  // ── build the audio graph (must run inside a user gesture) ────
  function ensureAudio() {
    if (ctx) {
      if (ctx.state === "suspended") ctx.resume();
      return;
    }
    var AC = window.AudioContext || window.webkitAudioContext;
    if (!AC) return; // no Web Audio — button still toggles, silently
    ctx = new AC();
    if (ctx.state === "suspended") ctx.resume();

    masterGain = ctx.createGain();
    masterGain.gain.value = 1; // unity bus; per-track gains carry volumes
    masterGain.connect(ctx.destination);

    rainGain = ctx.createGain();
    rainGain.gain.value = parseFloat(document.getElementById("vol-rain").value) * masterVol;
    rainGain.connect(masterGain);

    windGain = ctx.createGain();
    windGain.gain.value = parseFloat(document.getElementById("vol-wind").value) * masterVol;
    windGain.connect(masterGain);

    // ── rain wash ──
    var washLen = Math.floor(ctx.sampleRate * 30); // 30 s
    var wash = ctx.createBuffer(1, washLen, ctx.sampleRate);
    var wd = wash.getChannelData(0);
    wd.set(pinkNoise(washLen));
    seamlessLoop(wd, washLen);

    // one lowpass per channel so the two detuned copies actually
    // decorrelate L/R (a shared filter would mix them back together); a
    // shared highpass strips the pink-noise sub-bass boom (the "drone")
    washFilterL = makeBiquad(ctx, "lowpass", 5200, 0.5);
    washFilterR = makeBiquad(ctx, "lowpass", 5200, 0.5);
    var washHP = makeBiquad(ctx, "highpass", 180, 0.707);

    var a = ctx.createBufferSource();
    a.buffer = wash; a.loop = true; a.playbackRate.value = 1.0;
    a.connect(washHP);
    washHP.connect(washFilterL);
    route(washFilterL, makePan(ctx, -0.55), rainGain);
    a.start();
    var b = ctx.createBufferSource();
    b.buffer = wash; b.loop = true; b.playbackRate.value = 1.0015; // detune
    b.connect(washHP);
    washHP.connect(washFilterR);
    route(washFilterR, makePan(ctx, 0.55), rainGain);
    // start at a random buffer offset so the two copies are already
    // decorrelated at t=0 (otherwise they begin in phase = +3 dB crest)
    b.start(0, Math.random() * (washLen / ctx.sampleRate));

    // ── wind ──
    var windLen = Math.floor(ctx.sampleRate * 30);
    var wind = ctx.createBuffer(1, windLen, ctx.sampleRate);
    var wnd = wind.getChannelData(0);
    wnd.set(brownNoise(windLen));
    seamlessLoop(wnd, windLen);

    windFilter = makeBiquad(ctx, "lowpass", 320, 0.7);
    // highpass removes the brown noise's sub-100 Hz rumble, turning the
    // wind from a sub-bass drone into an audible whoosh (120-500 Hz)
    var windHP = makeBiquad(ctx, "highpass", 120, 0.707);
    var ws = ctx.createBufferSource();
    ws.buffer = wind; ws.loop = true;
    ws.connect(windHP);
    windHP.connect(windFilter);
    windFilter.connect(windGain);
    ws.start();

    // ── droplet grains ──
    grainFilter = ctx.createBiquadFilter();
    grainFilter.type = "bandpass";
    grainFilter.frequency.value = 3000; // the "pitter" band
    grainFilter.Q.value = 0.9;
    grainFilter.connect(rainGain);
    grainBufs = [];
    for (var g = 0; g < 16; g++) grainBufs.push(makeGrain(ctx));

    lastWashStep = ctx.currentTime;
    lastWindStep = ctx.currentTime;
  }

  // ── mean-reverting cutoff modulation (non-repeating by design) ──
  // A plain random walk drifts: over tens of seconds the cutoff can crawl
  // steadily up or down, which sounds like a slow swell/crescendo. Pulling
  // each step back toward the center keeps it wandering without drifting.
  function modulate() {
    var now = ctx.currentTime;
    if (washFilterL && now - lastWashStep > 2 + Math.random() * 2) {
      lastWashStep = now;
      washTarget = 5200 + (washTarget - 5200) * 0.85 + (Math.random() * 2 - 1) * 900;
      washTarget = Math.max(3600, Math.min(7200, washTarget));
      var t1 = now + 2 + Math.random() * 1.5;
      washFilterL.frequency.linearRampToValueAtTime(washTarget, t1);
      washFilterR.frequency.linearRampToValueAtTime(washTarget, t1);
    }
    if (windFilter && now - lastWindStep > 3 + Math.random() * 2) {
      lastWindStep = now;
      windTarget = 320 + (windTarget - 320) * 0.85 + (Math.random() * 2 - 1) * 80;
      windTarget = Math.max(200, Math.min(520, windTarget));
      var t2 = now + 3 + Math.random() * 2;
      windFilter.frequency.linearRampToValueAtTime(windTarget, t2);
    }
  }

  // ── droplet scheduler (lookahead: schedule 250 ms ahead) ──────
  function scheduleGrain(t) {
    var buf = grainBufs[(Math.random() * grainBufs.length) | 0];
    var src = ctx.createBufferSource();
    src.buffer = buf;
    src.playbackRate.value = 0.85 + Math.random() * 0.4; // drop size variety
    var g = ctx.createGain();
    var peak = 0.12 + Math.random() * 0.28;
    g.gain.setValueAtTime(0, t);
    g.gain.linearRampToValueAtTime(peak, t + 0.004);
    g.gain.exponentialRampToValueAtTime(0.001, t + 0.045 + Math.random() * 0.06);
    var pan = makePan(ctx, Math.random() * 1.5 - 0.75);
    route(src, pan, g);
    route(g, null, grainFilter);
    // Web Audio nodes connected into the graph are NOT garbage-collected
    // until disconnected: without this teardown, each finished grain (3
    // nodes, ~30/s) stays alive for the whole session — the slow creep to
    // ~150 MB. Tear the chain down the moment playback ends.
    src.onended = function() {
      src.disconnect();
      g.disconnect();
      if (pan) pan.disconnect();
    };
    src.start(t);
    src.stop(t + 0.15);
  }

  function startScheduler() {
    nextGrainTime = ctx.currentTime + 0.15;
    schedulerTimer = setInterval(function() {
      if (!playing) return;
      while (nextGrainTime < ctx.currentTime + 0.25) {
        scheduleGrain(nextGrainTime);
        nextGrainTime += 0.022 + Math.random() * 0.03; // 22–52 ms apart
      }
      modulate();
    }, 80);
  }

  // ── thunder: random short strike, released as soon as it ends ─
  function playThunder() {
    releaseThunder(); // drop any previous strike's decoded buffer
    var file = THUNDER_FILES[(Math.random() * THUNDER_FILES.length) | 0];
    var t = new Audio(file);
    currentThunder = t;
    t._strikeReleased = false;
    t.volume = parseFloat(document.getElementById("vol-thunder").value) * masterVol;
    function release() {
      if (t._strikeReleased) return;
      t._strikeReleased = true;
      if (currentThunder === t) currentThunder = null;
      t.pause();
      t.removeAttribute("src");
      t.load();
    }
    t.addEventListener("ended", release);
    t.addEventListener("error", release);
    var p = t.play();
    if (p && p.catch) p.catch(release);
    // safety net: never keep a decoded strike resident past ~15 s
    if (strikeReleaseTimer) clearTimeout(strikeReleaseTimer);
    strikeReleaseTimer = setTimeout(release, 15000);
  }

  function releaseThunder() {
    if (!currentThunder) return;
    var t = currentThunder;
    currentThunder = null;
    t._strikeReleased = true;
    t.pause();
    t.removeAttribute("src");
    t.load();
  }

  function scheduleThunder() {
    thunderTimer = setTimeout(function() {
      if (!playing) return;
      playThunder();
      scheduleThunder();
    }, 20000 + Math.random() * 40000);
  }

  // ── stop: close the context, free everything ──────────────────
  function stopAudio() {
    clearTimeout(thunderTimer);
    clearTimeout(strikeReleaseTimer);
    clearInterval(schedulerTimer);
    releaseThunder();
    if (ctx) {
      ctx.close().catch(function() {});
      ctx = null;
      masterGain = null;
      rainGain = null;
      windGain = null;
      grainFilter = null;
      washFilterL = null;
      washFilterR = null;
      windFilter = null;
      grainBufs = [];
    }
  }

  // ── UI ────────────────────────────────────────────────────────
  var btn = document.getElementById("rain-btn");
  btn.addEventListener("click", function() {
    if (!playing) {
      ensureAudio();
      startScheduler();
      scheduleThunder();
      btn.querySelector("i").className = "ph-light ph-stop";
      btn.classList.add("playing");
      playing = true;
    } else {
      stopAudio();
      btn.querySelector("i").className = "ph-light ph-play";
      btn.classList.remove("playing");
      playing = false;
    }
  });

  document.getElementById("vol-master").addEventListener("input", function(e) {
    masterVol = parseFloat(e.target.value);
    if (rainGain) rainGain.gain.value = parseFloat(document.getElementById("vol-rain").value) * masterVol;
    if (windGain) windGain.gain.value = parseFloat(document.getElementById("vol-wind").value) * masterVol;
    saveVolumes();
  });

  ["vol-rain", "vol-wind", "vol-thunder"].forEach(function(id) {
    document.getElementById(id).addEventListener("input", function(e) {
      if (id === "vol-rain" && rainGain) rainGain.gain.value = parseFloat(e.target.value) * masterVol;
      if (id === "vol-wind" && windGain) windGain.gain.value = parseFloat(e.target.value) * masterVol;
      // thunder volume is applied per-strike in playThunder
      saveVolumes();
    });
  });
})();
