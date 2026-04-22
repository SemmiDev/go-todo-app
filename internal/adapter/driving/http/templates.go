package http

import (
	"fmt"
	"html/template"

	"github.com/semmidev/todo-app/web"
)

// templates holds the pre-parsed HTML templates used by the page handlers.
type templates struct {
	index    *template.Template
	dashboard *template.Template
	callback  *template.Template
}

// parseTemplates parses all page templates from the embedded web filesystem.
// Each page template is cloned from a shared layout base.
func parseTemplates() (*templates, error) {
	tmplBase, err := template.ParseFS(web.TemplateFS, "layout.html")
	if err != nil {
		return nil, fmt.Errorf("parse base template: %w", err)
	}

	tmplIndex, err := template.Must(tmplBase.Clone()).ParseFS(web.TemplateFS, "index.html")
	if err != nil {
		return nil, fmt.Errorf("parse index template: %w", err)
	}

	tmplDashboard, err := template.Must(tmplBase.Clone()).ParseFS(web.TemplateFS, "dashboard.html")
	if err != nil {
		return nil, fmt.Errorf("parse dashboard template: %w", err)
	}

	tmplCallback, err := template.Must(tmplBase.Clone()).ParseFS(web.TemplateFS, "callback.html")
	if err != nil {
		return nil, fmt.Errorf("parse callback template: %w", err)
	}

	return &templates{
		index:     tmplIndex,
		dashboard: tmplDashboard,
		callback:  tmplCallback,
	}, nil
}
