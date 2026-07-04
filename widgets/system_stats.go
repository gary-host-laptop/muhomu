package widgets

import "html/template"

type SystemStatsWidget struct{}

func (w *SystemStatsWidget) ID() string { return "system-stats" }

func (w *SystemStatsWidget) Render(ctx RenderContext) (template.HTML, error) {
	inner := `<div class="widget-body" style="padding:6px">
  <canvas id="stats-canvas" width="300" height="150"></canvas>
</div>`
	return wrap("system-stats", "blue", "システム", "", inner), nil
}
