package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

const (
	defaultPageTitle = "My Homelab"
	settingPageTitle = "page_title"
)

type envConfig struct {
	Port            string
	DataDir         string
	ReloadTemplates bool
}

type App struct {
	store     *Store
	templates *templateSet
	iconsDir  string
}

func loadEnv() envConfig {
	// godotenv does not overwrite env vars that are already set, so it's
	// safe to call unconditionally — in a container the Docker ENVs win
	// and the (missing) .env is silently ignored.
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	reload, _ := strconv.ParseBool(os.Getenv("RELOAD_TEMPLATES"))
	return envConfig{
		Port:            port,
		DataDir:         dataDir,
		ReloadTemplates: reload,
	}
}

func ensureDirs(dataDir string) (string, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", err
	}
	icons := filepath.Join(dataDir, "icons")
	if err := os.MkdirAll(icons, 0o755); err != nil {
		return "", err
	}
	return icons, nil
}

func main() {
	cfg := loadEnv()

	iconsDir, err := ensureDirs(cfg.DataDir)
	if err != nil {
		log.Fatalf("create data dirs: %v", err)
	}

	store, err := openStore(cfg.DataDir)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer store.Close()

	go func() {
		ticker := time.NewTicker(12 * time.Hour)
		defer ticker.Stop()
		for {
			_ = store.Queries.DeleteExpiredSessions(context.Background(), time.Now().Unix())
			<-ticker.C
		}
	}()

	tmpl, err := newTemplateSet(cfg.ReloadTemplates)
	if err != nil {
		log.Fatalf("load templates: %v", err)
	}

	app := &App{
		store:     store,
		templates: tmpl,
		iconsDir:  iconsDir,
	}

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(middleware.Timeout(30 * time.Second))

	// Public: only what's needed to sign in or sign up. Static CSS/JS is
	// also public because the login/setup pages reference them.
	r.Get("/login", app.handleLoginPage)
	r.Post("/login", app.handleLoginSubmit)
	r.Post("/logout", app.handleLogout)
	r.Get("/setup", app.handleSetupPage)
	r.Post("/setup", app.handleSetupSubmit)
	r.Get("/styles.css", staticFileHandler("styles.css", "text/css; charset=utf-8"))
	r.Get("/scripts.js", staticFileHandler("scripts.js", "application/javascript; charset=utf-8"))
	r.Get("/admin.js", staticFileHandler("admin.js", "application/javascript; charset=utf-8"))
	r.Get("/app-icon.png", staticFileHandler("app-icon.png", "image/png"))

	// Everything else is behind auth. Unauthed browser requests redirect
	// to /login; unauthed JSON requests get 401.
	iconFS := http.StripPrefix("/icons/", http.FileServer(http.Dir(iconsDir)))
	r.Group(func(r chi.Router) {
		r.Use(app.requireAuth)
		r.Get("/", app.handleDashboard)
		r.Get("/icons/*", iconFS.ServeHTTP)
		r.Get("/admin", app.handleAdminPage)
		r.Get("/admin/api/services", app.handleListServices)
		r.Post("/admin/api/services", app.handleCreateService)
		r.Put("/admin/api/services/{id}", app.handleUpdateService)
		r.Delete("/admin/api/services/{id}", app.handleDeleteService)
		r.Post("/admin/api/services/reorder", app.handleReorderServices)
		r.Post("/admin/api/services/{id}/icon", app.handleUploadIcon)
		r.Post("/admin/api/settings", app.handleUpdateSettings)
	})

	addr := ":" + cfg.Port
	log.Printf("homelab-browser listening on %s (data dir: %s)", addr, cfg.DataDir)
	log.Fatal(http.ListenAndServe(addr, r))
}
