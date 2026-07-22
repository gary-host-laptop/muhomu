/* ── widget-image.js ───────────────────────────────────────────
   Cycles through images sequentially, preloading the next one
   for instant display on click.                                 */
(function() {
  var btn = document.getElementById("widget-img-next");
  var wrap = document.getElementById("widget-img-wrap");
  if (!btn || !wrap) return;
  var currentFilename = wrap.querySelector("img")?.dataset.filename || "";
  var preloaded = null;

  function setImage(url, filename) {
    var img = document.createElement("img");
    img.src = url; img.id = "widget-img";
    img.className = "active"; img.dataset.filename = filename;
    wrap.innerHTML = ""; wrap.appendChild(img);
    currentFilename = filename;
  }

  function preloadNext(after) {
    fetch("/api/widget-images/next?sequential=true&current=" + encodeURIComponent(after))
      .then(function(r) { return r.json(); })
      .then(function(d) {
        if (!d.url) return;
        var i = new Image();
        i.src = d.url;
        preloaded = { url: d.url, filename: d.filename };
      })
      .catch(function() {});
  }

  btn.addEventListener("click", function() {
    if (!preloaded) {
      fetch("/api/widget-images/next?sequential=true&current=" + encodeURIComponent(currentFilename))
        .then(function(r) { return r.json(); })
        .then(function(d) {
          if (!d.url) return;
          setImage(d.url, d.filename);
          preloadNext(d.filename);
        })
        .catch(function() {});
      return;
    }
    setImage(preloaded.url, preloaded.filename);
    preloaded = null;
    preloadNext(currentFilename);
  });

  if (currentFilename) preloadNext(currentFilename);
})();
