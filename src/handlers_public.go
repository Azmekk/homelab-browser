package main

import (
	"net/http"
	"strings"

	"github.com/Azmekk/homelabbrowser/db"
)

type serviceView struct {
	ID         int64
	Title      string
	Url        string
	IconPath   string
	OpenNewTab bool
}

func toServiceViews(in []db.Service) []serviceView {
	out := make([]serviceView, len(in))
	for i, s := range in {
		out[i] = serviceView{
			ID:         s.ID,
			Title:      s.Title,
			Url:        s.Url,
			IconPath:   s.IconPath,
			OpenNewTab: s.OpenNewTab != 0,
		}
	}
	return out
}

func (app *App) handleDashboard(w http.ResponseWriter, r *http.Request) {
	services, err := app.store.Queries.ListServices(r.Context())
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	app.templates.render(w, "dashboard", map[string]any{
		"PageTitle": app.store.PageTitle(r.Context()),
		"Services":  toServiceViews(services),
	})
}

func (app *App) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	hasUser, err := app.hasAnyUser(r.Context())
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !hasUser {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}
	app.templates.render(w, "login", map[string]any{
		"PageTitle": app.store.PageTitle(r.Context()),
		"Error":     "",
	})
}

func (app *App) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	u, err := app.authenticate(r.Context(), username, password)
	if err != nil {
		app.templates.render(w, "login", map[string]any{
			"PageTitle": app.store.PageTitle(r.Context()),
			"Error":     "Invalid username or password.",
			"Username":  username,
		})
		return
	}

	token, expires, err := app.createSession(r.Context(), u.ID)
	if err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	app.setSessionCookie(w, r, token, expires)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (app *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
		_ = app.store.Queries.DeleteSession(r.Context(), c.Value)
	}
	clearSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) handleSetupPage(w http.ResponseWriter, r *http.Request) {
	hasUser, err := app.hasAnyUser(r.Context())
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if hasUser {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	app.templates.render(w, "setup", map[string]any{
		"PageTitle": app.store.PageTitle(r.Context()),
		"Error":     "",
	})
}

func (app *App) handleSetupSubmit(w http.ResponseWriter, r *http.Request) {
	hasUser, err := app.hasAnyUser(r.Context())
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if hasUser {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	confirm := r.FormValue("confirm")

	render := func(msg string) {
		app.templates.render(w, "setup", map[string]any{
			"PageTitle": app.store.PageTitle(r.Context()),
			"Error":     msg,
			"Username":  username,
		})
	}

	if len(username) < 2 {
		render("Username must be at least 2 characters.")
		return
	}
	if len(password) < 8 {
		render("Password must be at least 8 characters.")
		return
	}
	if password != confirm {
		render("Passwords do not match.")
		return
	}

	hash, err := hashPassword(password)
	if err != nil {
		render("Could not hash password.")
		return
	}
	u, err := app.store.Queries.CreateUser(r.Context(), db.CreateUserParams{
		Username:     username,
		PasswordHash: hash,
	})
	if err != nil {
		render("Could not create user.")
		return
	}

	token, expires, err := app.createSession(r.Context(), u.ID)
	if err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	app.setSessionCookie(w, r, token, expires)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func isJSONRequest(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json") ||
		strings.HasPrefix(r.URL.Path, "/admin/api/")
}
