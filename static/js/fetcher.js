/* ── fetcher.js ─────────────────────────────────────────────────
   Click → fetch JSON → update DOM. [data-fetch] triggers fetch;
   response keys map to [data-fetch-key] elements within the widget.
   For custom rendering, set data-fetch-render="fnName".          */
(function() {
  document.addEventListener("click", function(e) {
    var btn = e.target.closest("[data-fetch]");
    if (!btn) return;
    if (btn.hasAttribute("data-preload")) return;
    var url = btn.getAttribute("data-fetch");
    var render = window[btn.getAttribute("data-fetch-render")];
    fetch(url).then(function(r) { return r.json(); }).then(function(data) {
      if (render) { render(data, btn); return; }
      var widget = btn.closest(".widget");
      if (!widget) return;
      Object.keys(data).forEach(function(key) {
        var el = widget.querySelector("[data-fetch-key='" + key + "']");
        if (el) el.textContent = data[key];
      });
    }).catch(function() {});
  });
})();
