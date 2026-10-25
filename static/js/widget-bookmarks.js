/* Bookmarks — htmx-driven folder navigation. The server renders every
   view (root listing or the inside-folder view) from the single source
   of truth; the client only collects dialog input, reads the current
   state + folder from the widget's data attributes, and asks htmx to
   POST the new state and swap in the re-rendered widget. Entering a
   folder and going back are pure htmx (hx-get) — zero JS.             */
(function() {
  var SEL = '.widget[data-widget="bookmarks"]';

  function widget() { return document.querySelector(SEL); }

  function state() {
    var root = widget();
    if (!root || !root.dataset.state) return [];
    try { return JSON.parse(root.dataset.state); } catch (e) { return []; }
  }

  // The folder currently being viewed: -1 = root listing. Read from the
  // server-rendered list element, so it survives htmx swaps.
  function currentFolder() {
    var root = widget();
    if (!root) return -1;
    var list = root.querySelector("#folder-list");
    var n = list && parseInt(list.dataset.folder, 10);
    return isNaN(n) ? -1 : n;
  }

  // The scrollable area of the current view: the links grid inside a
  // folder. Its scroll position is preserved across the htmx re-render
  // so edits deep in a long list don't jump to the top.
  function scrollContainer() {
    var root = widget();
    if (!root) return null;
    return root.querySelector(".folder-links.grid");
  }

  var pendingScrollRestore = false;
  var savedScrollTop = 0;

  // Height is fixed per the config "rows" option (default 3): the
  // server emits --bm-rows on the widget body (see newtab.css) and
  // every view — root listing or folder — gets the same fixed-height
  // box, scrolling internally when content overflows. Nothing in JS
  // measures or sizes the widget.

  function saveAndRender(next) {
    var root = widget();
    if (!root) return;
    var sc = scrollContainer();
    if (sc) {
      savedScrollTop = sc.scrollTop;
      pendingScrollRestore = true;
    }
    htmx.ajax("POST", "/api/widget/bookmarks", {
      values: { state: JSON.stringify(next), folder: currentFolder() },
      target: SEL,
      swap: "outerHTML"
    });
  }

  // ── Swap rendering: off-screen first ─────────────────────────────
  // Every view change (enter a folder, go back, CRUD re-render) is an
  // htmx swap of the whole widget. Firefox paints the freshly-inserted
  // favicon tiles once in their pre-layout state — icons at the top-left
  // of each tile, tiles not yet padded/sized — before the flex/grid
  // layout applies, which reads as "icons slide to the center while the
  // tile grows" on every swap.
  //
  // Fix: render the htmx response OFF-SCREEN first and force a style +
  // layout pass there; only then swap the settled widget into place.
  // By the time it is visible, the layout is final — there is no
  // pre-layout frame left to flash. The swap is atomic: the old widget
  // is replaced by the fully-laid-out new one in a single step.
  //
  // Second flash source: the favicon IMAGES themselves. Layout can be
  // forced synchronously, but the downloads finish asynchronously — an
  // uncached icon pops in over the visible widget a moment after the
  // swap (cached icons are instant, so the flash only shows the first
  // time a folder is opened). Fix: hold the swap until every favicon
  // has loaded or failed in the off-screen holder (images load there
  // normally), capped by a timeout so a slow favicon can't hang the
  // swap.
  document.addEventListener("htmx:beforeSwap", function(e) {
    var detail = e.detail || {};
    var target = detail.target;
    if (!target || !target.matches || !target.matches(SEL)) return;
    var html = detail.serverResponse;
    if (typeof html !== "string" || !html || !target.parentNode) return;
    e.preventDefault();

    // 1. Build the new widget off-screen at the same width and force a
    //    synchronous layout pass so its styles are fully applied.
    var holder = document.createElement("div");
    holder.setAttribute("aria-hidden", "true");
    holder.style.cssText =
      "position:fixed;top:0;left:-10000px;width:" + Math.max(target.offsetWidth, 1) +
      "px;visibility:hidden;pointer-events:none";
    holder.innerHTML = html;
    document.body.appendChild(holder);
    void holder.offsetHeight; // forced reflow: new subtree is styled + laid out

    var fresh = holder.querySelector(SEL);
    if (!fresh || !target.parentNode) {
      holder.remove();
      return;
    }

    // 2. Swap once the favicons are ready (loaded, failed, or timed
    //    out) — see the note above.
    var imgs = fresh.querySelectorAll("img.fav");
    var pending = imgs.length;
    var swapped = false;
    function swap() {
      if (swapped) return;
      swapped = true;
      if (!target.parentNode) {
        holder.remove();
        return;
      }

      // 3. Atomic swap: the settled widget replaces the old one.
      target.parentNode.replaceChild(fresh, target);
      holder.remove();

      // 4. Scroll restore for POST re-renders (handled here because the
      //    default swap — and its afterSwap event — was prevented).
      if (pendingScrollRestore) {
        pendingScrollRestore = false;
        var sc = scrollContainer();
        if (sc) sc.scrollTop = savedScrollTop;
      }

      // 5. Do what htmx would have done: process the new subtree's hx-*
      //    attributes and fire htmx:load so the CRUD handlers bind.
      if (window.htmx && htmx.process) htmx.process(fresh);
      if (window.htmx && htmx.trigger) htmx.trigger(fresh, "htmx:load");
    }

    if (pending === 0) {
      swap();
    } else {
      // complete covers both loaded and failed images, so a broken
      // favicon never blocks the swap.
      imgs.forEach(function(img) {
        if (img.complete) { pending--; return; }
        img.addEventListener("load", function() {
          pending--;
          if (pending === 0) swap();
        });
        img.addEventListener("error", function() {
          pending--;
          if (pending === 0) swap();
        });
      });
      if (pending === 0) {
        swap();
      } else {
        setTimeout(swap, 1500);
      }
    }
  });

  // After a drag ends, browsers may fire a click on the dragged row —
  // that must not trigger htmx navigation into the folder. We remember
  // the row that was dragged and swallow only clicks landing on it, so
  // normal clicks on other rows are never affected.
  var lastDrag = null;
  function guardNav(e) {
    if (lastDrag && lastDrag.contains(e.target)) e.stopImmediatePropagation();
  }

  function init() {
    var root = widget();
    if (!root || root.dataset.hxInit === "1") return;
    root.dataset.hxInit = "1";

    var fi = currentFolder();
    var list = root.querySelector("#folder-list");

    if (list) {
      list.addEventListener("dragstart", function(e) {
        lastDrag = e.target.closest(".folder-tile[data-drag-index]");
      });
      // tiles (root) and the ".." up row (folder view) both navigate
      list.querySelectorAll(".folder-tile[hx-get], .folder-up[hx-get]").forEach(function(row) {
        row.addEventListener("click", guardNav, true);
        // keyboard: Enter/Space activates the row's hx-get navigation
        row.addEventListener("keydown", function(e) {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            row.click();
          }
        });
      });
    }

    // Title-bar "+": add a folder at root, add a link inside a folder.
    var addBtn = root.querySelector("#bm-add-folder-btn");
    if (addBtn) {
      addBtn.addEventListener("click", async function() {
        if (fi < 0) {
          var r = await miniPrompt([{ key: "name", placeholder: "folder name" }]);
          if (!r || !r.name) return;
          var next = state();
          next.push({ folder: r.name, links: [] });
          saveAndRender(next);
        } else {
          var r = await linkPrompt();
          if (!r || !r.label || !r.url) return;
          var next = state();
          if (next[fi]) next[fi].links.push({ label: r.label, url: r.url, fav: r.fav });
          saveAndRender(next);
        }
      });
    }

    if (fi < 0) {
      // root view — folder tiles: rename/delete + drag-reorder
      list.querySelectorAll(".folder-tile[data-drag-index]").forEach(function(tile, i) {
        var edit = tile.querySelector(".tile-edit");
        edit.addEventListener("click", async function(e) {
          e.preventDefault(); e.stopPropagation();
          var cur = state()[i] || {};
          var r = await miniPrompt([{ key: "name", placeholder: "folder name" }],
            { name: cur.folder }, { deletable: true });
          if (!r) return;
          var next = state();
          if (r === "delete") next.splice(i, 1);
          else if (r.name) next[i].folder = r.name;
          saveAndRender(next);
        });
        edit.addEventListener("mousedown", function(e) { e.stopImmediatePropagation(); });
      });
      enableDragSort(list, ".folder-tile[data-drag-index]",
        function() { return state(); },
        function() {},
        function(arr) { saveAndRender(arr); });
    } else {
      // inside a folder — link tiles: edit/delete + drag-reorder
      list.querySelectorAll(".folder-links a[data-drag-index]").forEach(function(a, li) {
        // Favicon fallback: if the image fails to load or decode (missing
        // file, HTML saved as a favicon, blocked host), swap in a letter
        // tile so the bookmark stays readable instead of a broken glyph.
        var img = a.querySelector("img.fav");
        if (img) {
          img.addEventListener("error", function() {
            var span = a.querySelector("span");
            var letter = document.createElement("div");
            letter.className = "fav-letter";
            letter.textContent = (span && span.textContent ? span.textContent[0] : "?").toUpperCase();
            img.replaceWith(letter);
          });
        }
        var edit = a.querySelector(".tile-edit");
        edit.addEventListener("click", async function(e) {
          e.preventDefault(); e.stopImmediatePropagation();
          var cur = state()[fi] && state()[fi].links[li];
          if (!cur) return;
          var r = await linkPrompt({ label: cur.label, url: cur.url, fav: cur.fav });
          if (!r) return;
          var next = state();
          if (r === "delete") next[fi].links.splice(li, 1);
          else next[fi].links[li] = { label: r.label, url: r.url, fav: r.fav };
          saveAndRender(next);
        });
        edit.addEventListener("mousedown", function(e) { e.stopImmediatePropagation(); });
      });
      var linksDiv = list.querySelector(".folder-links");
      if (linksDiv) {
        enableDragSort(linksDiv, "a[data-drag-index]",
          function() { return state()[fi].links || []; },
          function() {},
          function(arr) {
            var next = state();
            if (next[fi]) next[fi].links = arr;
            saveAndRender(next);
          });
      }
    }
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
  document.addEventListener("htmx:load", function(e) {
    if (e.target.matches && e.target.matches(SEL)) init();
  });
})();
