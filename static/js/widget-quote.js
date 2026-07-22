/* ── widget-quote.js ───────────────────────────────────────────
   Render and init callbacks for the quote widget's preloader.
   Ensures clicking "next" always shows a different quote by
   passing the current quote text as ?current= to the API.       */
function render_quote(data, btn) {
  var widget = btn.closest(".widget");
  if (!widget) return;
  var textEl = widget.querySelector("[data-fetch-key='text']");
  var authEl = widget.querySelector("[data-fetch-key='author']");
  if (textEl) textEl.textContent = data.text;
  if (authEl) authEl.textContent = data.author;
  return "/api/quote/random?current=" + encodeURIComponent(data.text);
}

function quote_preload_init(btn) {
  var widget = btn.closest(".widget");
  if (!widget) return;
  var textEl = widget.querySelector("[data-fetch-key='text']");
  var current = textEl ? textEl.textContent : "";
  return "/api/quote/random" + (current ? "?current=" + encodeURIComponent(current) : "");
}
