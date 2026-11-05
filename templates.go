package main

import (
	"fmt"
	"html/template"
)

// indexTmpl is the parsed new-tab page template, created once at startup.
// settingsTmpl is the parsed settings page template.
var (
	indexTmpl    *template.Template
	settingsTmpl *template.Template
)

// parseTemplates loads both page templates. Keeping parsing in one place
// makes startup failures explicit and keeps main() focused on wiring.
func parseTemplates() error {
	var err error
	indexTmpl, err = template.New("index.tmpl").Funcs(template.FuncMap{
		"widgetHTML": func(ws map[string]template.HTML, key string) template.HTML {
			return ws[key]
		},
	}).ParseFiles("templates/index.tmpl")
	if err != nil {
		return fmt.Errorf("parse index template: %w", err)
	}
	settingsTmpl, err = template.ParseFiles("templates/settings.tmpl")
	if err != nil {
		return fmt.Errorf("parse settings template: %w", err)
	}
	return nil
}
