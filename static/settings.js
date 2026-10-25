/* settings.js — the settings page app.
 *
 * Data-driven: fetches the whole configuration surface from
 * GET /api/settings (config + defaults + quotes + themes + images),
 * renders a form for every key, and POSTs the working copy back on
 * save. The server writes config.yaml and the existing hot-reload
 * applies it to the new tab page — this page never touches YAML.
 *
 * Layout (header/columns, widget order only — including split-column
 * children) is edited in place with a nested widget-list editor; each
 * widget's options get their own entry in the widgets section. Images
 * and quotes have their own endpoints and save moments.
 */

(function () {
  "use strict";

  /* ── helpers ─────────────────────────────────────────────── */
  const $ = (sel, root) => (root || document).querySelector(sel);

  function el(tag, cls, text) {
    const n = document.createElement(tag);
    if (cls) n.className = cls;
    if (text !== undefined) n.textContent = text;
    return n;
  }

  function iconBtn(symbol, fn, danger, title) {
    const b = el("button", "icon-btn" + (danger ? " danger" : ""), symbol);
    b.type = "button";
    if (title) b.title = title;
    b.addEventListener("click", fn);
    return b;
  }

  const saveBtn = $("#save-btn");
  const resetBtn = $("#reset-btn");
  const statusEl = $("#save-status");
  const sectionsEl = $("#sections");

  /* ── state ───────────────────────────────────────────────── */
  let cfg = null;            // working copy of the config (mutated by forms)
  let quotes = [];           // saved quotes (server truth)
  let quotesWork = [];       // working copy edited by the quotes manager
  let quotesDirty = false;
  let themes = [];
  let images = { profile: [], bg: [], widget: [] };
  let saveQuotesBtn = null;
  let widgetsSectionBody = null; // rebuilt when layout add/remove changes the widget set
  const imgContainers = { profile: null, bg: null, widget: null };

  function setStatus(msg, cls) {
    statusEl.textContent = msg;
    statusEl.className = "save-status" + (cls ? " " + cls : "");
  }

  function markDirty() {
    saveBtn.disabled = false;
    setStatus("unsaved changes");
  }

  function markQuotesDirty() {
    quotesDirty = true;
    if (saveQuotesBtn) saveQuotesBtn.style.display = "";
    setStatus("unsaved quote changes");
  }

  function getPath(obj, path) {
    return path.split(".").reduce((o, k) => (o == null ? undefined : o[k]), obj);
  }
  function setPath(obj, path, value) {
    const parts = path.split(".");
    let cur = obj;
    for (let i = 0; i < parts.length - 1; i++) {
      const k = parts[i];
      if (typeof cur[k] !== "object" || cur[k] === null) cur[k] = {};
      cur = cur[k];
    }
    cur[parts[parts.length - 1]] = value;
  }
  const clone = (o) => JSON.parse(JSON.stringify(o));

  /* ── theme skinning ──────────────────────────────────────── */
  function applyTheme() {
    document.body.classList.remove("theme-dark", "theme-light");
    document.body.classList.add(cfg.theme === "light" ? "theme-light" : "theme-dark");
    let link = document.getElementById("custom-theme-link");
    if (cfg.theme_file) {
      if (!link) {
        link = document.createElement("link");
        link.id = "custom-theme-link";
        link.rel = "stylesheet";
        document.head.appendChild(link);
      }
      link.href = "/api/themes/" + encodeURIComponent(cfg.theme_file);
    } else if (link) {
      link.remove();
    }
  }

  /* ── generic field renderer ──────────────────────────────── */
  // f: {path, label, type, options?, placeholder?, min?, max?, onChange?, wide?}
  // target: the object the path is read/written on (defaults to cfg).
  function fieldGrid(cols) {
    const grid = el("div", "form-grid");
    if (cols) grid.style.setProperty("--field-cols", cols);
    return grid;
  }

  function renderField(container, f, target) {
    const row = el("div", "form-row" + (f.wide ? " wide" : ""));
    row.appendChild(el("label", null, f.label));
    const t = target || cfg;
    const val = () => getPath(t, f.path);
    const set = (v) => {
      setPath(t, f.path, v);
      markDirty();
      if (f.onChange) f.onChange(v);
    };

    switch (f.type) {
      case "text": {
        const inp = el("input");
        inp.type = "text";
        inp.placeholder = f.placeholder || "";
        inp.value = val() == null ? "" : val();
        inp.addEventListener("input", () => set(inp.value));
        row.appendChild(inp);
        break;
      }
      case "number": {
        const inp = el("input");
        inp.type = "number";
        inp.placeholder = f.placeholder || "";
        inp.value = val() == null ? "" : val();
        inp.addEventListener("input", () => {
          let v = parseInt(inp.value, 10);
          if (isNaN(v)) v = 0;
          if (f.min !== undefined) v = Math.max(f.min, v);
          if (f.max !== undefined) v = Math.min(f.max, v);
          set(v);
        });
        row.appendChild(inp);
        break;
      }
      case "select": {
        const sel = el("select", "form-select");
        f.options.forEach(([v, l]) => {
          const o = el("option");
          o.value = v;
          o.textContent = l || v;
          sel.appendChild(o);
        });
        sel.value = String(val() ?? "");
        sel.addEventListener("change", () => set(sel.value));
        row.appendChild(sel);
        break;
      }
      case "toggle":
      case "segmented": {
        const group = el("div", "option-group");
        const opts = f.type === "toggle"
          ? [["true", "on"], ["false", "off"]]
          : f.options;
        const cur = String(val() ?? (f.type === "toggle" ? "false" : ""));
        opts.forEach(([v, l]) => {
          const b = el("button", "option-btn" + (String(v) === cur ? " selected" : ""), l || v);
          b.addEventListener("click", () => {
            group.querySelectorAll(".option-btn").forEach((x) => x.classList.remove("selected"));
            b.classList.add("selected");
            set(v === "true" ? true : v === "false" ? false : v);
          });
          group.appendChild(b);
        });
        row.appendChild(group);
        break;
      }
      case "color": {
        const wrap = el("div");
        wrap.style.cssText = "display:flex;gap:8px;align-items:center;flex:1";
        const textInp = el("input");
        textInp.type = "text";
        textInp.placeholder = f.placeholder || "css color, e.g. var(--accent)";
        const colorInp = el("input");
        colorInp.type = "color";
        colorInp.style.width = "40px";
        const sync = (v) => {
          set(v);
          textInp.value = v;
          if (/^#[0-9a-fA-F]{6}$/.test(v)) colorInp.value = v;
        };
        textInp.value = val() || "";
        textInp.addEventListener("input", () => sync(textInp.value));
        colorInp.addEventListener("input", () => sync(colorInp.value));
        wrap.appendChild(colorInp);
        wrap.appendChild(textInp);
        row.appendChild(wrap);
        break;
      }
    }
    container.appendChild(row);
    return row;
  }

  /* ── search engines editor ───────────────────────────────── */
  function renderEngines(container) {
    container.innerHTML = "";
    const list = cfg.search_engines || (cfg.search_engines = []);
    list.forEach((e, i) => {
      const row = el("div", "list-row" + (e.default ? " is-active" : ""));
      const main = el("div", "row-main");
      main.appendChild(el("div", "row-title", e.name || "(unnamed)"));
      main.appendChild(el("div", "row-sub", e.url || ""));
      row.appendChild(main);
      if (e.default) row.appendChild(el("span", "row-badge", "default"));
      const acts = el("div", "row-actions");
      acts.appendChild(iconBtn("★", () => {
        list.forEach((x) => (x.default = false));
        e.default = true;
        renderEngines(container);
        markDirty();
      }, false, "make default"));
      acts.appendChild(iconBtn("↑", () => {
        if (i > 0) { [list[i - 1], list[i]] = [list[i], list[i - 1]]; renderEngines(container); markDirty(); }
      }, false, "move up"));
      acts.appendChild(iconBtn("↓", () => {
        if (i < list.length - 1) { [list[i], list[i + 1]] = [list[i + 1], list[i]]; renderEngines(container); markDirty(); }
      }, false, "move down"));
      acts.appendChild(iconBtn("×", () => { list.splice(i, 1); renderEngines(container); markDirty(); }, true, "remove"));
      row.appendChild(acts);
      container.appendChild(row);
    });

    const form = el("div", "add-form");
    const nameInp = el("input");
    nameInp.type = "text";
    nameInp.placeholder = "name e.g. google";
    const urlInp = el("input");
    urlInp.type = "url";
    urlInp.placeholder = "https://google.com/search?q=";
    const saveE = el("button", "btn primary", "add");
    saveE.addEventListener("click", () => {
      const name = nameInp.value.trim();
      const url = urlInp.value.trim();
      if (!name || !url) return;
      list.push({ name, url, default: false });
      renderEngines(container);
      markDirty();
    });
    const btns = el("div", "form-btns");
    btns.appendChild(saveE);
    form.appendChild(nameInp);
    form.appendChild(urlInp);
    form.appendChild(el("div", "form-hint", "append {query} or leave the url ending with = — the query is appended automatically"));
    form.appendChild(btns);
    container.appendChild(form);

    const addBtn = el("button", "btn", "+ add engine");
    addBtn.addEventListener("click", () => form.classList.toggle("visible"));
    container.appendChild(addBtn);
  }

  /* ── footer items checkboxes ─────────────────────────────── */
  function renderFooter(container) {
    container.innerHTML = "";
    const items = ["rx", "tx", "cpu", "ram"];
    const cur = (cfg.footer_items = (cfg.footer_items || []).filter(
      (it) => it !== "version" && it !== "github"
    ));
    const grid = el("div", "check-grid");
    items.forEach((it) => {
      const row = el("label", "check-row");
      const cb = el("input");
      cb.type = "checkbox";
      cb.checked = cur.includes(it);
      cb.addEventListener("change", () => {
        const idx = cur.indexOf(it);
        if (cb.checked && idx < 0) cur.push(it);
        if (!cb.checked && idx >= 0) cur.splice(idx, 1);
        markDirty();
      });
      row.appendChild(cb);
      row.appendChild(document.createTextNode(it));
      grid.appendChild(row);
    });
    container.appendChild(grid);
    container.appendChild(el("div", "form-hint", "muhomu version and the GitHub link always show at the far right."));
  }

  /* ── widget schemas & layout editor ──────────────────────── */
  const WIDGET_LABELS = {
    "quick-access": "quick access", "bookmarks": "bookmarks", "notes": "notes",
    "recently-visited": "recently visited", "rss": "rss", "quote": "quote",
    "kotoba": "kotoba", "system-stats": "system stats", "timer": "timer",
    "rain": "rain", "weather": "weather", "image": "image", "calendar": "calendar",
    "split-column": "split column", "clock": "clock", "identity": "identity",
    "profile": "profile", "search": "search",
  };
  const HEADER_WIDGETS = [
    { id: "clock", label: "clock" },
    { id: "identity", label: "identity" },
    { id: "profile", label: "profile" },
    { id: "search", label: "search" },
  ];
  const BODY_WIDGETS = [
    { id: "quick-access", label: "quick access" },
    { id: "bookmarks", label: "bookmarks" },
    { id: "notes", label: "notes" },
    { id: "recently-visited", label: "recently visited" },
    { id: "rss", label: "rss" },
    { id: "quote", label: "quote" },
    { id: "kotoba", label: "kotoba" },
    { id: "system-stats", label: "system stats" },
    { id: "timer", label: "timer" },
    { id: "rain", label: "rain" },
    { id: "weather", label: "weather" },
    { id: "image", label: "image" },
    { id: "calendar", label: "calendar" },
    { id: "split-column", label: "split column" },
  ];

  // Per-widget option schemas. Every widget also accepts `led`.
  const WIDGET_SCHEMA = {
    "bookmarks": {
      options: [{ path: "rows", label: "rows", type: "number", min: 1, max: 10, placeholder: "3" }],
    },
    "rss": {
      options: [
        { path: "style", label: "style", type: "select", options: [["vertical-list", "vertical list"], ["horizontal-cards", "horizontal cards"]] },
        { path: "limit", label: "limit", type: "number", placeholder: "25" },
        { path: "items-per-page", label: "items per page", type: "number", placeholder: "5" },
        { path: "cache", label: "cache ttl", type: "text", placeholder: "2h" },
        { path: "feeds", label: "feeds", type: "feed-list" },
      ],
    },
    "split-column": {
      options: [
        { path: "max-columns", label: "max columns", type: "number", placeholder: "2" },
        // child ORDER is layout; it lives in the layout editor, not here
        { path: "widgets", label: "nested widgets", type: "widget-list", layout: true },
      ],
    },
  };

  function widgetLabel(id) { return WIDGET_LABELS[id] || id; }

  function describeOptions(w) {
    const opts = w.options || {};
    const bits = [];
    Object.entries(opts).forEach(([k, v]) => {
      if (k === "led") return;
      if (Array.isArray(v)) bits.push(k + "[" + v.length + "]");
      else if (typeof v === "object" && v !== null) bits.push(k + "{…}");
      else bits.push(k + "=" + v);
    });
    return bits.length ? bits.join(" · ") : "options…";
  }

  // ── drag-and-drop: cross-container live reorder ────────────
  // Global drag state, because a widget can be dragged between lists
  // (columns, header, split-children). While dragging, the element is
  // MOVED through the DOM to the slot under the cursor in whichever
  // container it hovers over, so it visibly takes the place it would
  // have on the page — including sliding into another column. On drop
  // the source and target lists are updated; on cancel (dragend without
  // a drop) the involved containers are re-rendered back.
  const dragState = { el: null, item: null, src: null, target: null, dropped: false };

  function dragCleanup() {
    if (dragState.el) dragState.el.classList.remove("dragging");
    dragState.el = null;
    dragState.item = null;
    dragState.src = null;
    dragState.target = null;
    dragState.dropped = false;
  }

  // One-time wiring per container (guarded by data-drag, so it survives
  // re-renders and never stacks listeners). The container must carry
  // __drag = {list, horizontal?, ignoreSel?, onDrop, onCancel}; draggable
  // items must carry __item (the model object).
  function setupLiveDrag(container, itemSel) {
    if (container.dataset.drag) return;
    container.dataset.drag = "1";

    container.addEventListener("dragstart", (e) => {
      const cfg = container.__drag;
      if (!cfg) return;
      if (cfg.ignoreSel && e.target.closest(cfg.ignoreSel)) return;
      const itemEl = e.target.closest(itemSel);
      if (!itemEl) return;
      // Only items that belong to THIS list start a drag here — nested
      // containers (widget rows inside a column card) must not hijack.
      if (cfg.list.indexOf(itemEl.__item) < 0) return;
      dragState.el = itemEl;
      dragState.item = itemEl.__item;
      dragState.src = { container, list: cfg.list };
      dragState.target = null;
      dragState.dropped = false;
      itemEl.classList.add("dragging");
      e.dataTransfer.effectAllowed = "move";
      try { e.dataTransfer.setData("text/plain", ""); } catch (err) { /* old engines */ }
      if (!itemEl.__dragend) {
        itemEl.__dragend = 1;
        itemEl.addEventListener("dragend", () => {
          if (dragState.dropped || !dragState.src) { dragCleanup(); return; }
          // cancel: put everything back where it was
          const srcC = dragState.src.container;
          const tgtC = dragState.target;
          dragCleanup();
          if (srcC.__drag) srcC.__drag.onCancel();
          if (tgtC && tgtC !== srcC && tgtC.__drag) tgtC.__drag.onCancel();
        });
      }
    });

    container.addEventListener("dragover", (e) => {
      if (!dragState.el) return;
      const cfg = container.__drag;
      if (cfg.ignoreSel && e.target.closest(cfg.ignoreSel)) return;
      e.preventDefault();
      e.dataTransfer.dropEffect = "move";
      const horizontal = !!cfg.horizontal;
      // direct children only — nested lists (split-children) reorder
      // within their own container
      const others = Array.from(container.children).filter((x) => x.matches(itemSel) && x !== dragState.el);
      let insertIdx = others.length; // default: append at the end
      for (let i = 0; i < others.length; i++) {
        const r = others[i].getBoundingClientRect();
        const mid = horizontal ? r.left + r.width / 2 : r.top + r.height / 2;
        const pos = horizontal ? e.clientX : e.clientY;
        if (pos < mid) { insertIdx = i; break; }
      }
      // Adopt the element into this container (no-op when it's already
      // here — a foreign container gets it temporarily), then place it
      // at the insertion slot so the others shift aside around it.
      if (dragState.el.parentNode !== container) container.appendChild(dragState.el);
      const ref = insertIdx < others.length ? others[insertIdx] : null;
      if (ref) {
        if (ref !== dragState.el.nextElementSibling) container.insertBefore(dragState.el, ref);
      } else {
        container.appendChild(dragState.el);
      }
      dragState.target = container;
    });

    container.addEventListener("drop", (e) => {
      e.preventDefault();
      if (!dragState.el) return;
      const cfg = container.__drag;
      if (cfg.ignoreSel && e.target.closest(cfg.ignoreSel)) return;
      dragState.dropped = true;
      const srcList = dragState.src.list;
      const targetList = cfg.list;
      // The DOM order in this container (which includes the dragged item
      // at its slot) is the new truth for the target list. Direct
      // children only — nested lists commit to their own container.
      const order = Array.from(container.children).filter((x) => x.matches(itemSel)).map((x) => x.__item);
      if (srcList === targetList) {
        const same = order.length === srcList.length && order.every((x, i) => x === srcList[i]);
        if (!same) {
          srcList.length = 0;
          srcList.push(...order);
          cfg.onDrop();
        }
      } else {
        const item = dragState.item;
        const srcIdx = srcList.indexOf(item);
        if (srcIdx >= 0) srcList.splice(srcIdx, 1);
        targetList.length = 0;
        targetList.push(...order);
        dragState.src.container.__drag.onDrop();
        cfg.onDrop();
      }
      dragCleanup();
    });
  }

  /* ── widget picker popup ───────────────────────────────────
     One anchored menu shared by every [+] button (column heads
     and the header strip). It lives in document.body so layout
     re-renders never destroy it; position: fixed keeps it under
     the button. Closes on outside click, Esc, or after picking. */
  const widgetPicker = (() => {
    const pop = el("div", "widget-picker");
    document.body.appendChild(pop);
    let onPick = null;
    let anchor = null;

    function close() {
      pop.classList.remove("open");
      pop.innerHTML = "";
      onPick = null;
      anchor = null;
      document.removeEventListener("click", outside, true);
      document.removeEventListener("keydown", esc, true);
    }
    function outside(e) {
      if (anchor && anchor.contains(e.target)) return; // re-click of the [+] itself
      if (pop.contains(e.target)) return;
      close();
    }
    function esc(e) {
      if (e.key === "Escape") close();
    }
    function open(a, choices, cb) {
      close();
      anchor = a;
      onPick = cb;
      choices.forEach((c) => {
        const item = el("button", "picker-item");
        item.type = "button";
        item.textContent = c.label;
        item.addEventListener("click", () => {
          const pick = onPick;
          close();
          if (pick) pick(c.id);
        });
        pop.appendChild(item);
      });
      const r = anchor.getBoundingClientRect();
      pop.style.top = r.bottom + 6 + "px";
      pop.style.left = r.left + "px";
      pop.classList.add("open");
      // The [+] sits at the right edge of a column head — clamp the menu
      // horizontally so it never spills off-screen.
      const pr = pop.getBoundingClientRect();
      if (pr.right > window.innerWidth - 8) {
        pop.style.left = Math.max(8, window.innerWidth - pr.width - 8) + "px";
      }
      document.addEventListener("click", outside, true);
      document.addEventListener("keydown", esc, true);
    }
    return { open };
  })();

  // Renders a list of {id, options} widgets into container. ORDER ONLY:
  // options are edited in the widgets section, so rows carry no options
  // button. split-column children are layout too — they render as an
  // always-visible nested list below the row. New widgets are added from
  // the column/header [+] picker — nested lists have no add control; drag
  // an existing widget into a split instead.
  function renderWidgetList(container, list) {
    container.innerHTML = "";
    list.forEach((w, i) => {
      const row = el("div", "widget-row");
      row.draggable = true;
      row.__item = w;
      const grip = el("span", "drag-grip", "⋮⋮");
      grip.title = "drag to reorder";
      row.appendChild(grip);
      row.appendChild(el("span", "w-id", widgetLabel(w.id)));
      const acts = el("div", "row-actions");
      acts.appendChild(iconBtn("×", () => {
        list.splice(i, 1);
        renderWidgetList(container, list);
        refreshWidgetsSection();
        markDirty();
      }, true, "remove"));
      row.appendChild(acts);
      container.appendChild(row);

      if (w.id === "split-column") {
        const opts = w.options || (w.options = {});
        const children = opts.widgets || (opts.widgets = []);
        const sub = el("div", "split-children");
        container.appendChild(sub);
        renderWidgetList(sub, children);
      }
    });

    // .split-children is ignored so a drop over a nested list isn't
    // claimed by this (outer) list — the nested container handles it.
    container.__drag = {
      list,
      ignoreSel: ".split-children, select, input, button",
      onDrop: () => { renderWidgetList(container, list); refreshWidgetsSection(); markDirty(); },
      onCancel: () => { renderWidgetList(container, list); },
    };
    setupLiveDrag(container, ".widget-row");
  }

  // Options form for one widget instance (used by the widgets section).
  // Fields marked layout: true are edited in the layout editor and
  // skipped here.
  function renderWidgetOptions(panel, w) {
    panel.innerHTML = "";
    const schema = WIDGET_SCHEMA[w.id] || {};
    const opts = w.options || (w.options = {});
    const grid = fieldGrid(2);
    renderField(grid, { path: "led", label: "led", type: "color", placeholder: "var(--accent)" }, opts);
    const wide = [];
    (schema.options || []).forEach((f) => {
      if (f.layout) return;
      if (f.type === "feed-list") {
        wide.push(f);
      } else {
        renderField(grid, f, opts);
      }
    });
    panel.appendChild(grid);
    wide.forEach((f) => {
      panel.appendChild(el("div", "option-label", f.label));
      const sub = el("div", "list-editor");
      panel.appendChild(sub);
      renderFeedList(sub, opts[f.path] || (opts[f.path] = []));
    });
  }

  function renderFeedList(container, list) {
    container.innerHTML = "";
    list.forEach((f, i) => {
      const row = el("div", "list-row");
      const urlInp = el("input");
      urlInp.type = "url";
      urlInp.placeholder = "https://example.com/feed.xml";
      urlInp.value = f.url || "";
      urlInp.style.flex = "2";
      urlInp.addEventListener("input", () => { f.url = urlInp.value; markDirty(); });
      const titleInp = el("input");
      titleInp.type = "text";
      titleInp.placeholder = "title";
      titleInp.value = f.title || "";
      titleInp.style.flex = "1";
      titleInp.addEventListener("input", () => { f.title = titleInp.value; markDirty(); });
      const limitInp = el("input");
      limitInp.type = "number";
      limitInp.placeholder = "limit";
      limitInp.style.width = "70px";
      limitInp.value = f.limit == null ? "" : f.limit;
      limitInp.addEventListener("input", () => {
        const n = parseInt(limitInp.value, 10);
        f.limit = isNaN(n) ? 0 : n;
        markDirty();
      });
      const cacheInp = el("input");
      cacheInp.type = "text";
      cacheInp.placeholder = "cache";
      cacheInp.style.width = "80px";
      cacheInp.value = f.cache || "";
      cacheInp.addEventListener("input", () => { f.cache = cacheInp.value; markDirty(); });
      row.appendChild(urlInp);
      row.appendChild(titleInp);
      row.appendChild(limitInp);
      row.appendChild(cacheInp);
      row.appendChild(iconBtn("×", () => { list.splice(i, 1); renderFeedList(container, list); markDirty(); }, true, "remove feed"));
      container.appendChild(row);
    });
    const add = el("button", "btn", "+ add feed");
    add.addEventListener("click", () => {
      list.push({ url: "", title: "" });
      renderFeedList(container, list);
      markDirty();
    });
    container.appendChild(add);
  }

  function renderHeaderList(container, list) {
    container.innerHTML = "";
    list.forEach((w, i) => {
      const chip = el("div", "header-chip");
      chip.draggable = true;
      chip.__item = w;
      const grip = el("span", "drag-grip", "⋮⋮");
      grip.title = "drag to reorder";
      chip.appendChild(grip);
      chip.appendChild(el("span", "w-id", widgetLabel(w.id)));
      const acts = el("div", "row-actions");
      acts.appendChild(iconBtn("×", () => {
        list.splice(i, 1);
        renderHeaderList(container, list);
        refreshWidgetsSection();
        markDirty();
      }, true, "remove"));
      chip.appendChild(acts);
      container.appendChild(chip);
    });

    container.__drag = {
      list,
      horizontal: true,
      ignoreSel: "select, input, button",
      onDrop: () => { renderHeaderList(container, list); refreshWidgetsSection(); markDirty(); },
      onCancel: () => { renderHeaderList(container, list); },
    };
    setupLiveDrag(container, ".header-chip");

    // [+] chip at the end of the strip — opens the header-widget picker.
    const addBtn = el("button", "header-add-btn", "+");
    addBtn.type = "button";
    addBtn.title = "add header widget";
    addBtn.addEventListener("click", () => {
      widgetPicker.open(addBtn, HEADER_WIDGETS, (id) => {
        list.push({ id, options: {} });
        renderHeaderList(container, list);
        refreshWidgetsSection();
        markDirty();
      });
    });
    container.appendChild(addBtn);
  }

  function renderColumns(container) {
    container.innerHTML = "";
    // Render the columns side by side exactly like the page: small = 250px
    // fixed, full = the remaining width.
    const grid = el("div", "layout-grid");
    grid.style.gridTemplateColumns = cfg.columns
      .map((c) => (c.size === "small" ? "250px" : "minmax(0, 1fr)"))
      .join(" ");
    cfg.columns.forEach((col, ci) => {
      const card = el("div", "layout-col");
      card.__item = col;
      const head = el("div", "layout-col-head");
      const grip = el("span", "drag-grip", "⋮⋮");
      grip.title = "drag to reorder columns";
      head.appendChild(grip);
      head.appendChild(el("span", "col-label", "column " + (ci + 1)));
      const sizeSel = el("select", "form-select small");
      [["small", "small"], ["full", "full"]].forEach(([v, l]) => {
        const o = el("option");
        o.value = v;
        o.textContent = l;
        sizeSel.appendChild(o);
      });
      sizeSel.value = col.size || "full";
      sizeSel.addEventListener("change", () => {
        col.size = sizeSel.value;
        renderColumns(container); // column widths change with the size
        markDirty();
      });
      head.appendChild(sizeSel);
      const acts = el("div", "row-actions");
      const addBtn = iconBtn("+", () => {
        widgetPicker.open(addBtn, BODY_WIDGETS, (id) => {
          (col.widgets || (col.widgets = [])).push({ id, options: {} });
          renderColumns(container);
          refreshWidgetsSection();
          markDirty();
        });
      }, false, "add widget");
      acts.appendChild(addBtn);
      acts.appendChild(iconBtn("×", () => { cfg.columns.splice(ci, 1); renderColumns(container); markDirty(); }, true, "remove column"));
      head.appendChild(acts);
      card.appendChild(head);
      const listWrap = el("div", "layout-col-widgets");
      card.appendChild(listWrap);
      renderWidgetList(listWrap, col.widgets || (col.widgets = []));
      grid.appendChild(card);
    });
    container.appendChild(grid);

    // Columns reorder horizontally, like the page's left-to-right order.
    // Drags that start on a column's widget stack (or on form controls)
    // are ignored — grab a column by its head/grip.
    container.__drag = {
      list: cfg.columns,
      horizontal: true,
      ignoreSel: ".widget-row, .layout-col-widgets, select, input, button",
      onDrop: () => { renderColumns(container); markDirty(); },
      onCancel: () => { renderColumns(container); },
    };
    setupLiveDrag(container, ".layout-col");
  }

  /* ── images ──────────────────────────────────────────────── */
  function renderImageGrid(kind) {
    const container = imgContainers[kind];
    if (!container) return;
    container.innerHTML = "";
    const grid = el("div", "img-grid");
    (images[kind] || []).forEach((img) => {
      const slot = el("div", "img-slot filled" + (kind === "bg" ? " bg-slot" : ""));
      const im = el("img");
      im.src = img.url;
      im.alt = img.name;
      const rm = el("button", "slot-remove", "×");
      rm.title = "delete " + img.name;
      rm.addEventListener("click", (e) => {
        e.stopPropagation();
        deleteImage(kind, img.name);
      });
      slot.appendChild(im);
      slot.appendChild(rm);
      grid.appendChild(slot);
    });
    const addSlot = el("div", "img-slot" + (kind === "bg" ? " bg-slot" : ""));
    addSlot.appendChild(el("div", "slot-plus", "+"));
    addSlot.appendChild(el("div", "slot-label", kind));
    const fileInp = el("input");
    fileInp.type = "file";
    fileInp.accept = "image/*";
    fileInp.style.display = "none";
    fileInp.addEventListener("change", () => {
      if (fileInp.files[0]) uploadImage(kind, fileInp.files[0]);
      fileInp.value = "";
    });
    addSlot.appendChild(fileInp);
    addSlot.addEventListener("click", () => fileInp.click());
    grid.appendChild(addSlot);
    container.appendChild(grid);
  }

  async function refreshImages() {
    try {
      const res = await fetch("/api/settings");
      const data = await res.json();
      images = data.images || { profile: [], bg: [], widget: [] };
      ["profile", "bg", "widget"].forEach(renderImageGrid);
    } catch (err) {
      setStatus("could not refresh images: " + err.message, "err");
    }
  }

  async function uploadImage(kind, file) {
    const fd = new FormData();
    fd.append("file", file);
    try {
      const res = await fetch("/api/settings/images?type=" + kind, { method: "POST", body: fd });
      if (!res.ok) throw new Error((await res.text()) || res.status);
      setStatus("image uploaded", "ok");
    } catch (err) {
      setStatus("upload failed: " + err.message, "err");
    }
    await refreshImages();
  }

  async function deleteImage(kind, name) {
    try {
      const res = await fetch("/api/settings/images?type=" + kind + "&name=" + encodeURIComponent(name), { method: "DELETE" });
      if (!res.ok) throw new Error((await res.text()) || res.status);
      setStatus("image deleted", "ok");
    } catch (err) {
      setStatus("delete failed: " + err.message, "err");
    }
    await refreshImages();
  }

  /* ── quotes ──────────────────────────────────────────────── */
  function renderQuotes(container) {
    container.innerHTML = "";
    quotesWork.forEach((q, i) => {
      const row = el("div", "list-row");
      const main = el("div", "row-main");
      main.appendChild(el("div", "row-title", q.text || "(no text)"));
      main.appendChild(el("div", "row-sub", q.author || ""));
      row.appendChild(main);
      const acts = el("div", "row-actions");
      acts.appendChild(iconBtn("×", () => {
        quotesWork.splice(i, 1);
        markQuotesDirty();
        renderQuotes(container);
      }, true, "delete quote"));
      row.appendChild(acts);
      container.appendChild(row);
    });
    const form = el("div", "add-form");
    const textInp = el("input");
    textInp.type = "text";
    textInp.placeholder = "quote text";
    const authInp = el("input");
    authInp.type = "text";
    authInp.placeholder = "author";
    const addQ = el("button", "btn primary", "add");
    addQ.addEventListener("click", () => {
      const t = textInp.value.trim();
      if (!t) return;
      quotesWork.push({ text: t, author: authInp.value.trim() });
      markQuotesDirty();
      renderQuotes(container);
    });
    const btns = el("div", "form-btns");
    btns.appendChild(addQ);
    form.appendChild(textInp);
    form.appendChild(authInp);
    form.appendChild(btns);
    container.appendChild(form);
    const addBtn = el("button", "btn", "+ add quote");
    addBtn.addEventListener("click", () => form.classList.toggle("visible"));
    container.appendChild(addBtn);
  }

  async function saveQuotes() {
    try {
      const res = await fetch("/api/settings/quotes", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ quotes: quotesWork }),
      });
      if (!res.ok) throw new Error((await res.text()) || res.status);
      quotes = clone(quotesWork);
      quotesDirty = false;
      if (saveQuotesBtn) saveQuotesBtn.style.display = "none";
      setStatus("quotes saved", "ok");
    } catch (err) {
      setStatus("quote save failed: " + err.message, "err");
    }
  }

  /* ── sections ────────────────────────────────────────────── */
  function buildIdentity(body) {
    const grid = fieldGrid(2);
    renderField(grid, { path: "username", label: "display name", type: "text", placeholder: "your name" });
    renderField(grid, { path: "location.city", label: "city", type: "text", placeholder: "e.g. Tokyo" });
    renderField(grid, { path: "location.lat", label: "lat", type: "text", placeholder: "latitude" });
    renderField(grid, { path: "location.lon", label: "lon", type: "text", placeholder: "longitude" });
    body.appendChild(grid);
    body.appendChild(el("div", "form-hint", "display name is shown in the header; city is the weather label and coordinates are the fallback for open-meteo"));
  }

  function buildAppearance(body) {
    body.appendChild(el("div", "option-label", "theme"));
    const themeGrid = fieldGrid(2);
    renderField(themeGrid, { path: "theme", label: "base theme", type: "segmented", options: [["dark", "dark"], ["light", "light"]], onChange: applyTheme });
    const customRow = el("div", "form-row");
    customRow.appendChild(el("label", null, "custom theme"));
    const sel = el("select", "form-select");
    const none = el("option");
    none.value = "";
    none.textContent = "none";
    sel.appendChild(none);
    themes.forEach((t) => {
      const o = el("option");
      o.value = t;
      o.textContent = t.replace(/\.css$/, "");
      sel.appendChild(o);
    });
    sel.value = cfg.theme_file || "";
    sel.addEventListener("change", () => {
      cfg.theme_file = sel.value;
      applyTheme();
      markDirty();
    });
    customRow.appendChild(sel);
    themeGrid.appendChild(customRow);
    body.appendChild(themeGrid);

    body.appendChild(el("div", "subsection-label", "fonts"));
    const fontGrid = fieldGrid(3);
    renderField(fontGrid, { path: "font_latin", label: "latin font", type: "select", options: [["inter", "inter"]] });
    renderField(fontGrid, { path: "font_jp", label: "japanese fallback", type: "select", options: [["dotgothic16", "dotgothic16"]] });
    renderField(fontGrid, { path: "font_clock", label: "clock font", type: "select", options: [["orbitron", "orbitron"]] });
    renderField(fontGrid, { path: "ui_lang", label: "ui language", type: "select", options: [["en", "English"], ["ja", "日本語"]] });
    renderField(fontGrid, { path: "title_lang", label: "widget title lang", type: "segmented", options: [["ja", "ja"], ["en", "en"]] });
    body.appendChild(fontGrid);

    body.appendChild(el("div", "subsection-label", "clock"));
    const clockGrid = fieldGrid(2);
    renderField(clockGrid, { path: "clock_format", label: "format", type: "segmented", options: [["24h", "24h"], ["12h", "12h"]] });
    renderField(clockGrid, { path: "clock_seconds", label: "seconds", type: "toggle" });
    renderField(clockGrid, { path: "clock_theme", label: "color mode", type: "segmented", options: [["theme", "follow theme"], ["light", "force light"], ["dark", "force dark"]], wide: true });
    body.appendChild(clockGrid);

    body.appendChild(el("div", "subsection-label", "background"));
    const bgGrid = fieldGrid(2);
    renderField(bgGrid, { path: "bg_blur", label: "blur", type: "toggle" });
    renderField(bgGrid, { path: "default_led", label: "default led", type: "color", placeholder: "var(--accent)" });
    body.appendChild(bgGrid);
  }

  function buildLayout(body) {
    cfg.columns = cfg.columns || [];
    body.appendChild(el("div", "subsection-label", "header"));
    const headerWrap = el("div", "layout-header-row");
    body.appendChild(headerWrap);
    renderHeaderList(headerWrap, cfg.header || (cfg.header = []));
    body.appendChild(el("div", "form-hint", "the header is one row, left to right — drag to reorder; add header widgets with the [+] at the end of the strip. This section only sets ORDER; each widget's options live in the widgets section."));

    body.appendChild(el("div", "subsection-label", "columns"));
    const colsWrap = el("div");
    body.appendChild(colsWrap);
    renderColumns(colsWrap);
    const addCol = el("button", "btn", "+ add column");
    addCol.addEventListener("click", () => {
      cfg.columns.push({ size: "full", widgets: [] });
      renderColumns(colsWrap);
      refreshWidgetsSection();
      markDirty();
    });
    body.appendChild(addCol);
    body.appendChild(el("div", "form-hint", "columns sit side by side like the page — small = 250px, full = remaining width. Add widgets with the [+] in each column head; drag them between columns. split-column has no add — drop a widget into its nested list instead. Only ORDER is configured here — click a widget in the widgets section to configure it."));
  }

  // Every configured widget instance (header + columns, recursing into
  // split-column children) gets its own options form here — the layout
  // section only handles order.
  function collectWidgetInstances() {
    const out = [];
    const push = (w, loc) => out.push({ w, loc });
    const walk = (list, locPrefix) => {
      (list || []).forEach((w) => {
        push(w, locPrefix);
        if (w.id === "split-column" && w.options && Array.isArray(w.options.widgets)) {
          walk(w.options.widgets, locPrefix + " · split");
        }
      });
    };
    (cfg.header || []).forEach((w) => push(w, "header"));
    cfg.columns.forEach((col, ci) => walk(col.widgets || [], "column " + (ci + 1)));
    return out;
  }

  function buildWidgetInstances(body) {
    body.innerHTML = "";
    const instances = collectWidgetInstances();
    if (instances.length === 0) {
      body.appendChild(el("div", "form-hint", "no widgets configured — add them in the layout section"));
      return;
    }
    const listEl = el("div", "list-editor");
    instances.forEach(({ w, loc }) => {
      const row = el("div", "list-row");
      const main = el("div", "row-main");
      main.appendChild(el("div", "row-title", widgetLabel(w.id)));
      const hasOpts = w.options && Object.keys(w.options).length > 0;
      main.appendChild(el("div", "row-sub", loc + (hasOpts ? " · " + describeOptions(w) : "")));
      row.appendChild(main);
      const panel = el("div", "widget-options");
      const toggle = el("button", "w-opts-btn", "options…");
      toggle.addEventListener("click", () => {
        panel.classList.toggle("visible");
        if (panel.classList.contains("visible") && !panel.dataset.rendered) {
          renderWidgetOptions(panel, w);
          panel.dataset.rendered = "1";
        }
      });
      row.appendChild(toggle);
      listEl.appendChild(row);
      listEl.appendChild(panel);
    });
    body.appendChild(listEl);
  }

  function refreshWidgetsSection() {
    if (widgetsSectionBody) buildWidgetInstances(widgetsSectionBody);
  }

  function buildWidgets(body) {
    body.appendChild(el("div", "subsection-label", "widget options"));
    widgetsSectionBody = el("div");
    body.appendChild(widgetsSectionBody);
    buildWidgetInstances(widgetsSectionBody);
    body.appendChild(el("div", "form-hint", "one entry per configured widget — where it lives plus its options. Order is set in the layout section."));

    body.appendChild(el("div", "subsection-label", "search"));
    const searchGrid = fieldGrid(2);
    renderField(searchGrid, { path: "search_target", label: "open results", type: "segmented", options: [["_self", "same tab"], ["_blank", "new tab"]] });
    renderField(searchGrid, { path: "jlpt_level", label: "jlpt filter", type: "segmented", options: [["n1", "N1"], ["n2", "N2"], ["n3", "N3"], ["n4", "N4"], ["n5", "N5"], ["all", "ALL"]] });
    body.appendChild(searchGrid);
    body.appendChild(el("div", "option-label", "engines"));
    const engineWrap = el("div", "list-editor");
    body.appendChild(engineWrap);
    renderEngines(engineWrap);

    body.appendChild(el("div", "subsection-label", "footer"));
    const footWrap = el("div");
    body.appendChild(footWrap);
    renderFooter(footWrap);
  }

  function buildData(body) {
    body.appendChild(el("div", "subsection-label", "quotes"));
    const quoteWrap = el("div", "list-editor");
    body.appendChild(quoteWrap);
    renderQuotes(quoteWrap);
    saveQuotesBtn = el("button", "btn primary", "save quotes");
    saveQuotesBtn.style.display = "none";
    saveQuotesBtn.addEventListener("click", saveQuotes);
    body.appendChild(saveQuotesBtn);
    body.appendChild(el("div", "form-hint", "quotes are used by the quote widget when the database is empty. Saved to data/quotes.yaml."));

    body.appendChild(el("div", "subsection-label", "profile images"));
    imgContainers.profile = el("div");
    body.appendChild(imgContainers.profile);
    renderImageGrid("profile");
    body.appendChild(el("div", "img-hint", "shown randomly as the header avatar"));

    body.appendChild(el("div", "subsection-label", "background images"));
    imgContainers.bg = el("div");
    body.appendChild(imgContainers.bg);
    renderImageGrid("bg");
    body.appendChild(el("div", "img-hint", "shown randomly as the page background"));

    body.appendChild(el("div", "subsection-label", "image widget images"));
    imgContainers.widget = el("div");
    body.appendChild(imgContainers.widget);
    renderImageGrid("widget");
    body.appendChild(el("div", "img-hint", "cycled by the image widget"));
  }

  const SECTIONS = [
    { label: "identity", bar: "blue", build: buildIdentity },
    { label: "appearance", bar: "pink", build: buildAppearance },
    { label: "layout", bar: "green", build: buildLayout },
    { label: "widgets", bar: "green", build: buildWidgets },
    { label: "data", bar: "red", build: buildData },
  ];

  function renderAll() {
    sectionsEl.innerHTML = "";
    SECTIONS.forEach((s) => {
      const section = el("div", "section");
      const title = el("div", "section-title");
      title.appendChild(el("div", "st-bar " + s.bar));
      title.appendChild(el("span", "st-label", s.label));
      section.appendChild(title);
      const body = el("div", "section-body");
      s.build(body);
      section.appendChild(body);
      sectionsEl.appendChild(section);
    });
  }

  /* ── save / reset / init ─────────────────────────────────── */
  async function save() {
    saveBtn.disabled = true;
    setStatus("saving…");
    try {
      const res = await fetch("/api/settings", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(cfg),
      });
      const txt = await res.text();
      let data = {};
      try { data = txt ? JSON.parse(txt) : {}; } catch (e) { /* non-JSON error */ }
      if (!res.ok) throw new Error(data.error || res.statusText || res.status);
      cfg = data.config;
      renderAll();
      applyTheme();
      setStatus("saved — applied to the new tab", "ok");
    } catch (err) {
      saveBtn.disabled = false;
      setStatus("save failed: " + err.message, "err");
    }
  }

  async function reset() {
    if (!confirm("Reset all settings to defaults?")) return;
    try {
      const res = await fetch("/api/settings/reset", { method: "POST" });
      const txt = await res.text();
      let data = {};
      try { data = txt ? JSON.parse(txt) : {}; } catch (e) { /* ignore */ }
      if (!res.ok) throw new Error(data.error || res.statusText || res.status);
      cfg = data.config;
      renderAll();
      applyTheme();
      setStatus("reset to defaults", "ok");
    } catch (err) {
      setStatus("reset failed: " + err.message, "err");
    }
  }

  async function init() {
    try {
      const res = await fetch("/api/settings");
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      cfg = data.config;
      quotes = data.quotes || [];
      quotesWork = clone(quotes);
      themes = data.themes || [];
      images = data.images || { profile: [], bg: [], widget: [] };
      applyTheme();
      renderAll();
      setStatus("loaded");
    } catch (err) {
      setStatus("failed to load settings: " + err.message, "err");
    }
  }

  saveBtn.addEventListener("click", save);
  resetBtn.addEventListener("click", reset);
  init();
})();
