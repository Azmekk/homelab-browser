package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Azmekk/homelabbrowser/db"
	"github.com/go-chi/chi/v5"
)

const maxIconBytes = 2 * 1024 * 1024

var allowedIconExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".svg":  true,
	".webp": true,
	".ico":  true,
}

func (app *App) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	app.templates.render(w, "admin", map[string]any{
		"PageTitle": app.store.PageTitle(r.Context()),
		"Username":  user.Username,
	})
}

type serviceDTO struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Url        string `json:"url"`
	IconPath   string `json:"icon_path"`
	OpenNewTab bool   `json:"open_new_tab"`
	Position   int64  `json:"position"`
}

func dtoFromService(s db.Service) serviceDTO {
	return serviceDTO{
		ID:         s.ID,
		Title:      s.Title,
		Url:        s.Url,
		IconPath:   s.IconPath,
		OpenNewTab: s.OpenNewTab != 0,
		Position:   s.Position,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (app *App) handleListServices(w http.ResponseWriter, r *http.Request) {
	svcs, err := app.store.Queries.ListServices(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db error")
		return
	}
	out := make([]serviceDTO, len(svcs))
	for i, s := range svcs {
		out[i] = dtoFromService(s)
	}
	writeJSON(w, http.StatusOK, out)
}

type servicePayload struct {
	Title      string `json:"title"`
	Url        string `json:"url"`
	OpenNewTab bool   `json:"open_new_tab"`
}

func (p servicePayload) validate() error {
	if strings.TrimSpace(p.Title) == "" {
		return errors.New("title required")
	}
	if strings.TrimSpace(p.Url) == "" {
		return errors.New("url required")
	}
	return nil
}

func (app *App) handleCreateService(w http.ResponseWriter, r *http.Request) {
	var p servicePayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, "bad json")
		return
	}
	if err := p.validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	maxPos, err := app.store.Queries.MaxServicePosition(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db error")
		return
	}
	openFlag := int64(0)
	if p.OpenNewTab {
		openFlag = 1
	}
	svc, err := app.store.Queries.CreateService(r.Context(), db.CreateServiceParams{
		Title:      strings.TrimSpace(p.Title),
		Url:        strings.TrimSpace(p.Url),
		IconPath:   "",
		OpenNewTab: openFlag,
		Position:   maxPos + 1,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusCreated, dtoFromService(svc))
}

func (app *App) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	var p servicePayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, "bad json")
		return
	}
	if err := p.validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	openFlag := int64(0)
	if p.OpenNewTab {
		openFlag = 1
	}
	if err := app.store.Queries.UpdateService(r.Context(), db.UpdateServiceParams{
		Title:      strings.TrimSpace(p.Title),
		Url:        strings.TrimSpace(p.Url),
		OpenNewTab: openFlag,
		ID:         id,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "db error")
		return
	}
	svc, err := app.store.Queries.GetService(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, dtoFromService(svc))
}

func (app *App) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	svc, err := app.store.Queries.GetService(r.Context(), id)
	if err == nil && svc.IconPath != "" {
		app.removeIconFile(svc.IconPath)
	}
	if err := app.store.Queries.DeleteService(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, "db error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type reorderPayload struct {
	Order []int64 `json:"order"`
}

func (app *App) handleReorderServices(w http.ResponseWriter, r *http.Request) {
	var p reorderPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, "bad json")
		return
	}

	tx, err := app.store.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "tx error")
		return
	}
	defer tx.Rollback()
	qtx := app.store.Queries.WithTx(tx)
	for i, id := range p.Order {
		if err := qtx.SetServicePosition(r.Context(), db.SetServicePositionParams{
			Position: int64(i),
			ID:       id,
		}); err != nil {
			writeErr(w, http.StatusInternalServerError, "db error")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeErr(w, http.StatusInternalServerError, "commit error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (app *App) handleUploadIcon(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	svc, err := app.store.Queries.GetService(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeErr(w, http.StatusNotFound, "service not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "db error")
		return
	}

	if err := r.ParseMultipartForm(maxIconBytes + 1024); err != nil {
		writeErr(w, http.StatusBadRequest, "bad upload")
		return
	}
	file, hdr, err := r.FormFile("icon")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "missing icon file")
		return
	}
	defer file.Close()
	if hdr.Size > maxIconBytes {
		writeErr(w, http.StatusRequestEntityTooLarge, "icon too large (max 2MB)")
		return
	}
	ext := strings.ToLower(filepath.Ext(hdr.Filename))
	if !allowedIconExts[ext] {
		writeErr(w, http.StatusBadRequest, "unsupported icon format")
		return
	}

	rnd := make([]byte, 8)
	if _, err := rand.Read(rnd); err != nil {
		writeErr(w, http.StatusInternalServerError, "rng error")
		return
	}
	name := strconv.FormatInt(id, 10) + "-" + hex.EncodeToString(rnd) + ext
	destPath := filepath.Join(app.iconsDir, name)
	dst, err := os.Create(destPath)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "write error")
		return
	}
	if _, err := io.Copy(dst, io.LimitReader(file, maxIconBytes+1)); err != nil {
		dst.Close()
		os.Remove(destPath)
		writeErr(w, http.StatusInternalServerError, "write error")
		return
	}
	dst.Close()

	if svc.IconPath != "" {
		app.removeIconFile(svc.IconPath)
	}

	iconPath := "icons/" + name
	if err := app.store.Queries.SetServiceIconPath(r.Context(), db.SetServiceIconPathParams{
		IconPath: iconPath,
		ID:       id,
	}); err != nil {
		os.Remove(destPath)
		writeErr(w, http.StatusInternalServerError, "db error")
		return
	}

	updated, _ := app.store.Queries.GetService(r.Context(), id)
	writeJSON(w, http.StatusOK, dtoFromService(updated))
}

type settingsPayload struct {
	PageTitle string `json:"page_title"`
}

func (app *App) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var p settingsPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, "bad json")
		return
	}
	title := strings.TrimSpace(p.PageTitle)
	if title == "" {
		writeErr(w, http.StatusBadRequest, "page title required")
		return
	}
	if err := app.store.Queries.UpsertSetting(r.Context(), db.UpsertSettingParams{
		Key:   settingPageTitle,
		Value: title,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"page_title": title})
}

func (app *App) removeIconFile(relPath string) {
	if !strings.HasPrefix(relPath, "icons/") {
		return
	}
	base := strings.TrimPrefix(relPath, "icons/")
	if strings.ContainsAny(base, `/\`) || base == "" {
		return
	}
	_ = os.Remove(filepath.Join(app.iconsDir, base))
}
