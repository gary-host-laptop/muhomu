package widgets

import (
	"encoding/json"
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
		src := item.Favicon
		if IsExternalFavicon(src) || src == "" {
			// See favTile: external favicons render as a letter tile so
			// the paint never contacts a third-party icon host.
			fmt.Fprintf(&sb,
				`<a href="%s" draggable="true" data-drag-index="%d">
  <span class="ql-fav-letter">%s</span>
  %s
  <span class="qa-btn-group"><button class="qa-edit"><i class="ph-light ph-pencil-simple"></i></button></span>
</a>`, item.URL, i, htmlEscape(FavLetter(item.Label)), htmlEscape(item.Label))
			continue
		}
		fmt.Fprintf(&sb,
			`<a href="%s" draggable="true" data-drag-index="%d">
  <img class="ql-fav" src="%s" alt="">
  %s
  <span class="qa-btn-group"><button class="qa-edit"><i class="ph-light ph-pencil-simple"></i></button></span>
</a>`, item.URL, i, src, htmlEscape(item.Label))
	}
	sb.WriteString(`</div></div>`)
	state, _ := json.Marshal(items)
	return wrap(ctx, "quick-access", "クイックアクセス",
		`<button class="wt-btn" id="qa-add-btn"><i class="ph-light ph-plus"></i></button>`,
		sb.String(), string(state)), nil
}
