package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// themeVars are the base palette values inlined into the page as :root
// custom properties. theme.css owns the component rules; this map only
// carries the values that must exist before the stylesheet loads.
var themeVars = map[string]map[string]string{
	"dark": {
		"--bg":        "#060608",
		"--panel":     "#12111a",
		"--panel2":    "#16151f",
		"--border":    "#302d42",
		"--border-lt": "#4a4660",
		"--white":     "#e8e8f0",
		"--dim":       "#8a88a0",
		"--dimmer":    "#4a4660",
		"--accent":    "#8ab4f8",
		"--accent2":   "#c4a0f8",
		"--accent3":   "#90d870",
		"--red":       "#f28080",
		"--titlebar":  "#0e0d16",
	},
	"light": {
		"--bg":        "#e8e8e8",
		"--panel":     "#f2f2f2",
		"--panel2":    "#eaeaea",
		"--border":    "#c8c8cc",
		"--border-lt": "#b0b0b8",
		"--white":     "#1a1a24",
		"--dim":       "#5a5a6a",
		"--dimmer":    "#a0a0b0",
		"--accent":    "#3a6fd8",
		"--accent2":   "#7040c8",
		"--accent3":   "#3a9a3a",
		"--red":       "#c83030",
		"--titlebar":  "#dcdce0",
	},
}

func buildThemeStyle(cfg *AppConfig) template.CSS {
	vars := themeVars[cfg.Theme]
	if vars == nil {
		vars = themeVars["dark"]
	}
	parts := make([]string, 0, len(vars))
	for k, v := range vars {
		parts = append(parts, k+":"+v)
	}
	return template.CSS(":root{" + strings.Join(parts, ";") + "}")
}

func buildFontStyle(cfg *AppConfig) template.CSS {
	fontLatinMap := map[string]string{
		"inter": "'Inter', sans-serif",
	}
	fontJpMap := map[string]string{
		"dotgothic16": "'DotGothic16', monospace",
	}
	fontClockMap := map[string]string{
		"orbitron": "'Orbitron', monospace",
	}

	fontPixel := ""
	if fl, ok := fontLatinMap[cfg.FontLatin]; ok {
		jp := fontJpMap[cfg.FontJP]
		if jp == "" {
			jp = "'DotGothic16', monospace"
		}
		fontPixel = fl + ", " + jp
	} else if fj, ok := fontJpMap[cfg.FontJP]; ok {
		fontPixel = "'DotGothic16', " + fj
	} else if cfg.FontLatin == "" {
		fontPixel = "'Inter', 'DotGothic16', monospace"
	}

	var fontParts []string
	if fontPixel != "" {
		fontParts = append(fontParts, "--font-pixel:"+fontPixel)
	}
	if fontDoto := fontClockMap[cfg.FontClock]; fontDoto != "" {
		fontParts = append(fontParts, "--font-doto:"+fontDoto)
	}
	if len(fontParts) == 0 {
		return template.CSS("")
	}
	return template.CSS("body.theme-" + cfg.Theme + "{" + strings.Join(fontParts, ";") + "}")
}

func fontClassFor(cfg *AppConfig) string {
	return "font-" + cfg.FontClock
}

func clockClassFor(cfg *AppConfig) string {
	switch cfg.ClockTheme {
	case "light":
		return "clock-force-light"
	case "dark":
		return "clock-force-dark"
	default:
		return ""
	}
}

// buildThemeCSSLinks returns a <link> for the optional custom theme file,
// or an empty string when no file is configured. A missing file is logged
// and skipped rather than failing the page render.
func buildThemeCSSLinks(cfg *AppConfig) template.HTML {
	if cfg.ThemeFile == "" {
		return ""
	}
	if _, err := os.Stat(filepath.Join(appCfg.themesDir, cfg.ThemeFile)); err != nil {
		log.Printf("serveIndex: theme_file %q not found in %s", cfg.ThemeFile, appCfg.themesDir)
		return ""
	}
	return template.HTML(fmt.Sprintf(`<link rel="stylesheet" href="/api/themes/%s">`+"\n", cfg.ThemeFile))
}
