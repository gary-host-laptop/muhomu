/* ── preloader.js ──────────────────────────────────────────────
   Preloads content on page load, swaps cached data on click,
   then preloads the next. Attributes: data-preload (URL),
   data-preload-render (callback), data-preload-image (key in
   response for img preload), data-preload-init (fn for initial
   URL). Stateless APIs re-fetch same URL; stateful callbacks
   return the next URL to preload.                               */
(function() {
  "use strict";

  var cache = new WeakMap();

  document.addEventListener("click", function(e) {
    var btn = e.target.closest("[data-preload]");
    if (!btn) return;
    var data = cache.get(btn);
    cache.delete(btn);
    if (data) {
      renderThenPreload(btn, data);
    } else {
      fetchThenPreload(btn);
    }
  });

  function renderThenPreload(btn, data) {
    var nextUrl = render(btn, data);
    preloadNext(btn, nextUrl || btn.getAttribute("data-preload"));
  }

  function fetchThenPreload(btn) {
    var url = btn.getAttribute("data-preload");
    if (!url) return;
    fetch(url)
      .then(function(r) { return r.json(); })
      .then(function(data) { renderThenPreload(btn, data); })
      .catch(function() {});
  }

  function render(btn, data) {
    var renderFn = window[btn.getAttribute("data-preload-render")];
    if (renderFn) return renderFn(data, btn);
    var widget = btn.closest(".widget");
    if (!widget) return;
    Object.keys(data).forEach(function(key) {
      var el = widget.querySelector("[data-fetch-key='" + key + "']");
      if (el) el.textContent = data[key];
    });
  }

  function preloadNext(btn, url) {
    if (!url) return;
    fetch(url)
      .then(function(r) { return r.json(); })
      .then(function(data) {
        cache.set(btn, data);
        var imgKey = btn.getAttribute("data-preload-image");
        if (imgKey && data[imgKey]) {
          var img = new Image();
          img.src = data[imgKey];
        }
      })
      .catch(function() {});
  }

  function init() {
    document.querySelectorAll("[data-preload]").forEach(function(btn) {
      var initFn = window[btn.getAttribute("data-preload-init")];
      var url = initFn ? initFn(btn) : btn.getAttribute("data-preload");
      if (url) preloadNext(btn, url);
    });
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
