package main

import (
	"strings"
	"testing"

	"muhomu/widgets"
)

func footerIndex(t *testing.T, html, needle string) int {
	t.Helper()
	i := strings.Index(html, needle)
	if i < 0 {
		t.Fatalf("footer HTML missing %q: %s", needle, html)
	}
	return i
}

func TestBuildFooterHTMLKeepsStatsInConfigOrder(t *testing.T) {
	html := buildFooterHTML([]string{"ram", "rx", "cpu", "tx"}, widgets.SystemStats{
		RX: 1.5, TX: 2.5, CPU: 42.5, RAM: 63,
	})

	positions := []int{
		footerIndex(t, html, `id="fi-ram"`),
		footerIndex(t, html, `id="fi-rx"`),
		footerIndex(t, html, `id="fi-cpu"`),
		footerIndex(t, html, `id="fi-tx"`),
		footerIndex(t, html, `class="fi-right"`),
	}
	for i := 1; i < len(positions); i++ {
		if positions[i-1] >= positions[i] {
			t.Fatalf("footer items are not in config order: %s", html)
		}
	}

	if n := strings.Count(html, `class="fi-version"`); n != 1 {
		t.Fatalf("expected exactly one version item, got %d: %s", n, html)
	}
	if n := strings.Count(html, `class="fi-item fi-github"`); n != 1 {
		t.Fatalf("expected exactly one github item, got %d: %s", n, html)
	}
	if !strings.Contains(html, "muhomu v"+version) {
		t.Fatalf("version missing from footer: %s", html)
	}
}

func TestBuildFooterHTMLAlwaysShowsRightGroup(t *testing.T) {
	html := buildFooterHTML(nil, widgets.SystemStats{})
	if !strings.HasPrefix(html, `<span class="fi-right">`) {
		t.Fatalf("expected right group to be the only content, got: %s", html)
	}
	if !strings.Contains(html, `class="fi-version"`) || !strings.Contains(html, `fi-github`) {
		t.Fatalf("right group missing version/github: %s", html)
	}
}

func TestBuildFooterHTMLIgnoresConfiguredVersionAndGithub(t *testing.T) {
	html := buildFooterHTML([]string{"cpu", "version", "github", "ram"}, widgets.SystemStats{})
	cpu := footerIndex(t, html, `id="fi-cpu"`)
	ram := footerIndex(t, html, `id="fi-ram"`)
	right := footerIndex(t, html, `class="fi-right"`)
	if cpu >= ram || ram >= right {
		t.Fatalf("stats should stay left of the right group: %s", html)
	}
	if n := strings.Count(html, `class="fi-version"`); n != 1 {
		t.Fatalf("expected exactly one version item, got %d: %s", n, html)
	}
	if n := strings.Count(html, `class="fi-item fi-github"`); n != 1 {
		t.Fatalf("expected exactly one github item, got %d: %s", n, html)
	}
}
