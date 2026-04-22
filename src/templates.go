package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"sync"
)

//go:embed wwwroot/templates/*.html
var templatesFS embed.FS

type templateSet struct {
	mu     sync.RWMutex
	pages  map[string]*template.Template
	reload bool
}

var pageNames = []string{"dashboard", "admin", "login", "setup"}

func newTemplateSet(reload bool) (*templateSet, error) {
	ts := &templateSet{reload: reload}
	if err := ts.load(); err != nil {
		return nil, err
	}
	return ts, nil
}

func (t *templateSet) load() error {
	pages := make(map[string]*template.Template, len(pageNames))
	for _, name := range pageNames {
		tpl := template.New("layout.html").Funcs(template.FuncMap{
			"boolInt": func(i int64) bool { return i != 0 },
		})
		parsed, err := tpl.ParseFS(templatesFS,
			"wwwroot/templates/layout.html",
			"wwwroot/templates/"+name+".html",
		)
		if err != nil {
			return fmt.Errorf("parse %s: %w", name, err)
		}
		pages[name] = parsed
	}
	t.mu.Lock()
	t.pages = pages
	t.mu.Unlock()
	return nil
}

func (t *templateSet) render(w http.ResponseWriter, name string, data any) {
	if t.reload {
		if err := t.load(); err != nil {
			http.Error(w, "template reload error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	t.mu.RLock()
	tpl, ok := t.pages[name]
	t.mu.RUnlock()
	if !ok {
		http.Error(w, "template not found: "+name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, "template render error: "+err.Error(), http.StatusInternalServerError)
	}
}

//go:embed wwwroot/styles.css wwwroot/scripts.js wwwroot/admin.js
var staticFS embed.FS

func staticFileHandler(name string, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(staticFS, path.Join("wwwroot", name))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(data)
	}
}
