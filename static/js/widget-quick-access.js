/* Quick access CRUD — htmx-driven, same pattern as bookmarks: dialogs
   collect input, state is read from data-state, htmx POSTs the new
   state and swaps in the server-rendered widget.                    */
(function() {
  var SEL = '.widget[data-widget="quick-access"]';

  function widget() { return document.querySelector(SEL); }

  function state() {
    var root = widget();
    if (!root || !root.dataset.state) return [];
    try { return JSON.parse(root.dataset.state); } catch (e) { return []; }
  }

  function saveAndRender(next) {
    if (!widget()) return;
    htmx.ajax("POST", "/api/widget/quick-access", {
      values: { state: JSON.stringify(next) },
      target: SEL,
      swap: "outerHTML"
    });
  }

  function init() {
    var root = widget();
    if (!root || root.dataset.hxInit === "1") return;
    root.dataset.hxInit = "1";

    var addBtn = root.querySelector("#qa-add-btn");
    if (addBtn) {
      addBtn.addEventListener("click", async function() {
        var r = await linkPrompt();
        if (!r || !r.label || !r.url) return;
        var next = state();
        next.push({ label: r.label, url: r.url, favicon: r.fav || "" });
        saveAndRender(next);
      });
    }

    root.querySelectorAll(".quick-links a[data-drag-index]").forEach(function(a, i) {
      var edit = a.querySelector(".qa-edit");
      edit.addEventListener("click", async function(e) {
        e.preventDefault(); e.stopPropagation();
        var cur = state()[i] || {};
        var r = await linkPrompt({ label: cur.label, url: cur.url, fav: cur.favicon || cur.fav || "" });
        if (!r) return;
        var next = state();
        if (r === "delete") next.splice(i, 1);
        else if (r.label && r.url) next[i] = { label: r.label, url: r.url, favicon: r.fav || "" };
        saveAndRender(next);
      });
    });

    var container = root.querySelector(".quick-links");
    if (container) {
      enableDragSort(container, "a[data-drag-index]",
        function() { return state(); },
        function() {},
        function(arr) { saveAndRender(arr); });
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
