package main

// widgetIDsWithScript is the set of widget IDs that have a JS file in
// static/js/. Widgets without JS (calendar, weather, and the ones rendered
// purely via htmx: image, kotoba, quote, system-stats) are excluded to
// avoid 404s.
var widgetIDsWithScript = map[string]bool{
	"bookmarks":        true,
	"notes":            true,
	"quick-access":     true,
	"rain":             true,
	"recently-visited": true,
	"timer":            true,
}

// collectActiveWidgetIDs recursively walks the widget tree and returns all
// unique widget IDs that have a corresponding JS file in static/js/.
func collectActiveWidgetIDs(configs []WidgetConfig) []string {
	seen := map[string]bool{}
	var ids []string
	var walk func([]WidgetConfig)
	walk = func(cfgs []WidgetConfig) {
		for _, w := range cfgs {
			if w.ID == "split-column" {
				walk(splitColumnChildren(w))
			} else if widgetIDsWithScript[w.ID] && !seen[w.ID] {
				seen[w.ID] = true
				ids = append(ids, w.ID)
			}
		}
	}
	walk(configs)
	return ids
}

// widgetListContains reports whether id appears in a widget config list,
// recursing into split-column children.
func widgetListContains(cfgs []WidgetConfig, id string) bool {
	for _, w := range cfgs {
		if w.ID == id {
			return true
		}
		if w.ID == "split-column" && widgetListContains(splitColumnChildren(w), id) {
			return true
		}
	}
	return false
}

// allWidgetConfigs flattens the configured header and columns into a
// widget list.
func allWidgetConfigs() []WidgetConfig {
	cfg := getConfig()
	var out []WidgetConfig
	out = append(out, cfg.Header...)
	for _, col := range cfg.Columns {
		out = append(out, col.Widgets...)
	}
	return out
}

// flattenWidgetConfigs expands split-column children into plain
// WidgetConfigs, recursing so nested splits are handled too.
func flattenWidgetConfigs(cfgs []WidgetConfig) []WidgetConfig {
	var out []WidgetConfig
	for _, w := range cfgs {
		if w.ID != "split-column" {
			out = append(out, w)
			continue
		}
		out = append(out, flattenWidgetConfigs(splitColumnChildren(w))...)
	}
	return out
}

// widgetConfigFor returns the first WidgetConfig matching id, recursing
// into split-column children as well.
func widgetConfigFor(id string) (WidgetConfig, bool) {
	for _, w := range allWidgetConfigs() {
		if w.ID == id {
			return w, true
		}
		if w.ID == "split-column" {
			if found, ok := widgetConfigForList(splitColumnChildren(w), id); ok {
				return found, true
			}
		}
	}
	return WidgetConfig{}, false
}

func widgetConfigForList(cfgs []WidgetConfig, id string) (WidgetConfig, bool) {
	for _, w := range cfgs {
		if w.ID == id {
			return w, true
		}
		if w.ID == "split-column" {
			if found, ok := widgetConfigForList(splitColumnChildren(w), id); ok {
				return found, true
			}
		}
	}
	return WidgetConfig{}, false
}

// splitColumnChildren parses a split-column widget's nested widget list.
// The config stores children as untyped YAML/JSON maps; this is the one
// place that normalises them into WidgetConfig values.
func splitColumnChildren(w WidgetConfig) []WidgetConfig {
	rawChildren, ok := w.Options["widgets"].([]interface{})
	if !ok {
		return nil
	}
	var childConfigs []WidgetConfig
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
		for k, v := range m {
			if k != "id" && k != "type" && k != "options" {
				opts[k] = v
			}
		}
		childConfigs = append(childConfigs, WidgetConfig{ID: id, Options: opts})
	}
	return childConfigs
}
