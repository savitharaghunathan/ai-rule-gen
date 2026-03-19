package templates

import (
	"embed"
	"fmt"
	"text/template"
)

//go:embed extraction/*.tmpl generation/*.tmpl testing/*.tmpl confidence/*.tmpl
var FS embed.FS

// Load parses a named template from the embedded filesystem.
func Load(name string) (*template.Template, error) {
	tmpl, err := template.ParseFS(FS, name)
	if err != nil {
		return nil, fmt.Errorf("loading template %s: %w", name, err)
	}
	return tmpl, nil
}
