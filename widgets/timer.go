package widgets

import "html/template"

type TimerWidget struct{}

func (w *TimerWidget) ID() string { return "timer" }

func (w *TimerWidget) Render(ctx RenderContext) (template.HTML, error) {
	// The "vol" row is deliberately the same markup the rain widget uses for
	// its master volume: the rain-track / rain-slider classes are the shared
	// styling for audio controls, so both widgets line up identically. The
	// timer body has a fixed height (--audio-body-h, shared with rain), so
	// this row must not make the widget taller — the display absorbs it.
	inner, err := renderWidgetTemplate("timer.tmpl", nil)
	if err != nil {
		return "", err
	}
	acts := `<button class="wt-act" id="timer-trash" title="clear timer"><i class="ph-light ph-trash"></i></button>
    <button class="wt-act" id="timer-start"><i class="ph-light ph-play"></i></button>`
	return wrap(ctx, "timer", "タイマー", acts, string(inner)), nil
}
