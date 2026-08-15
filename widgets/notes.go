package widgets

import (
	"fmt"
	"html/template"
)

type NotesWidget struct{}

func (w *NotesWidget) ID() string { return "notes" }

func (w *NotesWidget) Render(ctx RenderContext) (template.HTML, error) {
	content, err := dbNotes(ctx.DB)
	if err != nil {
		content = ""
	}
	// The textarea autosaves via htmx: a debounced hx-post with
	// hx-swap="none" (no DOM swap, so focus/caret/IME survive typing).
	// name="state" feeds the existing /api/widget/{id} POST handler.
	inner := fmt.Sprintf(
		`<div class="widget-body"><textarea id="notes" name="state"
  hx-post="/api/widget/notes" hx-trigger="input changed delay:800ms" hx-swap="none"
  placeholder="// type here. auto-saves.">%s</textarea></div>`,
		htmlEscape(content),
	)
	return wrap(ctx, "notes", "メモ帳",
		`<button class="wt-act" id="notes-clear" title="clear notes"><i class="ph-light ph-trash"></i></button>`,
		inner), nil
}


