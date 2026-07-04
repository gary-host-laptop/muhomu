package widgets

import (
	"fmt"
	"html/template"
	"strings"
)

type RecentlyVisitedWidget struct{}

func (w *RecentlyVisitedWidget) ID() string { return "recently-visited" }

func (w *RecentlyVisitedWidget) Render(ctx RenderContext) (template.HTML, error) {
	items, err := dbRecent(ctx.DB)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString(`<div class="widget-body"><div class="recent-grid" id="recent-grid">`)
	for _, item := range items {
		fmt.Fprintf(&sb,
			`<div class="recent-tile">
  <div class="rt-name">%s</div>
  <div class="rt-url">%s</div>
  <button class="recent-x"><i class="ph-light ph-x"></i></button>
</div>`, htmlEscape(item.Name), htmlEscape(item.URL))
	}
	sb.WriteString(`</div>
<div class="recent-quick-add">
  <input type="text" id="rqa-name" placeholder="label" autocomplete="off" />
  <input type="text" id="rqa-url" placeholder="https://..." autocomplete="off" />
  <button id="rqa-btn"><span>+</span></button>
</div></div>`)
	return wrap("recently-visited", "pink", "後で見る", "", sb.String()), nil
}
