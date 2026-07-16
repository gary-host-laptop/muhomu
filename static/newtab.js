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

/* ── DRAG AND DROP SORT ─────────────────────────────────────── */
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
    container.querySelectorAll(itemSelector).forEach((el) => el.classList.remove("drag-over"));
    item.classList.add("drag-over");
  });
  container.addEventListener("dragleave", (e) => {
    if (!e.relatedTarget || !container.contains(e.relatedTarget)) {
      container.querySelectorAll(itemSelector).forEach((el) => el.classList.remove("drag-over"));
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
    container.querySelectorAll(itemSelector).forEach((el) => el.classList.remove("dragging", "drag-over"));
    dragSrc = null;
  });
}

/* ── CLOCK ───────────────────────────────────────────────────── */
function pad(n) { return String(n).padStart(2, "0"); }

const DAYS   = ["sunday","monday","tuesday","wednesday","thursday","friday","saturday"];
const MONTHS = ["january","february","march","april","may","june",
                "july","august","september","october","november","december"];

// Read clock settings from body data attributes set by server.
window._clockFormat  = document.body.dataset.clockFormat  || "24h";
window._clockSeconds = document.body.dataset.clockSeconds !== "false";
window._uiLang       = document.body.dataset.uiLang       || "en";

function getGreetingStrings(h, lang) {
  if (lang === "ja") {
    if (h < 5)  return ["おやすみ","夜更かしですね"];
    if (h < 12) return ["おはようございます","おはよう"];
    if (h < 17) return ["こんにちは","午後もよろしく"];
    if (h < 21) return ["こんばんは","お疲れ様です"];
    return ["おやすみなさい","また明日"];
  }
  if (h < 5)  return ["still up?","burning the midnight oil"];
  if (h < 12) return ["good morning","rise and shine"];
  if (h < 17) return ["good afternoon","hope your day is going well"];
  if (h < 21) return ["good evening","winding down?"];
  return ["good night","time to rest"];
}

function setGreeting(el, text) {
  if (!el) return;
  el.style.opacity = "0";
  setTimeout(() => { el.textContent = text; el.style.opacity = "1"; }, 200);
}

let greetingSet = null;
let lastHour = -1;

function getWeekNum(d) {
  const onejan = new Date(d.getFullYear(), 0, 1);
  return Math.ceil(((d - onejan) / 86400000 + onejan.getDay() + 1) / 7);
}

function tick() {
  const now  = new Date();
  const rawH = now.getHours();
  const m    = now.getMinutes();
  const s    = now.getSeconds();

  let h = rawH, suffix = "";
  if (window._clockFormat === "12h") {
    suffix = h >= 12 ? " PM" : " AM";
    h = h % 12 || 12;
  }
  const clockEl = document.getElementById("clock");
  if (clockEl) {
    clockEl.textContent = window._clockSeconds
      ? `${pad(h)}:${pad(m)}:${pad(s)}${suffix}`
      : `${pad(h)}:${pad(m)}${suffix}`;
  }

  if (rawH !== lastHour) {
    lastHour = rawH;
    greetingSet = getGreetingStrings(rawH, window._uiLang);
    setGreeting(document.getElementById("greeting"),
      greetingSet[Math.floor(Math.random() * greetingSet.length)]);
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

/* ── SEARCH ENGINE WIRING ────────────────────────────────────── */
let activeEngine = null;
(function () {
  const d = window.__INITIAL_DATA__ || {};
  const bar = document.querySelector(".search-engines");
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
        window.location.href = url;
        e.target.value = "";
      }
    });
  }
})();

/* ── FOCUS SEARCH ON KEYPRESS ───────────────────────────────── */
document.addEventListener("keydown", (e) => {
  const el = document.getElementById("search-input");
  const tag = document.activeElement.tagName;
  if (tag === "INPUT" || tag === "TEXTAREA") return;
  if (!e.ctrlKey && !e.metaKey && !e.altKey && e.key.length === 1) {
    el.focus();
  }
});

window.addEventListener("load", () => {
  const el = document.getElementById("search-input");
  if (!el) return;
  el.focus();
  setTimeout(() => el.focus(), 100);
});

/* ── FOOTER LIVE STATS ───────────────────────────────────────────
   Updates network speeds and system stats from /api/stats every 2s. */
(function() {
  const rxEl = document.getElementById("fi-rx");
  const txEl = document.getElementById("fi-tx");
  const cpuEl = document.getElementById("fi-cpu");
  const ramEl = document.getElementById("fi-ram");
  if (!rxEl && !txEl && !cpuEl && !ramEl) return;

  function fmtSpeed(kbps) {
    if (kbps >= 1024) return (kbps / 1024).toFixed(1) + " MB/s";
    return kbps.toFixed(1) + " KB/s";
  }

  function updateFooter() {
    fetch("/api/stats")
      .then(function(r) { return r.json(); })
      .then(function(d) {
        if (rxEl) rxEl.textContent = "⬇ " + fmtSpeed(d.rx || 0);
        if (txEl) txEl.textContent = "⬆ " + fmtSpeed(d.tx || 0);
        if (cpuEl) cpuEl.textContent = "◉ cpu " + (d.cpu || 0).toFixed(1) + "%";
        if (ramEl) ramEl.textContent = "◉ ram " + (d.ram || 0).toFixed(0) + "%";
      })
      .catch(function() {});
  }

  // Set initial values from server-rendered data if available
  var initData = window.__INITIAL_DATA__ || {};
  if (initData.nt_rx !== undefined && rxEl) rxEl.textContent = "⬇ " + fmtSpeed(initData.nt_rx);
  if (initData.nt_tx !== undefined && txEl) txEl.textContent = "⬆ " + fmtSpeed(initData.nt_tx);
  if (initData.nt_cpu !== undefined && cpuEl) cpuEl.textContent = "◉ cpu " + (initData.nt_cpu || 0).toFixed(1) + "%";
  if (initData.nt_ram !== undefined && ramEl) ramEl.textContent = "◉ ram " + (initData.nt_ram || 0).toFixed(0) + "%";

  // Live update every 2 seconds
  setInterval(updateFooter, 2000);
})();

/* ── PROFILE TAGLINE ─────────────────────────────────────────── */
(function () {
  const el = document.getElementById("profile-tagline");
  if (el) el.textContent = TAGLINES[Math.floor(Math.random() * TAGLINES.length)];
})();
