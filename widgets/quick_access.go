package widgets

import (
	"fmt"
	"html/template"
	"strings"
)

type QuickAccessWidget struct{}

func (w *QuickAccessWidget) ID() string { return "quick-access" }

func (w *QuickAccessWidget) Render(ctx RenderContext) (template.HTML, error) {
	items, err := dbQuickAccess(ctx.DB)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString(`<div class="widget-body" style="padding:6px 10px"><div class="quick-links">`)
	for i, item := range items {
		fav := item.Favicon
		fmt.Fprintf(&sb,
			`<a href="%s" draggable="true" data-drag-index="%d">
  <img class="ql-fav" src="%s" alt="">
  %s
  <span class="qa-btn-group"><button class="qa-edit"><i class="ph-light ph-pencil-simple"></i></button></span>
</a>`, item.URL, i, fav, htmlEscape(item.Label))
	}
	sb.WriteString(`</div></div>`)
	return wrap(ctx, "quick-access", "クイックアクセス",
		`<button class="wt-btn" id="qa-add-btn"><i class="ph-light ph-plus"></i></button>`,
		sb.String()), nil
}


