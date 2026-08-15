package widgets

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// step is the horizontal slot pitch used by the graph: graphW/(histN-1).
// The polyline x-coordinates in these tests are derived from it.
const testStep = graphW / float64(histN-1)

func TestStatsGraphFullRingRendersSlidingPoint(t *testing.T) {
	// A full ring (31 points = 30 visible + 1 extra) must render the
	// newest point one slot OFF-SCREEN right (x = 30*step ≈ 310.3). The
	// CSS glide then starts each new frame at translateX(0) — whose
	// visible points sit exactly where the previous frame's resting
	// points were — so the swap is pixel-identical and the line only
	// ever moves left. If the extra point is missing, the left edge
	// snaps right at every swap (the oscillation bug).
	hist := make([]StatsPoint, histN+1)
	for i := range hist {
		hist[i] = StatsPoint{CPU: 50, RAM: 40}
	}
	s := SystemStats{CPU: 50, RAM: 40, History: hist}
	html := string(statsGraph(s))

	// class must be applied when the ring is full
	if !strings.Contains(html, `<g class="stats-scroll">`) {
		t.Fatalf("full ring must carry the stats-scroll class:\n%s", html)
	}

	// polyline must contain a point at x = histN*step (off-screen right)
	lastX := float64(histN) * testStep
	wantLast := fmt.Sprintf("%.1f", lastX)
	re := regexp.MustCompile(`<polyline points="([^"]*)"`)
	matches := re.FindAllStringSubmatch(html, -1)
	if len(matches) != 2 {
		t.Fatalf("expected 2 polylines (cpu+mem), got %d", len(matches))
	}
	for _, m := range matches {
		fields := strings.Fields(m[1])
		if len(fields) != histN+1 {
			t.Fatalf("expected %d points, got %d", histN+1, len(fields))
		}
		last := fields[len(fields)-1]
		if !strings.HasPrefix(last, wantLast+",") {
			t.Fatalf("newest point should be off-screen right (x≈%s), got %q", wantLast, last)
		}
	}
}

func TestStatsGraphWarmupHasNoGlideClass(t *testing.T) {
	// During warmup the ring is still growing (fewer than histN+1
	// points), so gliding would shift the existing points — the class
	// must be absent and the line just extends rightward.
	hist := make([]StatsPoint, 5)
	for i := range hist {
		hist[i] = StatsPoint{CPU: 10, RAM: 20}
	}
	s := SystemStats{CPU: 10, RAM: 20, History: hist}
	html := string(statsGraph(s))

	if strings.Contains(html, "stats-scroll") {
		t.Fatalf("warmup render must not carry the stats-scroll class:\n%s", html)
	}
}

func TestStatsGraphStaticElementsOutsideGroup(t *testing.T) {
	hist := make([]StatsPoint, histN+1)
	for i := range hist {
		hist[i] = StatsPoint{CPU: 50, RAM: 40}
	}
	html := string(statsGraph(SystemStats{CPU: 50, RAM: 40, History: hist}))

	g := strings.Index(html, `<g class="stats-scroll">`)
	if g == -1 {
		t.Fatal("missing stats-scroll group")
	}
	gEnd := strings.Index(html[g:], "</g>") + g

	// gridlines are rendered before the group (static)
	if gi := strings.Index(html, `class="stats-grid"`); gi == -1 || gi > g {
		t.Fatal("gridlines must be outside (before) the animated group")
	}
	// mid divider is rendered after the group (static)
	if mi := strings.Index(html, `class="stats-mid"`); mi == -1 || mi < gEnd {
		t.Fatal("mid divider must be outside (after) the animated group")
	}
}
