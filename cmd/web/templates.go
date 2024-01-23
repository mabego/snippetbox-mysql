package main

import (
	"html/template"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/mabego/snippetbox-mysql/internal/models"
	"github.com/mabego/snippetbox-mysql/ui"
)

// templateData holds dynamic data to pass to HTML templates.
type templateData struct {
	IsAuthenticated bool
	CurrentYear     int
	Flash           string
	CSRFToken       string
	Snippet         *models.Snippet
	Review          *models.Review
	User            *models.User
	Snippets        []*models.Snippet
	Form            any
}

func humanDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("Jan 02 2006 at 15:04")
}

var functions = template.FuncMap{"humanDate": humanDate}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	// Use fs.Glob to get a slice of all the 'page' files in the ui.Files embedded filesystem.
	pages, err := fs.Glob(ui.Files, "html/*.page.tmpl")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		// Create a slice containing the file path patterns to parse for the new templates.
		patterns := []string{
			"html/base.layout.tmpl",
			"html/*.partial.tmpl",
			page,
		}

		// Use ParseFS to parse the template files in the ui.Files embedded filesystem.
		ts, err := template.New(name).Funcs(functions).ParseFS(ui.Files, patterns...)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}
