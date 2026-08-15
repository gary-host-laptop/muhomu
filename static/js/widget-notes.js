/* Notes widget — htmx autosave. The textarea carries
   hx-post="/api/widget/notes" hx-trigger="input changed delay:800ms"
   hx-swap="none", so saves are debounced by htmx and the DOM is never
   swapped (focus/caret/IME survive typing). This file only handles the
   clear button and the beforeunload safety net for in-flight saves.    */
(function() {
  const notesEl = document.getElementById("notes");
  if (!notesEl) return;

  // dirty becomes false once a save request *successfully* completes; the
  // unload beacon only fires while there is unsaved input in flight. A failed
  // save leaves dirty true so the beacon still rescues the content on unload.
  let dirty = false;
  notesEl.addEventListener("input", () => { dirty = true; });
  document.addEventListener("htmx:afterRequest", (e) => {
    if (!e.detail.successful) return;
    const path = (e.detail.pathInfo && e.detail.pathInfo.requestPath) ||
                 (e.detail.requestConfig && e.detail.requestConfig.path);
    if (path === "/api/widget/notes") dirty = false;
  });

  const clearBtn = document.getElementById("notes-clear");
  if (clearBtn) {
    clearBtn.addEventListener("click", () => {
      if (!confirm("clear notes?")) return;
      notesEl.value = "";
      htmx.ajax("POST", "/api/widget/notes", {
        values: { state: "" },
        swap: "none",
      });
    });
  }

  window.addEventListener("beforeunload", () => {
    if (!dirty) return;
    // Form-encoded, same as the htmx autosave, so both write paths land on
    // the same endpoint (URLSearchParams sets the urlencoded content-type,
    // which handleWidgetPost's r.FormValue("state") parses).
    navigator.sendBeacon("/api/widget/notes",
      new URLSearchParams({ state: notesEl.value }));
  });
})();
