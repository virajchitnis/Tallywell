package server

import (
	"html/template"
	"net/http"

	"github.com/tallywell/tallywell/internal/model"
)

// templateFuncs are helpers available in templates.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"money": func(c model.Cents) string { return c.Display() },
		"date": func(d model.Date) string {
			if d.IsZero() {
				return ""
			}
			return d.String()
		},
	}
}

// render executes the named template with data, writing a 500 on error.
func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// pageData is the common envelope passed to page templates.
type pageData struct {
	Active  string // nav highlight key
	Flash   string
	Content any
}
