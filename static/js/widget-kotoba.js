/* ── widget-kotoba.js ─────────────────────────────────────────
   Render callback for the kotoba widget. Called by preloader.js
   on fetch complete. Updates kanji character, reading, meaning,
   and JLPT level in the widget DOM.                             */
function render_kotoba(w, btn) {
  var widget = btn.closest(".widget");
  if (!widget) return;
  var charEl = widget.querySelector("#kanji-char");
  if (charEl) {
    charEl.innerHTML = "<a href='https://jisho.org/search/" +
      encodeURIComponent(w.k) + "%20%23kanji' " +
      "style='color:inherit;text-decoration:none;'>" + w.k + "</a>";
  }
  var readEl = widget.querySelector("#kanji-reading");
  if (readEl) readEl.textContent = w.r;
  var meanEl = widget.querySelector("#kanji-meaning");
  if (meanEl) meanEl.textContent = w.m;
  var levelEl = widget.querySelector("#kanji-level");
  if (levelEl) {
    levelEl.textContent = w.l.toUpperCase();
    levelEl.className = "kanji-level " + w.l;
  }
}
