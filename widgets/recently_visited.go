package widgets

import (
	"encoding/json"
	"html/template"
)

type RecentlyVisitedWidget struct{}

func (w *RecentlyVisitedWidget) ID() string { return "recently-visited" }

func (w *RecentlyVisitedWidget) Render(ctx RenderContext) (template.HTML, error) {
	items, err := ctx.Store.Recent()
	if err != nil {
		return "", err
	}
	inner, err := renderWidgetTemplate("recently-visited.tmpl", items)
	if err != nil {
		return "", err
	}
	state, _ := json.Marshal(items)
	return wrap(ctx, "recently-visited", "後で見る", "", string(inner), string(state)), nil
}
