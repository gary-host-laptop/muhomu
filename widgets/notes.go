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
	inner := fmt.Sprintf(
		`<div class="widget-body"><textarea id="notes" placeholder="// type here. auto-saves.">%s</textarea></div>`,
		htmlEscape(content),
	)
	return wrap("notes", "green", "メモ帳",
		`<button class="wt-act" id="notes-clear"><i class="ph-light ph-trash"></i></button>`,
		inner), nil
}
