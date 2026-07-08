package widgets

import (
	"html/template"
	"strings"
)

type SplitColumnWidget struct{}

func (w *SplitColumnWidget) ID() string { return "split-column" }

func (w *SplitColumnWidget) Render(ctx RenderContext) (template.HTML, error) {
	// ── Parse max-columns ──
	maxCols := 2
	if v, ok := ctx.Options["max-columns"]; ok {
		if n, ok := v.(float64); ok {
			maxCols = int(n)
		}
	}

	// ── Parse children ──
	rawChildren, ok := ctx.Options["widgets"].([]interface{})
	if !ok || len(rawChildren) == 0 {
		return "", nil
	}

	type childConfig struct {
		ID      string
		Options map[string]interface{}
	}
	var childConfigs []childConfig

	for _, raw := range rawChildren {
		m, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := m["id"].(string)
		if id == "" {
			id, _ = m["type"].(string)
		}
		if id == "" {
			continue
		}
		opts, _ := m["options"].(map[string]interface{})
		if opts == nil {
			opts = make(map[string]interface{})
		}
		// copy top-level keys (except reserved ones) as options
		for k, v := range m {
			if k != "id" && k != "type" && k != "options" {
				opts[k] = v
			}
		}
		childConfigs = append(childConfigs, childConfig{ID: id, Options: opts})
	}

	if len(childConfigs) == 0 {
		return "", nil
	}

	// ── Distribute into columns ──
	numCols := maxCols
	if len(childConfigs) < numCols {
		numCols = len(childConfigs)
	}
	if numCols < 1 {
		numCols = 1
	}
	cols := make([][]childConfig, numCols)
	for i, cfg := range childConfigs {
		idx := i % numCols
		cols[idx] = append(cols[idx], cfg)
	}

	// ── Render each column ──
	var sb strings.Builder
	sb.WriteString(`<div class="split-column">`)
	for _, col := range cols {
		sb.WriteString(`<div class="split-col">`)
		for _, child := range col {
			widget, ok := Registry()[child.ID]
			if !ok {
				continue
			}
			childCtx := ctx
			childCtx.Options = child.Options
			html, err := widget.Render(childCtx)
			if err != nil {
				continue
			}
			sb.WriteString(string(html))
		}
		sb.WriteString(`</div>`)
	}
	sb.WriteString(`</div>`)

	return template.HTML(sb.String()), nil
}
