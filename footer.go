package main

import (
	"fmt"
	"net/http"
	"strings"

	"muhomu/widgets"
)

// version is baked in at build time via -ldflags "-X main.version=…";
// shown in the footer as "muhomu v…". The Dockerfile passes a VERSION
// build arg. Defaults to the current release if built without ldflags.
var version = "1.1.1"

// githubRepoURL is where the footer's GitHub icon links.
const githubRepoURL = "https://github.com/gary-host-laptop/muhomu"

func fmtFooterSpeed(kbps float64) string {
	if kbps >= 1024 {
		return fmt.Sprintf("%.1f MB/s", kbps/1024)
	}
	return fmt.Sprintf("%.1f KB/s", kbps)
}

func buildFooterHTML(footerItems []string, s widgets.SystemStats) string {
	var b strings.Builder
	written := 0
	for _, item := range footerItems {
		var stat string
		switch item {
		case "rx":
			stat = fmt.Sprintf(`<span class="fi-item fi-rx" id="fi-rx">⬇ %s</span>`, fmtFooterSpeed(s.RX))
		case "tx":
			stat = fmt.Sprintf(`<span class="fi-item fi-tx" id="fi-tx">⬆ %s</span>`, fmtFooterSpeed(s.TX))
		case "cpu":
			stat = fmt.Sprintf(`<span class="fi-item fi-cpu" id="fi-cpu">◉ cpu %.1f%%</span>`, s.CPU)
		case "ram":
			stat = fmt.Sprintf(`<span class="fi-item fi-ram" id="fi-ram">◉ ram %.0f%%</span>`, s.RAM)
		default:
			continue
		}
		if written > 0 {
			b.WriteString(`<span class="fi-sep">·</span>`)
		}
		b.WriteString(stat)
		written++
	}

	// The version and GitHub link are fixed furniture: they always sit at
	// the far right, regardless of footer_items order or membership.
	b.WriteString(`<span class="fi-right">`)
	b.WriteString(`<span class="fi-version">muhomu v` + version + `</span>`)
	b.WriteString(`<span class="fi-sep">·</span>`)
	b.WriteString(`<a class="fi-item fi-github" href="` + githubRepoURL +
		`" target="_blank" rel="noopener" title="muhomu on GitHub" aria-label="muhomu on GitHub"><i class="ph-light ph-github-logo"></i></a>`)
	b.WriteString(`</span>`)
	return b.String()
}

// handleFooterStats serves the footer's live stats fragment. The page
// footer polls this endpoint every 2s via hx-get.
func handleFooterStats(w http.ResponseWriter, r *http.Request) {
	cfg := getConfig()
	out := buildFooterHTML(cfg.FooterItems, currentSystemStats())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte(out))
}
