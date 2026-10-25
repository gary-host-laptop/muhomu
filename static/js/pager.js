/* ── pager.js ──────────────────────────────────────────────────
   Prev/next pagination for pre-rendered pages. [data-pager] on
   buttons, [data-page] on page divs, [data-pager-indicator] on
   counter, all within a [data-pager-wrap] container.            */
(function() {
  // ── Indicator width lock ──────────────────────────────────────
  // The RSS page indicator uses a proportional font (Orbitron): "1 / 5"
  // is narrower than "5 / 5". The pager group is pinned to the
  // titlebar's right edge, so each time the page number changes the
  // growing indicator shoves the prev arrow left. Freeze the indicator
  // at the width of its widest possible text ("N / N") so the arrows
  // never move. Measured after fonts are ready; re-run on htmx:load in
  // case the widget is ever re-rendered.
  function lockIndicator(ind, force) {
    if (!ind) return;
    // force: the DOM-ready pass may have measured with a fallback font
    // before Orbitron loaded — clear and re-measure.
    if (force) ind.style.width = "";
    if (ind.style.width) return;
    var m = /(\d+)\s*\/\s*(\d+)/.exec(ind.textContent || "");
    if (!m) return;
    var cur = ind.textContent;
    ind.textContent = m[2] + " / " + m[2]; // widest text
    ind.style.width = ind.offsetWidth + "px";
    ind.textContent = cur;
  }
  function lockAll(force) {
    [].forEach.call(document.querySelectorAll("[data-pager-indicator]"), function(ind) {
      lockIndicator(ind, force);
    });
  }
  // The digits are proportional, so the width must be measured with the
  // real font loaded — re-lock once fonts are ready, whichever pass ran
  // first. Without document.fonts (older browsers) the DOM-ready pass is
  // the only one.
  if (document.fonts && document.fonts.ready) {
    document.fonts.ready.then(function() { lockAll(true); });
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", function() { lockAll(); });
  } else {
    lockAll();
  }
  document.addEventListener("htmx:load", function(e) {
    var root = e.target;
    if (root && root.querySelectorAll) {
      [].forEach.call(root.querySelectorAll("[data-pager-indicator]"), lockIndicator);
    }
  });

  document.addEventListener("click", function(e) {
    var btn = e.target.closest("[data-pager]");
    if (!btn) return;
    var dir = btn.getAttribute("data-pager");
    // Buttons may live in the widget titlebar (RSS), outside the
    // [data-pager-wrap] that holds the pages — scope from the widget
    // root in that case.
    var wrap = btn.closest("[data-pager-wrap]") || btn.closest(".widget");
    if (!wrap) return;
    var pages = wrap.querySelectorAll("[data-page]");
    var current = wrap.querySelector("[data-page].active");
    if (!current) return;
    var curIdx = parseInt(current.getAttribute("data-page"));
    var nextIdx = dir === "next" ? curIdx + 1 : curIdx - 1;
    if (nextIdx < 0 || nextIdx >= pages.length) return;
    current.classList.remove("active");
    pages[nextIdx].classList.add("active");
    var ind = wrap.querySelector("[data-pager-indicator]");
    if (ind) ind.textContent = (nextIdx + 1) + " / " + pages.length;
    [].forEach.call(wrap.querySelectorAll("[data-pager]"), function(b) {
      var d = b.getAttribute("data-pager");
      b.disabled = (d === "prev" && nextIdx === 0) || (d === "next" && nextIdx >= pages.length - 1);
    });
  });
})();
