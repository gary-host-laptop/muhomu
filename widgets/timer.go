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
	inner := `<div class="timer-body">
  <div class="rain-track rain-master">
    <span class="rain-track-label">vol</span>
    <input type="range" class="rain-slider rain-slider-master" id="timer-vol" min="0" max="1" step="0.01" value="1">
  </div>
  <div class="rain-divider"></div>
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
  <button id="timer-reset" style="display:none;"></button>
</div>`
	acts := `<button class="wt-act" id="timer-trash" title="clear timer"><i class="ph-light ph-trash"></i></button>
    <button class="wt-act" id="timer-start"><i class="ph-light ph-play"></i></button>`
	return wrap(ctx, "timer", "タイマー", acts, inner), nil
}


