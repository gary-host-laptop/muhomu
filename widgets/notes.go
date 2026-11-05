package widgets

import "html/template"

type NotesWidget struct{}

func (w *NotesWidget) ID() string { return "notes" }

func (w *NotesWidget) Render(ctx RenderContext) (template.HTML, error) {
	content, err := ctx.Store.Notes()
	if err != nil {
		content = ""
	}
	// The textarea autosaves via htmx: a debounced hx-post with
	// hx-swap="none" (no DOM swap, so focus/caret/IME survive typing).
	// name="state" feeds the existing /api/widget/{id} POST handler.
	inner, err := renderWidgetTemplate("notes.tmpl", content)
	if err != nil {
		return "", err
	}
	return wrap(ctx, "notes", "メモ帳",
		`<button class="wt-act" id="notes-clear" title="clear notes"><i class="ph-light ph-trash"></i></button>`,
		string(inner)), nil
}
