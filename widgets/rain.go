package widgets

import "html/template"

type RainWidget struct{}

func (w *RainWidget) ID() string { return "rain" }

func (w *RainWidget) Render(ctx RenderContext) (template.HTML, error) {
	inner := `<div class="rain-player">
  <div class="rain-track rain-master">
    <span class="rain-track-label">vol</span>
    <input type="range" class="rain-slider rain-slider-master" id="vol-master" min="0" max="1" step="0.01" value="1">
  </div>
  <div class="rain-divider"></div>
  <div class="rain-track">
    <span class="rain-track-label">雨</span>
    <input type="range" class="rain-slider" id="vol-rain" min="0" max="1" step="0.01" value="0.7">
  </div>
  <div class="rain-track">
    <span class="rain-track-label">風</span>
    <input type="range" class="rain-slider" id="vol-wind" min="0" max="1" step="0.01" value="0.4">
  </div>
  <div class="rain-track">
    <span class="rain-track-label">雷</span>
    <input type="range" class="rain-slider" id="vol-thunder" min="0" max="1" step="0.01" value="0.5">
  </div>
</div>`
	return wrap("rain", "blue", "雨音",
		`<button class="wt-act" id="rain-btn"><i class="ph-light ph-play"></i></button>`,
		inner), nil
}
