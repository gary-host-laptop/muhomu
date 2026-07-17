package widgets

import "html/template"

type StatusWidget struct{}

func (w *StatusWidget) ID() string { return "status" }

func (w *StatusWidget) Render(ctx RenderContext) (template.HTML, error) {
	inner := `<div class="widget-body"><div class="status-rows">
  <div class="status-row"><span class="label">day</span><span class="value" id="day-name">—</span></div>
  <div class="status-row"><span class="label">week</span><span class="value" id="week-num">—</span></div>
  <div class="status-row"><span class="label">year</span><span class="value" id="year-progress">—</span></div>
  <div class="status-row"><span class="label">temp</span><span class="value" id="temperature">—</span></div>
</div></div>`
	return wrap(ctx, "status", "状態", "", inner), nil
}
