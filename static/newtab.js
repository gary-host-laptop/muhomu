/* ── TIMER ───────────────────────────────────────────────────── */
(function () {
  const display = document.getElementById("timer-display");
  if (!display) return; // widget not in DOM
  const minInput = document.getElementById("timer-min");
  const secInput = document.getElementById("timer-sec");
  const startBtn = document.getElementById("timer-start");
  const resetBtn = document.getElementById("timer-reset");
  // Lazy — only created when timer actually fires, not on page load.
  let alarm = null;
  function getAlarm() {
    if (!alarm) alarm = new Audio("/static/assets/sounds/alarm.mp3");
    return alarm;
  }

  let total = 0;
  let remaining = 0;
  let interval = null;
  let running = false;

  function pad(n) {
    return String(n).padStart(2, "0");
  }

  function render(secs) {
    const m = Math.floor(secs / 60);
    const s = secs % 60;
    display.textContent = `${pad(m)}:${pad(s)}`;
    display.classList.toggle("urgent", secs <= 10 && secs > 0);
  }

  function stop() {
    clearInterval(interval);
    interval = null;
    running = false;
    startBtn.querySelector("i").className = "ph-light ph-play";
    startBtn.classList.remove("active");
    minInput.disabled = false;
    secInput.disabled = false;
  }

  startBtn.addEventListener("click", () => {
    if (running) {
      stop();
      return;
    }
    const m = parseInt(minInput.value) || 0;
    const s = parseInt(secInput.value) || 0;
    total = m * 60 + s;
    if (total <= 0) return;
    remaining = total;
    render(remaining);
    minInput.disabled = true;
    secInput.disabled = true;
    startBtn.querySelector("i").className = "ph-light ph-pause";
    startBtn.classList.add("active");
    running = true;
    interval = setInterval(() => {
      remaining--;
      render(remaining);
      if (remaining <= 0) {
        stop();
        getAlarm().currentTime = 0;
        getAlarm().play();
        display.classList.add("urgent");
      }
    }, 1000);
  });

  resetBtn.addEventListener("click", () => {
    stop();
    if (alarm) { alarm.pause(); alarm.currentTime = 0; }
    remaining = 0;
    render(0);
    minInput.value = 0;
    secInput.value = 0;
    display.classList.remove("urgent");
  });

  // timer-trash button triggers the same reset logic
  const trashBtn = document.getElementById("timer-trash");
  if (trashBtn) trashBtn.addEventListener("click", () => resetBtn.click());
})();

/* ── RAIN PLAYER ────────────────────────────────────────────── */
(function () {
  if (!document.getElementById("vol-rain")) return; // widget not in DOM
  const tracks = {
    rain:    { file: "/static/assets/sounds/heavy-rain.mp3", loop: true,  id: "vol-rain",    audio: null },
    wind:    { file: "/static/assets/sounds/wind.mp3",       loop: true,  id: "vol-wind",    audio: null },
    thunder: { file: "/static/assets/sounds/thunder.mp3",    loop: false, id: "vol-thunder", audio: null },
  };

  // Lazy init — only create Audio objects on first play,
  // not on page load. Avoids buffering 3 audio files into
  // memory for users who never use the rain player.
  function initAudio() {
    Object.values(tracks).forEach((t) => {
      if (t.audio) return;
      t.audio = new Audio(t.file);
      t.audio.loop = t.loop;
      t.audio.volume = parseFloat(document.getElementById(t.id).value);
    });
  }

  let playing = false;
  let thunderTimer = null;

  function scheduleThunder() {
    const delay = 20000 + Math.random() * 40000;
    thunderTimer = setTimeout(() => {
      if (!playing) return;
      const a = tracks.thunder.audio;
      a.currentTime = 0;
      a.play();
      scheduleThunder();
    }, delay);
  }

  const btn = document.getElementById("rain-btn");

  btn.addEventListener("click", () => {
    initAudio(); // create Audio objects only now if not already done
    if (!playing) {
      const playRain = tracks.rain.audio.play();
      const playWind = tracks.wind.audio.play();
      Promise.all([playRain, playWind])
        .then(() => {
          scheduleThunder();
          btn.querySelector("i").className = "ph-light ph-stop";
          btn.classList.add("playing");
          playing = true;
        })
        .catch(() => {
          btn.querySelector("i").className = "ph-light ph-stop";
          btn.classList.add("playing");
          playing = true;
        });
    } else {
      tracks.rain.audio.pause();
      tracks.wind.audio.pause();
      tracks.thunder.audio.pause();
      clearTimeout(thunderTimer);
      btn.querySelector("i").className = "ph-light ph-play";
      btn.classList.remove("playing");
      playing = false;
    }
  });

  let masterVol = 1;
  document.getElementById("vol-master")?.addEventListener("input", (e) => {
    masterVol = parseFloat(e.target.value);
    Object.values(tracks).forEach((t) => {
      if (!t.audio) return;
      const trackVol = parseFloat(document.getElementById(t.id).value);
      t.audio.volume = trackVol * masterVol;
    });
  });

  Object.entries(tracks).forEach(([, t]) => {
    document.getElementById(t.id).addEventListener("input", (e) => {
      if (!t.audio) return;
      t.audio.volume = parseFloat(e.target.value) * masterVol;
    });
  });
})();

/* ── PROFILE IMAGE is set server-side in the template ─────────── */

/* ── CLOCK ───────────────────────────────────────────────────── */
function pad(n) {
  return String(n).padStart(2, "0");
}
const DAYS = [
  "Sunday",
  "Monday",
  "Tuesday",
  "Wednesday",
  "Thursday",
  "Friday",
  "Saturday",
];
const MONTHS = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];

/* getGreetingStrings() is provided by i18n.js */

function getWeekNum(d) {
  const onejan = new Date(d.getFullYear(), 0, 1);
  return Math.ceil(((d - onejan) / 86400000 + onejan.getDay() + 1) / 7);
}

function getYearProgress(d) {
  const start = new Date(d.getFullYear(), 0, 0);
  const end = new Date(d.getFullYear() + 1, 0, 0);
  const pct = (((d - start) / (end - start)) * 100).toFixed(1);
  return pct + "%";
}

let greetingSet = null;
let lastHour = -1;

function setGreeting(el, text) {
  if (!el) return;
  el.textContent = text;
  const accents = ["--accent", "--accent2", "--accent3", "--red"];
  const chosen = accents[Math.floor(Math.random() * accents.length)];
  const val = `var(${chosen})`;
  el.style.color = val;
  const usernameEl = document.getElementById("username");
  if (usernameEl) usernameEl.style.color = val;
  document.documentElement.style.setProperty("--greeting-accent", val);
}

let binaryClock = false;

/* ── DRAG AND DROP SORT ─────────────────────────────────────── */
/* getData/setData are sync functions operating on in-memory arrays. */
function enableDragSort(container, itemSelector, getData, setData, reload) {
  if (!container) return;
  let dragSrc = null;
  container.addEventListener("dragstart", (e) => {
    if (!document.body.classList.contains("edit-mode")) return;
    const item = e.target.closest(itemSelector);
    if (!item) return;
    dragSrc = item;
    item.classList.add("dragging");
    e.dataTransfer.effectAllowed = "move";
  });
  container.addEventListener("dragover", (e) => {
    if (!dragSrc) return;
    const item = e.target.closest(itemSelector);
    if (!item || item === dragSrc) return;
    e.preventDefault();
    container
      .querySelectorAll(itemSelector)
      .forEach((el) => el.classList.remove("drag-over"));
    item.classList.add("drag-over");
  });
  container.addEventListener("dragleave", (e) => {
    if (!e.relatedTarget || !container.contains(e.relatedTarget)) {
      container
        .querySelectorAll(itemSelector)
        .forEach((el) => el.classList.remove("drag-over"));
    }
  });
  container.addEventListener("drop", (e) => {
    e.preventDefault();
    const target = e.target.closest(itemSelector);
    if (!target || !dragSrc || target === dragSrc) return;
    const srcIdx = parseInt(dragSrc.dataset.dragIndex);
    const tgtIdx = parseInt(target.dataset.dragIndex);
    const arr = getData();
    const [moved] = arr.splice(srcIdx, 1);
    arr.splice(tgtIdx, 0, moved);
    setData(arr);
    dragSrc = null;
    reload();
  });
  container.addEventListener("dragend", () => {
    container
      .querySelectorAll(itemSelector)
      .forEach((el) => el.classList.remove("dragging", "drag-over"));
    dragSrc = null;
  });
}

function renderBinaryGroup(val) {
  // 6 bits: top row = bits 5,4 | bottom row = bits 3,2,1,0
  const group = document.createElement("div");
  group.className = "bin-group";
  const bits = [];
  for (let i = 5; i >= 0; i--) bits.push((val >> i) & 1);
  // Row 1: bits[0], bits[1], then 2 empty spacers
  bits.slice(0, 2).forEach((b) => {
    const d = document.createElement("div");
    d.className = "bin-bit" + (b ? " on" : "");
    group.appendChild(d);
  });
  // 2 empty cells to complete row 1
  for (let i = 0; i < 2; i++) {
    const e = document.createElement("div");
    e.className = "bin-empty";
    group.appendChild(e);
  }
  // Row 2: bits[2..5]
  bits.slice(2).forEach((b) => {
    const d = document.createElement("div");
    d.className = "bin-bit" + (b ? " on" : "");
    group.appendChild(d);
  });
  return group;
}

function renderBinary(h, m, s) {
  // h should always be 24h (0-23) — binary clocks don't use 12h
  // s === null means hide seconds
  const el = document.getElementById("clock");
  el.innerHTML = "";

  el.appendChild(renderBinaryGroup(h));

  const sep1 = document.createElement("span");
  sep1.className = "bin-sep";
  sep1.textContent = ":";
  el.appendChild(sep1);

  el.appendChild(renderBinaryGroup(m));

  if (s !== null) {
    const sep2 = document.createElement("span");
    sep2.className = "bin-sep";
    sep2.textContent = ":";
    el.appendChild(sep2);
    el.appendChild(renderBinaryGroup(s));
  }
}

document.getElementById("clock").addEventListener("click", () => {
  binaryClock = !binaryClock;
  const el = document.getElementById("clock");
  el.classList.toggle("binary", binaryClock);
  if (!binaryClock) el.innerHTML = "";
});

// Read clock settings from body data attributes (set by server in template).
// Falls back to 24h / show seconds if attributes are absent.
window._clockFormat  = document.body.dataset.clockFormat  || "24h";
window._clockSeconds = document.body.dataset.clockSeconds !== "false";
window._uiLang       = document.body.dataset.uiLang       || "en";

function tick() {
  const now = new Date();
  const rawH = now.getHours();
  const m = now.getMinutes();
  const s = now.getSeconds();

  if (binaryClock) {
    // Binary always uses 24h raw hours; pass null for s when hiding seconds
    renderBinary(rawH, m, window._clockSeconds ? s : null);
  } else {
    let h = rawH;
    let suffix = "";
    if (window._clockFormat === "12h") {
      suffix = h >= 12 ? " PM" : " AM";
      h = h % 12 || 12;
    }
    document.getElementById("clock").textContent = window._clockSeconds
      ? `${pad(h)}:${pad(m)}:${pad(s)}${suffix}`
      : `${pad(h)}:${pad(m)}${suffix}`;
  }

  if (rawH !== lastHour) {
    lastHour = rawH;
    greetingSet = getGreetingStrings(rawH, window._uiLang || "en");
    setGreeting(
      document.getElementById("greeting"),
      greetingSet[Math.floor(Math.random() * greetingSet.length)],
    );

    document.getElementById("date").textContent =
      `${DAYS[now.getDay()].toUpperCase()} · ${MONTHS[now.getMonth()].toUpperCase()} ${now.getDate()} · ${now.getFullYear()}`;

    const weekEl = document.getElementById("week-num");
    const yearEl = document.getElementById("year-progress");
    if (weekEl) weekEl.textContent = `W${pad(getWeekNum(now))}`;
    if (yearEl) yearEl.textContent = `y · ${Math.round((now - new Date(now.getFullYear(),0,0)) / (new Date(now.getFullYear()+1,0,0) - new Date(now.getFullYear(),0,0)) * 100)}%`;
  }
}
window._tickInterval = setInterval(tick, 1000);
tick();

/* ── SEARCH ─────────────────────────────────────────────────── */
let activeEngine = document.querySelector(".engine-btn.active");

document.querySelectorAll(".engine-btn").forEach((btn) => {
  btn.addEventListener("click", () => {
    document
      .querySelectorAll(".engine-btn")
      .forEach((b) => b.classList.remove("active"));
    btn.classList.add("active");
    activeEngine = btn;
  });
});

document.getElementById("search-input").addEventListener("keydown", (e) => {
  if (e.key === "Enter") {
    const q = e.target.value.trim();
    if (!q) return;
    const url = activeEngine.dataset.url + encodeURIComponent(q);
    window.open(url, "_blank");
    e.target.value = "";
  }
});

/* ── STORAGE HELPER ─────────────────────────────────────────────
   Write-only from the client. The server renders all initial state.
   JS only posts mutations; the next page load reflects them via SSR. */
const Store = {
  async set(key, val) {
    try {
      await fetch("/api/data", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ [key]: val }),
      });
    } catch {
      localStorage.setItem(key, JSON.stringify(val));
    }
  },
};

/* ── NOTES ──────────────────────────────────────────────────── */
const notesEl = document.getElementById("notes");
if (notesEl) {
  let notesSaveTimer = null;
  notesEl.addEventListener("input", () => {
    clearTimeout(notesSaveTimer);
    notesSaveTimer = setTimeout(() => Store.set("nt_notes", notesEl.value), 800);
  });
  document.getElementById("notes-clear")?.addEventListener("click", async () => {
    if (confirm("clear notes?")) {
      clearTimeout(notesSaveTimer);
      notesEl.value = "";
      await Store.set("nt_notes", "");
    }
  });
  // Flush any pending save on tab close. fetch() is cancelled during
  // unload so sendBeacon is used — it queues the request at the browser
  // level and guarantees delivery even as the page tears down.
  window.addEventListener("beforeunload", () => {
    if (!notesSaveTimer) return;
    clearTimeout(notesSaveTimer);
    notesSaveTimer = null;
    navigator.sendBeacon(
      "/api/data",
      new Blob([JSON.stringify({ nt_notes: notesEl.value })], { type: "application/json" }),
    );
  });
}

/* ── CONTEXT MENU ───────────────────────────────────────────────
   Usage: CtxMenu.show(e, [ {label, action, danger?} ])           */
const CtxMenu = (() => {
  const el = document.getElementById("ctx-menu");

  document.addEventListener("click", () => hide());
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") hide();
  });

  function hide() {
    el.classList.remove("visible");
    el.innerHTML = "";
  }

  function show(e, items) {
    e.preventDefault();
    e.stopPropagation();
    hide();
    items.forEach(({ label, action, danger }) => {
      const div = document.createElement("div");
      div.className = "ctx-item" + (danger ? " danger" : "");
      div.textContent = label;
      div.addEventListener("click", () => {
        hide();
        action();
      });
      el.appendChild(div);
    });
    el.classList.add("visible");
    // position, keep within viewport
    const vw = window.innerWidth,
      vh = window.innerHeight;
    let x = e.clientX,
      y = e.clientY;
    el.style.left = x + "px";
    el.style.top = y + "px";
    const r = el.getBoundingClientRect();
    if (r.right > vw) el.style.left = x - r.width + "px";
    if (r.bottom > vh) el.style.top = y - r.height + "px";
  }

  return { show, hide };
})();

/* ── MINI PROMPT ────────────────────────────────────────────────
   Lightweight inline prompt to avoid ugly browser prompt()       */
function miniPrompt(fields, prefill, opts) {
  const deletable = opts && opts.deletable;
  return new Promise((resolve) => {
    const overlay = document.createElement("div");
    overlay.style.cssText = `position:fixed;inset:0;z-index:10000;display:flex;align-items:center;justify-content:center;background:rgba(0,0,0,0.6)`;
    const box = document.createElement("div");
    box.style.cssText = `background:var(--panel);border:1px solid var(--border-lt);padding:20px;display:flex;flex-direction:column;gap:10px;min-width:280px;clip-path:polygon(0 0,100% 0,100% calc(100% - 10px),calc(100% - 14px) 100%,0 100%)`;
    const inputs = {};
    fields.forEach(({ key, placeholder }) => {
      const inp = document.createElement("input");
      inp.type = "text";
      inp.placeholder = placeholder;
      inp.style.cssText = `background:var(--panel2);border:1px solid var(--border);color:var(--white);font-family:var(--font-pixel);font-size:11px;padding:6px 10px;outline:none;width:100%`;
      if (prefill && prefill[key] !== undefined) inp.value = prefill[key];
      inputs[key] = inp;
      box.appendChild(inp);
    });
    const row = document.createElement("div");
    row.style.cssText = `display:flex;gap:8px;justify-content:flex-end`;
    const finish = (val) => {
      document.body.removeChild(overlay);
      resolve(val);
    };
    if (deletable) {
      const del = document.createElement("button");
      del.textContent = "delete";
      del.className = "timer-btn";
      del.style.cssText = `margin-right:auto;color:var(--red);border-color:var(--red)`;
      del.addEventListener("click", () => finish("delete"));
      row.appendChild(del);
    }
    const ok = document.createElement("button");
    ok.textContent = deletable ? "save" : "ok";
    ok.className = "timer-btn";
    const cancel = document.createElement("button");
    cancel.textContent = "cancel";
    cancel.className = "timer-btn";
    row.appendChild(cancel);
    row.appendChild(ok);
    box.appendChild(row);
    overlay.appendChild(box);
    document.body.appendChild(overlay);
    const firstInp = Object.values(inputs)[0];
    firstInp.focus();
    ok.addEventListener("click", () => {
      const result = {};
      Object.entries(inputs).forEach(([k, v]) => (result[k] = v.value.trim()));
      finish(result);
    });
    cancel.addEventListener("click", () => finish(null));
    overlay.addEventListener("keydown", (e) => {
      if (e.key === "Enter") ok.click();
      if (e.key === "Escape") cancel.click();
    });
  });
}

/* ── EDIT MODE ──────────────────────────────────────────────── */
const editToggle = document.getElementById("edit-toggle");
editToggle.addEventListener("click", () => {
  document.body.classList.toggle("edit-mode");
});

/* ── SETTINGS ───────────────────────────────────────────────── */
// settings page removed — configuration lives in config.yaml

/* ── FAVICON AUTO-FETCH ─────────────────────────────────────── */
function guessFavicon(url) {
  try {
    const { hostname } = new URL(url);
    return `https://icons.duckduckgo.com/ip3/${hostname}.ico`;
  } catch {
    return "";
  }
}

/* ── SHARED LINK DIALOG ──────────────────────────────────────── */
function linkPrompt(prefill) {
  const isEdit = !!prefill;
  return new Promise((resolve) => {
    const overlay = document.createElement("div");
    overlay.style.cssText = `position:fixed;inset:0;z-index:10000;display:flex;align-items:center;justify-content:center;background:rgba(0,0,0,0.65)`;
    const box = document.createElement("div");
    box.style.cssText = `background:var(--panel);border:1px solid var(--border-lt);padding:18px;display:flex;flex-direction:column;gap:10px;min-width:300px;clip-path:polygon(0 0,100% 0,100% calc(100% - 10px),calc(100% - 14px) 100%,0 100%)`;
    const s = (el, css) => { el.style.cssText = css; return el; };
    const inp = (ph) => {
      const i = document.createElement("input");
      i.type = "text";
      i.placeholder = ph;
      s(i, `background:var(--panel2);border:1px solid var(--border);color:var(--white);font-family:var(--font-pixel);font-size:11px;padding:6px 10px;outline:none;width:100%;transition:border-color 0.1s`);
      i.addEventListener("focus", () => (i.style.borderColor = "var(--accent)"));
      i.addEventListener("blur",  () => (i.style.borderColor = "var(--border)"));
      return i;
    };
    const labelInp = inp("label");
    const urlInp   = inp("https://...");
    const favInp   = inp("favicon url (optional override)");
    if (isEdit) {
      labelInp.value = prefill.label || "";
      urlInp.value   = prefill.url   || "";
      favInp.value   = prefill.fav   || "";
    }
    const previewRow = document.createElement("div");
    s(previewRow, `display:flex;align-items:center;gap:10px;padding:8px;background:var(--panel2);border:1px solid var(--border)`);
    const previewImg = document.createElement("img");
    s(previewImg, `width:24px;height:24px;object-fit:contain;display:none`);
    const previewLetter = document.createElement("div");
    s(previewLetter, `width:24px;height:24px;display:flex;align-items:center;justify-content:center;background:var(--panel);border:1px solid var(--border-lt);color:var(--dim);font-family:var(--font-pixel);font-size:12px;text-transform:uppercase;flex-shrink:0`);
    previewLetter.textContent = "?";
    const previewLabel = document.createElement("span");
    s(previewLabel, `font-family:var(--font-pixel);font-size:11px;color:var(--dimmer);flex:1;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap`);
    previewLabel.textContent = "preview";
    const fetchStatus = document.createElement("span");
    s(fetchStatus, `font-family:var(--font-pixel);font-size:9px;color:var(--dimmer);flex-shrink:0`);
    previewRow.appendChild(previewImg);
    previewRow.appendChild(previewLetter);
    previewRow.appendChild(previewLabel);
    previewRow.appendChild(fetchStatus);
    function showFav(src, lbl) {
      previewImg.src = src;
      previewImg.style.display = "block";
      previewLetter.style.display = "none";
      previewImg.onerror = () => {
        previewImg.style.display = "none";
        previewLetter.style.display = "flex";
        previewLetter.textContent = (lbl || "?")[0];
        fetchStatus.textContent = "no favicon";
      };
      previewImg.onload = () => { fetchStatus.textContent = ""; };
    }
    function updatePreview() {
      const lbl = labelInp.value.trim();
      previewLabel.textContent = lbl || "preview";
      previewLetter.textContent = (lbl || "?")[0];
      const override = favInp.value.trim();
      if (override) { fetchStatus.textContent = ""; showFav(override, lbl); }
      else {
        const auto = guessFavicon(urlInp.value.trim());
        if (auto) { fetchStatus.textContent = "auto"; showFav(auto, lbl); }
        else { previewImg.style.display = "none"; previewLetter.style.display = "flex"; fetchStatus.textContent = ""; }
      }
    }
    urlInp.addEventListener("blur", () => {
      const url = urlInp.value.trim();
      if (!url) return;
      if (!labelInp.value.trim()) {
        try { const host = new URL(url).hostname.replace(/^www\./, ""); labelInp.value = host.split(".")[0]; } catch {}
      }
      updatePreview();
    });
    urlInp.addEventListener("input", updatePreview);
    favInp.addEventListener("input", updatePreview);
    labelInp.addEventListener("input", () => {
      const lbl = labelInp.value.trim();
      previewLabel.textContent = lbl || "preview";
      previewLetter.textContent = (lbl || "?")[0];
    });
    const btnRow = document.createElement("div");
    s(btnRow, `display:flex;gap:8px;justify-content:flex-end`);
    const finish = (result) => { document.body.removeChild(overlay); resolve(result); };
    if (isEdit) {
      const del = document.createElement("button");
      del.textContent = "delete";
      del.className = "timer-btn";
      del.style.cssText = `margin-right:auto;color:var(--red);border-color:var(--red)`;
      del.addEventListener("click", () => finish("delete"));
      btnRow.appendChild(del);
    }
    const cancel = document.createElement("button");
    cancel.textContent = "cancel";
    cancel.className = "timer-btn";
    const ok = document.createElement("button");
    ok.textContent = isEdit ? "save" : "add";
    ok.className = "timer-btn";
    btnRow.appendChild(cancel);
    btnRow.appendChild(ok);
    box.appendChild(labelInp);
    box.appendChild(urlInp);
    box.appendChild(favInp);
    box.appendChild(previewRow);
    box.appendChild(btnRow);
    overlay.appendChild(box);
    document.body.appendChild(overlay);
    labelInp.focus();
    if (isEdit) updatePreview();
    ok.addEventListener("click", () => {
      const label = labelInp.value.trim();
      const url   = urlInp.value.trim();
      if (!label || !url) return;
      const fav = favInp.value.trim() || guessFavicon(url);
      finish({ label, url, fav });
    });
    cancel.addEventListener("click", () => finish(null));
    overlay.addEventListener("keydown", (e) => {
      if (e.key === "Enter")  ok.click();
      if (e.key === "Escape") cancel.click();
    });
  });
}

const bookmarkAddPrompt = linkPrompt;

/* ── BOOKMARKS ─────────────────────────────────────────────────
   In-memory state seeded from server-injected __INITIAL_DATA__.
   All mutations update _bmData then save to API. Re-render rebuilds
   the DOM from _bmData (no server round-trip needed for edits).     */
/* BM_DEFAULTS is defined in data/defaults.js */

let _bmData = (window.__INITIAL_DATA__ && window.__INITIAL_DATA__.nt_bookmarks)
  ? window.__INITIAL_DATA__.nt_bookmarks
  : BM_DEFAULTS;

async function saveBm() {
  await Store.set("nt_bookmarks", _bmData);
}

async function loadBookmarks() {
  const list = document.getElementById("folder-list");

  const openFolders = new Set();
  list.querySelectorAll("details.folder").forEach((el, i) => {
    if (el.open) openFolders.add(i);
  });

  list.innerHTML = "";

  _bmData.forEach((folder, fi) => {
    const details = document.createElement("details");
    details.className = "folder";
    details.draggable = true;
    details.dataset.dragIndex = fi;

    const summary = document.createElement("summary");
    summary.className = "folder-head";
    const icon = document.createElement("span");
    icon.className = "folder-icon";
    icon.textContent = "◈";
    summary.appendChild(icon);
    summary.appendChild(document.createTextNode(" " + folder.folder));

    const headBtns = document.createElement("span");
    headBtns.className = "folder-head-btns";

    const addLinkBtn = document.createElement("button");
    addLinkBtn.className = "folder-head-btn";
    addLinkBtn.innerHTML = '<i class="ph-light ph-plus"></i>';
    addLinkBtn.title = "add link";
    addLinkBtn.addEventListener("click", async (e) => {
      e.preventDefault();
      e.stopPropagation();
      const r = await bookmarkAddPrompt();
      if (!r || r === "delete") return;
      _bmData[fi].links.push({ label: r.label, url: r.url, fav: r.fav });
      await saveBm();
      loadBookmarks();
    });

    const editFolderBtn = document.createElement("button");
    editFolderBtn.className = "folder-head-btn";
    editFolderBtn.innerHTML = '<i class="ph-light ph-pencil-simple"></i>';
    editFolderBtn.title = "edit folder";
    editFolderBtn.addEventListener("click", async (e) => {
      e.preventDefault();
      e.stopPropagation();
      const r = await miniPrompt(
        [{ key: "name", placeholder: "folder name" }],
        { name: _bmData[fi].folder },
        { deletable: true },
      );
      if (!r) return;
      if (r === "delete") {
        _bmData.splice(fi, 1);
      } else if (r.name) {
        _bmData[fi].folder = r.name;
      }
      await saveBm();
      loadBookmarks();
    });

    headBtns.appendChild(editFolderBtn);
    headBtns.appendChild(addLinkBtn);
    summary.appendChild(headBtns);
    details.appendChild(summary);

    const linksDiv = document.createElement("div");
    linksDiv.className = "folder-links grid";

    folder.links.forEach((link, li) => {
      const a = document.createElement("a");
      a.href = link.url;
      a.target = "_blank";
      a.className = "fav-tile";
      a.draggable = true;
      a.dataset.dragIndex = li;

      const img = document.createElement("img");
      img.className = "fav";
      img.src = link.fav || guessFavicon(link.url);
      img.alt = "";
      img.addEventListener("error", () => {
        const letter = document.createElement("div");
        letter.className = "fav-letter";
        letter.textContent = (link.label || "?")[0];
        img.replaceWith(letter);
      });

      const label = document.createElement("span");
      label.textContent = link.label;

      const edit = document.createElement("button");
      edit.className = "tile-edit";
      edit.innerHTML = '<i class="ph-light ph-pencil-simple"></i>';
      edit.title = "edit";
      edit.addEventListener("click", async (e) => {
        e.preventDefault();
        e.stopImmediatePropagation();
        const current = _bmData[fi].links[li];
        const r = await bookmarkAddPrompt({
          label: current.label,
          url: current.url,
          fav: current.fav,
        });
        if (!r) return;
        if (r === "delete") {
          _bmData[fi].links.splice(li, 1);
        } else {
          _bmData[fi].links[li] = { label: r.label, url: r.url, fav: r.fav };
        }
        await saveBm();
        loadBookmarks();
      });
      edit.addEventListener("mousedown", (e) => e.stopImmediatePropagation());
      edit.addEventListener("mouseup", (e) => e.stopImmediatePropagation());

      a.appendChild(img);
      a.appendChild(label);
      a.appendChild(edit);
      linksDiv.appendChild(a);
    });

    details.appendChild(linksDiv);
    enableDragSort(
      linksDiv,
      "a[data-drag-index]",
      () => _bmData[fi].links,
      (arr) => { _bmData[fi].links = arr; saveBm(); },
      loadBookmarks,
    );
    if (openFolders.has(fi)) details.open = true;
    list.appendChild(details);
  });
}

const folderList = document.getElementById("folder-list");
if (folderList) {
  enableDragSort(
    folderList,
    "details[data-drag-index]",
    () => _bmData,
    (arr) => { _bmData = arr; saveBm(); },
    loadBookmarks,
  );
}

// Run on load to wire up all event listeners.
// SSR gave us the structure; js takes over for interactivity.
loadBookmarks();

document
  .getElementById("bm-add-folder-btn")
  .addEventListener("click", async () => {
    const r = await miniPrompt([{ key: "name", placeholder: "folder name" }]);
    if (!r || !r.name) return;
    _bmData.push({ folder: r.name, links: [] });
    await saveBm();
    loadBookmarks();
  });
function linkPrompt(prefill) {
  const isEdit = !!prefill;
  return new Promise((resolve) => {
    const overlay = document.createElement("div");
    overlay.style.cssText = `position:fixed;inset:0;z-index:10000;display:flex;align-items:center;justify-content:center;background:rgba(0,0,0,0.65)`;

    const box = document.createElement("div");
    box.style.cssText = `background:var(--panel);border:1px solid var(--border-lt);padding:18px;display:flex;flex-direction:column;gap:10px;min-width:300px;clip-path:polygon(0 0,100% 0,100% calc(100% - 10px),calc(100% - 14px) 100%,0 100%)`;

    const s = (el, css) => {
      el.style.cssText = css;
      return el;
    };
    const inp = (ph) => {
      const i = document.createElement("input");
      i.type = "text";
      i.placeholder = ph;
      s(
        i,
        `background:var(--panel2);border:1px solid var(--border);color:var(--white);font-family:var(--font-pixel);font-size:11px;padding:6px 10px;outline:none;width:100%;transition:border-color 0.1s`,
      );
      i.addEventListener(
        "focus",
        () => (i.style.borderColor = "var(--accent)"),
      );
      i.addEventListener("blur", () => (i.style.borderColor = "var(--border)"));
      return i;
    };

    const labelInp = inp("label");
    const urlInp = inp("https://...");
    const favInp = inp("favicon url (optional override)");

    if (isEdit) {
      labelInp.value = prefill.label || "";
      urlInp.value = prefill.url || "";
      favInp.value = prefill.fav || "";
    }

    // preview row
    const previewRow = document.createElement("div");
    s(
      previewRow,
      `display:flex;align-items:center;gap:10px;padding:8px;background:var(--panel2);border:1px solid var(--border)`,
    );

    const previewImg = document.createElement("img");
    s(previewImg, `width:24px;height:24px;object-fit:contain;display:none`);

    const previewLetter = document.createElement("div");
    s(
      previewLetter,
      `width:24px;height:24px;display:flex;align-items:center;justify-content:center;background:var(--panel);border:1px solid var(--border-lt);color:var(--dim);font-family:var(--font-pixel);font-size:12px;text-transform:uppercase;flex-shrink:0`,
    );
    previewLetter.textContent = "?";

    const previewLabel = document.createElement("span");
    s(
      previewLabel,
      `font-family:var(--font-pixel);font-size:11px;color:var(--dimmer);flex:1;min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap`,
    );
    previewLabel.textContent = "preview";

    const fetchStatus = document.createElement("span");
    s(
      fetchStatus,
      `font-family:var(--font-pixel);font-size:9px;color:var(--dimmer);flex-shrink:0`,
    );

    previewRow.appendChild(previewImg);
    previewRow.appendChild(previewLetter);
    previewRow.appendChild(previewLabel);
    previewRow.appendChild(fetchStatus);

    function showFav(src, lbl) {
      previewImg.src = src;
      previewImg.style.display = "block";
      previewLetter.style.display = "none";
      previewImg.onerror = () => {
        previewImg.style.display = "none";
        previewLetter.style.display = "flex";
        previewLetter.textContent = (lbl || "?")[0];
        fetchStatus.textContent = "no favicon";
      };
      previewImg.onload = () => {
        fetchStatus.textContent = "";
      };
    }

    function updatePreview() {
      const lbl = labelInp.value.trim();
      previewLabel.textContent = lbl || "preview";
      previewLetter.textContent = (lbl || "?")[0];

      const override = favInp.value.trim();
      if (override) {
        fetchStatus.textContent = "";
        showFav(override, lbl);
      } else {
        const auto = guessFavicon(urlInp.value.trim());
        if (auto) {
          fetchStatus.textContent = "auto";
          showFav(auto, lbl);
        } else {
          previewImg.style.display = "none";
          previewLetter.style.display = "flex";
          fetchStatus.textContent = "";
        }
      }
    }

    // Auto-fetch on URL change
    urlInp.addEventListener("blur", () => {
      const url = urlInp.value.trim();
      if (!url) return;
      // Auto-fill label from domain if empty
      if (!labelInp.value.trim()) {
        try {
          const host = new URL(url).hostname.replace(/^www\./, "");
          labelInp.value = host.split(".")[0];
        } catch {}
      }
      updatePreview();
    });
    urlInp.addEventListener("input", updatePreview);
    favInp.addEventListener("input", updatePreview);
    labelInp.addEventListener("input", () => {
      const lbl = labelInp.value.trim();
      previewLabel.textContent = lbl || "preview";
      previewLetter.textContent = (lbl || "?")[0];
    });

    const btnRow = document.createElement("div");
    s(btnRow, `display:flex;gap:8px;justify-content:flex-end`);

    const finish = (result) => {
      document.body.removeChild(overlay);
      resolve(result);
    };

    if (isEdit) {
      const del = document.createElement("button");
      del.textContent = "delete";
      del.className = "timer-btn";
      del.style.cssText = `margin-right:auto;color:var(--red);border-color:var(--red)`;
      del.addEventListener("click", () => finish("delete"));
      btnRow.appendChild(del);
    }

    const cancel = document.createElement("button");
    cancel.textContent = "cancel";
    cancel.className = "timer-btn";
    const ok = document.createElement("button");
    ok.textContent = isEdit ? "save" : "add";
    ok.className = "timer-btn";
    btnRow.appendChild(cancel);
    btnRow.appendChild(ok);

    box.appendChild(labelInp);
    box.appendChild(urlInp);
    box.appendChild(favInp);
    box.appendChild(previewRow);
    box.appendChild(btnRow);
    overlay.appendChild(box);
    document.body.appendChild(overlay);
    labelInp.focus();

    if (isEdit) updatePreview();

    ok.addEventListener("click", () => {
      const label = labelInp.value.trim();
      const url = urlInp.value.trim();
      if (!label || !url) return;
      // Save override if provided, otherwise save the auto-fetched URL
      const fav = favInp.value.trim() || guessFavicon(url);
      finish({ label, url, fav });
    });
    cancel.addEventListener("click", () => finish(null));
    overlay.addEventListener("keydown", (e) => {
      if (e.key === "Enter") ok.click();
      if (e.key === "Escape") cancel.click();
    });
  });
}

/* ── RECENTLY VISITED ───────────────────────────────────────────
   In-memory, seeded from __INITIAL_DATA__.                        */
let _recentData = (window.__INITIAL_DATA__ && window.__INITIAL_DATA__.nt_recent)
  ? window.__INITIAL_DATA__.nt_recent
  : [];

async function saveRecent() {
  await Store.set("nt_recent", _recentData);
}

function loadRecent() {
  const grid = document.getElementById("recent-grid");
  grid.innerHTML = "";
  _recentData.forEach((item, i) => {
    const tile = document.createElement("div");
    tile.className = "recent-tile";
    const rtName = document.createElement("div");
    rtName.className = "rt-name";
    rtName.textContent = item.name;
    const rtUrl = document.createElement("div");
    rtUrl.className = "rt-url";
    rtUrl.textContent = item.url;
    tile.appendChild(rtName);
    tile.appendChild(rtUrl);
    tile.addEventListener("click", () => window.open(item.url, "_blank"));
    const x = document.createElement("button");
    x.className = "recent-x";
    x.innerHTML = '<i class="ph-light ph-x"></i>';
    x.addEventListener("click", async (e) => {
      e.stopPropagation();
      _recentData.splice(i, 1);
      await saveRecent();
      loadRecent();
    });
    tile.appendChild(x);
    grid.appendChild(tile);
  });
}

async function quickAddRecent() {
  const name = document.getElementById("rqa-name").value.trim();
  const url  = document.getElementById("rqa-url").value.trim();
  if (!name || !url) return;
  _recentData.unshift({ name, url });
  if (_recentData.length > 8) _recentData.pop();
  await saveRecent();
  document.getElementById("rqa-name").value = "";
  document.getElementById("rqa-url").value  = "";
  loadRecent();
}

document.getElementById("rqa-btn")?.addEventListener("click", quickAddRecent);
document.getElementById("rqa-url")?.addEventListener("keydown", (e) => {
  if (e.key === "Enter") quickAddRecent();
});

// Wire delete handlers on ssr-rendered recent tiles.
loadRecent();

/* ── QUICK ACCESS ───────────────────────────────────────────────
   In-memory, seeded from __INITIAL_DATA__.                        */
/* QA_DEFAULTS is defined in data/defaults.js */

let _qaData = (window.__INITIAL_DATA__ && window.__INITIAL_DATA__.nt_quick)
  ? window.__INITIAL_DATA__.nt_quick
  : QA_DEFAULTS;

async function saveQa() {
  await Store.set("nt_quick", _qaData);
}

function loadQuickAccess() {
  const container = document.querySelector(".quick-links");
  if (!container) return;
  container.innerHTML = "";
  _qaData.forEach((item, i) => {
    const a = document.createElement("a");
    a.href = item.url;
    a.target = "_blank";
    a.draggable = true;
    a.dataset.dragIndex = i;
    const favImg = document.createElement("img");
    favImg.className = "ql-fav";
    favImg.src = item.favicon || item.fav || guessFavicon(item.url);
    favImg.alt = "";
    a.appendChild(favImg);
    a.appendChild(document.createTextNode(" " + item.label));
    const btnGroup = document.createElement("span");
    btnGroup.style.cssText = "margin-left:auto;display:none;align-items:center;gap:0;flex-shrink:0;";
    btnGroup.className = "qa-btn-group";
    const edit = document.createElement("button");
    edit.className = "qa-edit";
    edit.innerHTML = '<i class="ph-light ph-pencil-simple"></i>';
    edit.title = "edit";
    edit.addEventListener("click", async (e) => {
      e.preventDefault();
      e.stopPropagation();
      const current = _qaData[i];
      const r = await linkPrompt({ label: current.label, url: current.url, fav: current.favicon || current.fav });
      if (!r) return;
      if (r === "delete") {
        _qaData.splice(i, 1);
      } else if (r.label && r.url) {
        _qaData[i] = { label: r.label, url: r.url, favicon: r.fav || "" };
      }
      await saveQa();
      loadQuickAccess();
    });
    btnGroup.appendChild(edit);
    a.appendChild(btnGroup);
    container.appendChild(a);
  });
}

const quickContainer = document.querySelector(".quick-links");
if (quickContainer) {
  enableDragSort(
    quickContainer,
    "a[data-drag-index]",
    () => _qaData,
    (arr) => { _qaData = arr; saveQa(); },
    loadQuickAccess,
  );
}

document.getElementById("qa-add-btn")?.addEventListener("click", async () => {
  const r = await linkPrompt();
  if (!r || !r.label || !r.url) return;
  _qaData.push({ label: r.label, url: r.url, favicon: r.fav || "" });
  await saveQa();
  loadQuickAccess();
});

// Wire edit handlers — SSR gave us the links, js makes them interactive.
loadQuickAccess();

/* ── QUOTES ─────────────────────────────────────────────────── */
/* QUOTES array is defined in data/quotes.js */

// Get the current quote text and author from the DOM
const quoteTextEl = document.getElementById("quote-text");
const quoteAuthorEl = document.getElementById("quote-author");

let quoteIndex = 0;

// If the DOM already has a quote, find its index in QUOTES
if (quoteTextEl && quoteAuthorEl) {
  const currentText = quoteTextEl.textContent.trim();
  const currentAuthor = quoteAuthorEl.textContent.trim();
  const foundIndex = QUOTES.findIndex(
    (q) => q.text === currentText && q.author === currentAuthor,
  );
  if (foundIndex !== -1) {
    quoteIndex = foundIndex;
  } else {
    // If not found (e.g., custom quote), set to a random index
    quoteIndex = Math.floor(Math.random() * QUOTES.length);
    // Update the DOM with a random quote from QUOTES
    const q = QUOTES[quoteIndex];
    quoteTextEl.textContent = q.text;
    quoteAuthorEl.textContent = q.author;
  }
} else {
  // Fallback: set first quote
  quoteIndex = 0;
  const q = QUOTES[0];
  if (quoteTextEl) quoteTextEl.textContent = q.text;
  if (quoteAuthorEl) quoteAuthorEl.textContent = q.author;
}

document.getElementById("quote-next")?.addEventListener("click", () => {
  let idx;
  do {
    idx = Math.floor(Math.random() * QUOTES.length);
  } while (idx === quoteIndex && QUOTES.length > 1);
  quoteIndex = idx;
  const q = QUOTES[idx];
  if (quoteTextEl) quoteTextEl.textContent = q.text;
  if (quoteAuthorEl) quoteAuthorEl.textContent = q.author;
});

/* ── KANJI WORD OF THE DAY ──────────────────────────────────── */
/* WORDS array is defined in data/words.js */

function renderWord(w) {
  const el = document.getElementById("kanji-char");
  if (el) {
    el.innerHTML = "";
    const a = document.createElement("a");
    a.href =
      "https://jisho.org/search/" + encodeURIComponent(w.k) + "%20%23kanji";
    a.target = "_blank";
    a.style.cssText = "color:inherit;text-decoration:none;";
    a.textContent = w.k;
    el.appendChild(a);
  }
  const readingEl = document.getElementById("kanji-reading");
  if (readingEl) readingEl.textContent = w.r;
  const meaningEl = document.getElementById("kanji-meaning");
  if (meaningEl) meaningEl.textContent = w.m;
  const lvlEl = document.getElementById("kanji-level");
  if (lvlEl) {
    lvlEl.textContent = w.l.toUpperCase();
    lvlEl.className = "kanji-level " + w.l;
  }
}

// Get current word from DOM
let wordIndex = 0;
const charEl = document.getElementById("kanji-char");
if (charEl) {
  const currentChar = charEl.textContent.trim();
  // Find index in WORDS
  const foundIndex = WORDS.findIndex((w) => w.k === currentChar);
  if (foundIndex !== -1) {
    wordIndex = foundIndex;
  } else {
    // Not found, set random
    wordIndex = Math.floor(Math.random() * WORDS.length);
    renderWord(WORDS[wordIndex]);
  }
} else {
  wordIndex = 0;
  renderWord(WORDS[0]);
}

document.getElementById("kanji-next")?.addEventListener("click", () => {
  let idx;
  do {
    idx = Math.floor(Math.random() * WORDS.length);
  } while (idx === wordIndex && WORDS.length > 1);
  wordIndex = idx;
  renderWord(WORDS[wordIndex]);
});

/* ── WIDGET IMAGE ────────────────────────────────────────────── */
(function () {
  const btn  = document.getElementById("widget-img-next");
  const wrap = document.getElementById("widget-img-wrap");
  if (!btn || !wrap) return;

  let currentFilename = wrap.querySelector("img")?.dataset.filename || "";
  let preloaded = null;

  function setImage(url, filename) {
    const img = document.createElement("img");
    img.src = url;
    img.id = "widget-img";
    img.className = "active";
    img.dataset.filename = filename;
    wrap.innerHTML = "";
    wrap.appendChild(img);
    currentFilename = filename;
  }

  function preloadNext(afterFilename) {
    fetch(`/api/widget-images/next?current=${encodeURIComponent(afterFilename)}`)
      .then((r) => r.json())
      .then((d) => {
        if (!d.url) return;
        const img = new Image();
        img.src = d.url; // browser fetches silently
        preloaded = { url: d.url, filename: d.filename };
      })
      .catch(() => {});
  }

  btn.addEventListener("click", () => {
    if (!preloaded) {
      // preload not ready yet (e.g. clicked faster than the bg fetch completed)
      fetch(`/api/widget-images/next?current=${encodeURIComponent(currentFilename)}`)
        .then((r) => r.json())
        .then((d) => {
          if (!d.url) return;
          setImage(d.url, d.filename);
          preloadNext(d.filename);
        })
        .catch(() => {});
      return;
    }
    setImage(preloaded.url, preloaded.filename);
    preloaded = null;
    preloadNext(currentFilename);
  });

  // Start preloading immediately on page load so first click is instant.
  if (currentFilename) preloadNext(currentFilename);
})();
/* Theme class is set server-side on <body> via the template.
   settings.js still uses localStorage for instant preview, but
   the canonical value lives in the db and is applied by Go.       */

/* ── FOCUS SEARCH ON KEYPRESS ───────────────────────────────── */
document.addEventListener("keydown", (e) => {
  const el = document.getElementById("search-input");
  const tag = document.activeElement.tagName;
  if (tag === "INPUT" || tag === "TEXTAREA") return;
  if (!e.ctrlKey && !e.metaKey && !e.altKey && e.key.length === 1) {
    el.focus();
  }
});

// Focus search on load. Firefox retains url bar focus on new tab opens
// even after window.load fires, so we delay slightly to override it.
window.addEventListener("load", () => {
  const el = document.getElementById("search-input");
  if (!el) return;
  // immediate attempt
  el.focus();
  // delayed fallback to beat firefox's new tab focus management
  setTimeout(() => el.focus(), 100);
});
/* ── PROFILE TAGLINE ─────────────────────────────────────────── */
(function () {
  const el = document.getElementById("profile-tagline");
  if (el)
    el.textContent = TAGLINES[Math.floor(Math.random() * TAGLINES.length)];
})();

/* ── SEARCH ENGINE WIRING ────────────────────────────────────── */
/* Engines are SSR'd as buttons; we wire click + Enter here.      */
(function () {
  const d = window.__INITIAL_DATA__ || {};
  const searchTarget = d.nt_search_target || "_blank";

  // engines may already be rendered by server; just wire them
  const bar = document.querySelector(".engine-bar") || document.querySelector(".search-engines");
  if (bar) {
    bar.querySelectorAll(".engine-btn").forEach((btn) => {
      btn.addEventListener("click", () => {
        bar.querySelectorAll(".engine-btn").forEach((b) => b.classList.remove("active"));
        btn.classList.add("active");
        activeEngine = btn;
      });
    });
    activeEngine = bar.querySelector(".engine-btn.active") || bar.querySelector(".engine-btn");
  }

  const searchInput = document.getElementById("search-input");
  if (searchInput) {
    const newInput = searchInput.cloneNode(true);
    searchInput.parentNode.replaceChild(newInput, searchInput);
    newInput.addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        const q = e.target.value.trim();
        if (!q) return;
        const url = (activeEngine?.dataset?.url || "https://duckduckgo.com/?q=") + encodeURIComponent(q);
        window.open(url, searchTarget);
        e.target.value = "";
      }
    });
  }
})();


/* ── SYSTEM STATS WIDGET ──────────────────────────────────────── */
(function () {
  const canvas = document.getElementById("stats-canvas");
  if (!canvas) return;
  const ctx = canvas.getContext("2d");

  const history = { cpu: [], mem: [] };
  const MAX = 30;

  // Keep canvas pixel dimensions in sync with its CSS size.
  // This fixes stretching when the widget is in a wide column.
  function syncSize() {
    const rect = canvas.getBoundingClientRect();
    const w = Math.round(rect.width);
    const h = Math.round(rect.height);
    if (canvas.width !== w || canvas.height !== h) {
      canvas.width  = w;
      canvas.height = h;
    }
  }
  const ro = new ResizeObserver(syncSize);
  ro.observe(canvas);
  syncSize();

  async function fetchStats() {
    try {
      const r = await fetch("/api/stats");
      if (!r.ok) return null;
      return r.json();
    } catch {
      return null;
    }
  }

  function pushHistory(arr, val) {
    arr.push(val);
    if (arr.length > MAX) arr.shift();
  }

  function drawGraph(data, color, yOffset, height) {
    if (!data.length) return;
    const w = canvas.width;
    const step = w / (MAX - 1);

    ctx.beginPath();
    ctx.strokeStyle = color;
    ctx.lineWidth = 1.5;

    data.forEach((val, i) => {
      const x = i * step;
      const y = yOffset + height - (val / 100) * height;
      if (i === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    });
    ctx.stroke();

    ctx.lineTo((data.length - 1) * step, yOffset + height);
    ctx.lineTo(0, yOffset + height);
    ctx.closePath();
    ctx.fillStyle = color.replace(")", ", 0.08)").replace("rgb", "rgba");
    ctx.fill();
  }

  function render(stats) {
    const w = canvas.width;
    const h = canvas.height;
    if (!w || !h) return;
    const style = getComputedStyle(document.documentElement);
    const dim    = style.getPropertyValue("--dimmer").trim();
    const border = style.getPropertyValue("--border").trim();
    const white  = style.getPropertyValue("--white").trim();

    ctx.clearRect(0, 0, w, h);

    // Grid lines
    ctx.strokeStyle = border;
    ctx.lineWidth = 0.5;
    for (let i = 0; i <= 4; i++) {
      const y  = Math.round((h / 2) * (i / 4));
      const y2 = Math.round(h / 2 + (h / 2) * (i / 4));
      ctx.beginPath(); ctx.moveTo(0, y);  ctx.lineTo(w, y);  ctx.stroke();
      ctx.beginPath(); ctx.moveTo(0, y2); ctx.lineTo(w, y2); ctx.stroke();
    }

    // CPU graph (top half)
    drawGraph(history.cpu, `rgb(138, 180, 248)`, 0, h / 2 - 4);
    // MEM graph (bottom half)
    drawGraph(history.mem, `rgb(144, 216, 112)`, h / 2 + 4, h / 2 - 4);

    // Labels
    ctx.font = `9px var(--font-pixel, monospace)`;
    ctx.fillStyle = dim;
    ctx.fillText("CPU", 4, 12);
    ctx.fillText("MEM", 4, h / 2 + 16);

    if (stats) {
      ctx.fillStyle = white;
      ctx.fillText(`${Math.round(stats.cpu || 0)}%`, w - 28, 12);
      ctx.fillText(`${Math.round(stats.ram || 0)}%`, w - 28, h / 2 + 16);
    }

    // Divider
    ctx.strokeStyle = border;
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(0, h / 2);
    ctx.lineTo(w, h / 2);
    ctx.stroke();
  }

  async function update() {
    syncSize();
    const stats = await fetchStats();
    if (stats) {
      pushHistory(history.cpu, stats.cpu || 0);
      pushHistory(history.mem, stats.ram || 0);
    } else {
      pushHistory(history.cpu, 0);
      pushHistory(history.mem, 0);
    }
    render(stats);
  }

  update();
  setInterval(update, 2000);
})();
