package widgets

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

//go:embed templates/*.tmpl
var widgetTemplateFS embed.FS

var widgetTemplates = template.Must(
	template.New("widgets").ParseFS(widgetTemplateFS, "templates/*.tmpl"),
)

// renderWidgetTemplate executes one embedded widget template and returns
// its markup as template.HTML. Widget inner markup lives in templates/;
// wrap() still builds the shared widget chrome.
func renderWidgetTemplate(name string, data any) (template.HTML, error) {
	var buf bytes.Buffer
	if err := widgetTemplates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("render widget template %s: %w", name, err)
	}
	return template.HTML(buf.String()), nil
}
