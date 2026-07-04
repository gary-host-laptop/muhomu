package widgets

import "html/template"

type TimerWidget struct{}

func (w *TimerWidget) ID() string { return "timer" }

func (w *TimerWidget) Render(ctx RenderContext) (template.HTML, error) {
	inner := `
<div class="timer-display" id="timer-display">00:00</div>
<div class="timer-inputs">
  <div>
    <input type="number" class="timer-input" id="timer-min" value="0" min="0" max="99" placeholder="00">
    <div class="timer-label">min</div>
  </div>
  <span class="timer-sep">:</span>
  <div>
    <input type="number" class="timer-input" id="timer-sec" value="0" min="0" max="59" placeholder="00">
    <div class="timer-label">sec</div>
  </div>
</div>
<button id="timer-reset" style="display:none;"></button>`
	acts := `<button class="wt-act" id="timer-trash" title="clear timer"><i class="ph-light ph-trash"></i></button>
    <button class="wt-act" id="timer-start"><i class="ph-light ph-play"></i></button>`
	return wrap("timer", "green", "タイマー", acts, inner), nil
}
