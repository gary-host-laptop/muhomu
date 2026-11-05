package widgets

import (
	"fmt"
	"html/template"
	"math"
	"strings"
)

type SystemStatsWidget struct{}

func (w *SystemStatsWidget) ID() string { return "system-stats" }

type systemStatsView struct {
	Graph template.HTML
}

type statsGraphView struct {
	GridLines  []string
	HasHistory bool
	Scroll     bool
	CPUArea    string
	CPUPoints  string
	MEMArea    string
	MEMPoints  string
	CPU        string
	RAM        string
}

func (w *SystemStatsWidget) Render(ctx RenderContext) (template.HTML, error) {
	body, err := renderStatsBody(ctx.Stats)
	if err != nil {
		return "", err
	}
	return wrap(ctx, "system-stats", "システム", "", string(body)), nil
}

// The system-stats body drives htmx polling of this widget. Each poll re-fetches
// the WHOLE widget from the fragment endpoint and swaps the widget root
// (hx-target="closest .widget", hx-swap="outerHTML") — the full-widget swap
// every widget uses, so chrome and content always come from a single render
// (see the convention documented above wrap in widget.go).
//
// The trigger lives on the body div rather than the root: the body is
// re-rendered along with the widget, so each poll re-arms exactly one timer
// and the swap removes the root the old timer belonged to. Those attributes
// live in templates/system-stats.tmpl.
//
// The scroll glide in static/css/stats.css runs for exactly one poll period
// (2s), so each freshly-swapped line settles into place just as the next
// fragment arrives — if you change this period, change the animation
// duration to match. Re-creating the graph on every poll is what re-enters
// the glide at translateX(0), so the full-widget swap leaves it unchanged.
func renderStatsBody(s SystemStats) (template.HTML, error) {
	graph, err := statsGraph(s)
	if err != nil {
		return "", err
	}
	return renderWidgetTemplate("system-stats.tmpl", systemStatsView{Graph: graph})
}

// ── canvas-style graph ────────────────────────────────────────
// The original widget drew a 300x150 canvas: a CPU sparkline in the
// top half, a RAM sparkline in the bottom half, gridlines, a mid
// divider, and 9px pixel-font labels in the corners. This renders the
// same picture server-side — SVG for the lines (preserveAspectRatio
// none reproduces the old horizontal stretch behaviour) plus
// absolutely-positioned HTML labels, which stay crisp because they are
// not scaled with the SVG.

const (
	graphW  = 300.0
	graphH  = 150.0
	histN   = 30
	bandTop = 71.0 // height of each half-graph (h/2 - 4 in the old canvas)
	bandOff = 79.0 // y offset of the bottom half (h/2 + 4)
)

func clampPct(v float64) float64 {
	// Guard against NaN/Inf leaking from the stats collector (e.g. a
	// 0/0 delta between samples) into the SVG coordinates.
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// sparkY maps a 0–100 value into the old canvas coordinate space of a
// half-graph band: 0% at the band's bottom edge, 100% at the top.
func sparkY(v float64, yOff float64) float64 {
	return yOff + bandTop - (clampPct(v)/100)*bandTop
}

func sparkPoints(hist []StatsPoint, yOff float64, val func(StatsPoint) float64) string {
	step := graphW / float64(histN-1)
	var b strings.Builder
	for i, p := range hist {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%.1f,%.1f", float64(i)*step, sparkY(val(p), yOff))
	}
	return b.String()
}

// areaPoints closes the sparkline polygon along the band's baseline so
// the line can be filled with a translucent area below it. The right
// edge closes one slot past the visible width — the ring renders 31
// points (the 31st sits off-screen right at x = graphW+step, see
// sparkPoints), and the closing edge must reach that point so the fill
// covers it as it slides in.
func areaPoints(points string, yOff float64) string {
	bottom := yOff + bandTop
	step := graphW / float64(histN-1)
	return fmt.Sprintf("0,%.1f %s %.1f,%.1f", bottom, points, graphW+step, bottom)
}

func statsGraph(s SystemStats) (template.HTML, error) {
	return renderWidgetTemplate("stats-graph.tmpl", buildStatsGraphView(s))
}

func buildStatsGraphView(s SystemStats) statsGraphView {
	v := statsGraphView{
		CPU: fmt.Sprintf("%.0f%%", s.CPU),
		RAM: fmt.Sprintf("%.0f%%", s.RAM),
	}

	// gridlines — percentage reference lines on both bands, marking
	// 0% / 25% / 50% / 75% of each band (0% at the band's bottom edge).
	// 100% is deliberately omitted: that would mean the machine is
	// fried, and the line just clutters the top. The bottom band's 0%
	// line (y=graphH) sits on the viewBox edge, where half its 0.5px
	// stroke would be clipped thin — inset it by half a stroke so every
	// line renders at the same weight.
	for _, band := range []float64{0, bandOff} {
		for _, pct := range []float64{0, 25, 50, 75} {
			y := sparkY(pct, band)
			if y > graphH-0.5 {
				y = graphH - 0.5
			}
			v.GridLines = append(v.GridLines, fmt.Sprintf("%.1f", y))
		}
	}

	if len(s.History) >= 2 {
		cpu := sparkPoints(s.History, 0, func(p StatsPoint) float64 { return p.CPU })
		mem := sparkPoints(s.History, bandOff, func(p StatsPoint) float64 { return p.RAM })
		// The lines live in a .stats-scroll group so CSS can glide the
		// whole line left by one slot on each poll (see the keyframes in
		// static/css/stats.css) instead of the SVG teleporting to its new
		// position.
		//
		// The class is only applied once the ring is FULL (histN+1
		// points, i.e. statsMax in stats.go). The extra point renders
		// at x = histN*step = graphW+step, one slot OFF-SCREEN right; the
		// new frame at translateX(0) is then pixel-identical to the
		// previous frame's resting position (translateX(-step)), so the
		// swap is invisible and the glide slides the oldest point off
		// left while the newest enters from the right — the line only
		// ever moves left. During warmup the ring is still growing (no
		// points dropped), so a glide would shift the existing points;
		// without the class the line just extends rightward each poll.
		// Gridlines and the mid divider stay outside the group — they
		// are a fixed reference frame.
		v.HasHistory = true
		v.Scroll = len(s.History) > histN
		v.CPUArea = areaPoints(cpu, 0)
		v.CPUPoints = cpu
		v.MEMArea = areaPoints(mem, bandOff)
		v.MEMPoints = mem
	}

	return v
}
