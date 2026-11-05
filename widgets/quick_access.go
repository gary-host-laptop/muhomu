package widgets

import (
	"encoding/json"
	"html/template"

	"muhomu/internal/media"
)

type QuickAccessWidget struct{}

func (w *QuickAccessWidget) ID() string { return "quick-access" }

type quickAccessView struct {
	Index     int
	Label     string
	URL       string
	Favicon   string
	Letter    string
	UseLetter bool
}

func (w *QuickAccessWidget) Render(ctx RenderContext) (template.HTML, error) {
	items, err := ctx.Store.QuickAccess()
	if err != nil {
		return "", err
	}

	views := make([]quickAccessView, len(items))
	for i, item := range items {
		v := quickAccessView{
			Index:   i,
			Label:   item.Label,
			URL:     item.URL,
			Favicon: item.Favicon,
		}
		// See favTile: external favicons render as a letter tile so
		// the paint never contacts a third-party icon host.
		if media.IsExternalFavicon(v.Favicon) || v.Favicon == "" {
			v.UseLetter = true
			v.Letter = media.FavLetter(item.Label)
			v.Favicon = ""
		}
		views[i] = v
	}

	inner, err := renderWidgetTemplate("quick-access.tmpl", views)
	if err != nil {
		return "", err
	}
	state, _ := json.Marshal(items)
	return wrap(ctx, "quick-access", "クイックアクセス",
		`<button class="wt-btn" id="qa-add-btn"><i class="ph-light ph-plus"></i></button>`,
		string(inner), string(state)), nil
}
