package templates

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

//go:embed *.html
var files embed.FS

// Render executes the named HTML template file with the given data and returns
// the rendered HTML string.
//
// templateName must match the filename inside notification/templates/, e.g.
// "leave_applied.html", "employee_created.html".
//
// data can be any struct whose exported fields match the template's {{.Field}}
// references. Wrap with a TemplateData helper (see below) to include shared
// fields like AppName and AppURL automatically.
func Render(templateName string, data any) (string, error) {
	tmpl, err := template.New(templateName).ParseFS(files, templateName)
	if err != nil {
		return "", fmt.Errorf("templates: parse %q: %w", templateName, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("templates: execute %q: %w", templateName, err)
	}

	return buf.String(), nil
}
