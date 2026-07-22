/* ── pager.js ──────────────────────────────────────────────────
   Prev/next pagination for pre-rendered pages. [data-pager] on
   buttons, [data-page] on page divs, [data-pager-indicator] on
   counter, all within a [data-pager-wrap] container.            */
(function() {
  document.addEventListener("click", function(e) {
    var btn = e.target.closest("[data-pager]");
    if (!btn) return;
    var dir = btn.getAttribute("data-pager");
    var wrap = btn.closest("[data-pager-wrap]");
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
