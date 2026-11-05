package widgets

import "html/template"

type RainWidget struct{}

func (w *RainWidget) ID() string { return "rain" }

func (w *RainWidget) Render(ctx RenderContext) (template.HTML, error) {
	inner, err := renderWidgetTemplate("rain.tmpl", nil)
	if err != nil {
		return "", err
	}
	return wrap(ctx, "rain", "雨音",
		`<button class="wt-act" id="rain-btn"><i class="ph-light ph-play"></i></button>`,
		string(inner)), nil
}
