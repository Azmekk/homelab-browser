package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/Azmekk/homelabbrowser/db"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "session"
	sessionDuration   = 90 * 24 * time.Hour
)

type ctxKey int

const userCtxKey ctxKey = 1

type sessionUser struct {
	ID       int64
	Username string
}

func newSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func checkPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func (app *App) createSession(ctx context.Context, userID int64) (string, time.Time, error) {
	token, err := newSessionToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().Add(sessionDuration)
	err = app.store.Queries.CreateSession(ctx, db.CreateSessionParams{
		Token:     token,
		UserID:    userID,
		ExpiresAt: expires.Unix(),
	})
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

func (app *App) setSessionCookie(w http.ResponseWriter, r *http.Request, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(sessionDuration.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (app *App) currentUser(r *http.Request) (*sessionUser, string, bool) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return nil, "", false
	}
	row, err := app.store.Queries.GetSession(r.Context(), c.Value)
	if err != nil {
		return nil, "", false
	}
	if time.Unix(row.ExpiresAt, 0).Before(time.Now()) {
		_ = app.store.Queries.DeleteSession(r.Context(), c.Value)
		return nil, "", false
	}
	return &sessionUser{ID: row.UserID, Username: row.Username}, c.Value, true
}

func (app *App) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, token, ok := app.currentUser(r)
		if !ok {
			if isJSONRequest(r) {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		newExpiry := time.Now().Add(sessionDuration)
		_ = app.store.Queries.RefreshSession(r.Context(), db.RefreshSessionParams{
			ExpiresAt: newExpiry.Unix(),
			Token:     token,
		})
		app.setSessionCookie(w, r, token, newExpiry)

		ctx := context.WithValue(r.Context(), userCtxKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userFromContext(ctx context.Context) (*sessionUser, bool) {
	u, ok := ctx.Value(userCtxKey).(*sessionUser)
	return u, ok
}

func (app *App) hasAnyUser(ctx context.Context) (bool, error) {
	n, err := app.store.Queries.CountUsers(ctx)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (app *App) authenticate(ctx context.Context, username, password string) (*db.User, error) {
	u, err := app.store.Queries.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}
	if !checkPassword(u.PasswordHash, password) {
		return nil, errors.New("invalid credentials")
	}
	return &u, nil
}
