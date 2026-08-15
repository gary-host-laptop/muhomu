/* Recently visited — htmx-driven: quick-add / delete mutate the state
   read from data-state, htmx POSTs it and swaps in the server-rendered
   widget. Tiles navigate on click.                                    */
(function() {
  var SEL = '.widget[data-widget="recently-visited"]';

  function widget() { return document.querySelector(SEL); }

  function state() {
    var root = widget();
    if (!root || !root.dataset.state) return [];
    try { return JSON.parse(root.dataset.state); } catch (e) { return []; }
  }

  function saveAndRender(next) {
    if (!widget()) return;
    htmx.ajax("POST", "/api/widget/recently-visited", {
      values: { state: JSON.stringify(next) },
      target: SEL,
      swap: "outerHTML"
    });
  }

  function init() {
    var root = widget();
    if (!root || root.dataset.hxInit === "1") return;
    root.dataset.hxInit = "1";

    var grid = root.querySelector("#recent-grid");
    if (grid) {
      grid.querySelectorAll(".recent-tile").forEach(function(tile, i) {
        tile.addEventListener("click", function() {
          var item = state()[i];
          if (item && item.url) window.location.href = item.url;
        });
        var x = tile.querySelector(".recent-x");
        x.addEventListener("click", async function(e) {
          e.stopPropagation();
          var next = state();
          next.splice(i, 1);
          saveAndRender(next);
        });
      });
    }

    var nameInp = root.querySelector("#rqa-name");
    var urlInp = root.querySelector("#rqa-url");
    var btn = root.querySelector("#rqa-btn");

    async function quickAdd() {
      var name = (nameInp && nameInp.value.trim()) || "";
      var url = (urlInp && urlInp.value.trim()) || "";
      if (!name || !url) return;
      var next = state();
      next.unshift({ name: name, url: url });
      if (next.length > 8) next.pop();
      saveAndRender(next);
      if (nameInp) nameInp.value = "";
      if (urlInp) urlInp.value = "";
    }
    if (btn) btn.addEventListener("click", quickAdd);
    if (urlInp) urlInp.addEventListener("keydown", function(e) {
      if (e.key === "Enter") quickAdd();
    });
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
