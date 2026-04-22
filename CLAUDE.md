# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A self-hosted homelab service dashboard. A Go server (chi v5 + stdlib `html/template`) renders a public dashboard of service tiles plus a gated admin panel (CRUD + icon upload + drag-or-button reorder + open-in-new-tab toggle). All state — page title, services, users, sessions — lives in a single SQLite DB; uploaded icons sit next to it. Frontend is a modern dark responsive design with a top bar that shows a live clock and Open-Meteo weather.

## Commands

All Go commands run from `src/`.

- Run locally: `go run .` (uses `./data/` by default; create a `.env` with `BIND_URL=:8080` or export it; set `RELOAD_TEMPLATES=true` during dev to re-parse templates per request).
- Build binary: `go build -o homelab-browser .`
- Regenerate DB code after editing `db/schema.sql` or `db/queries.sql`: `sqlc generate` (run from `src/`; requires `sqlc` — `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`).
- Compile/watch SCSS during dev: `./update-and-watch-css.sh` (requires `sass` — `npm i -g sass`, or invoke `npx sass ...` directly).
- Docker build: `docker build -t homelab-browser .` from the repo root.
- Docker compose: `cp docker-compose.yml.example docker-compose.yml && docker compose up -d`.

No test suite exists.

## Architecture notes

- **Router.** chi v5 with `middleware.{RealIP,Logger,Recoverer,Compress,Timeout}`. Public routes live at the root; admin routes live under `chi.Router.Group` with an `app.requireAuth` middleware that redirects unauth'd browser requests to `/login` and returns 401 JSON for `/admin/api/*`.
- **Data lives in `DATA_DIR`.** Defaults to `./data` locally and `/data` in the container (Dockerfile pins both via env). `DATA_DIR/app.db` is the SQLite database; `DATA_DIR/icons/` holds uploaded icons served at `/icons/<file>`. The directory is created on boot.
- **SQLite + sqlc.** The DB driver is `modernc.org/sqlite` (pure Go, so the Dockerfile builds with `CGO_ENABLED=0` and needs no SQLite apt packages). `src/db/schema.sql` is the source of truth for tables; it is both read by sqlc at `sqlc generate` time AND executed at boot via `//go:embed` to apply idempotent `CREATE TABLE IF NOT EXISTS`. Query code in `src/db/*.go` is generated from `src/db/queries.sql` and checked into git, so production builds do not need the sqlc binary.
- **First-run setup.** If the `users` table is empty, `GET /login` redirects to `GET /setup`, where the operator creates the initial admin account. Once any user exists, `/setup` redirects back to `/login`. There are no default credentials — the app refuses to be usable until a human creates one.
- **Sessions.** 32-byte random hex tokens stored in the `sessions` table; expiry is **90 days sliding** — `requireAuth` calls `RefreshSession` and rewrites the cookie on every authed request. The cookie is `HttpOnly`, `SameSite=Lax`, and `Secure` when the request is TLS. A goroutine in `main` calls `DeleteExpiredSessions` every 12 hours.
- **Passwords.** bcrypt via `golang.org/x/crypto/bcrypt`. Setup enforces username ≥ 2 chars, password ≥ 8 chars with a confirmation field.
- **Templates.** Stdlib `html/template` with a `layout.html` + per-page body (dashboard, admin, login, setup). The template set is parsed once at boot via `//go:embed` of `wwwroot/templates/*.html`; `RELOAD_TEMPLATES=true` makes `templateSet.render` reparse on each request for fast iteration.
- **Static assets.** `styles.css`, `scripts.js`, and `admin.js` are embedded at compile time via `//go:embed` and served by simple handlers. The Dockerfile runs dart-sass in the builder stage so the embedded `styles.css` is the compiled artifact. Uploaded icons are NOT embedded — they come from `DATA_DIR/icons` at runtime.
- **Admin UI.** `admin.html` is an Alpine.js 3 component (`adminApp()` in `admin.js`) loaded via CDN. Reorder uses SortableJS (also CDN) for drag AND visible ▲/▼ buttons for touch/mobile — both paths POST the new id order to `/admin/api/services/reorder`, which writes positions in a single transaction. Icon upload is a separate multipart POST after the create/update JSON call succeeds. Each service has a standalone `open_new_tab` checkbox in the list row that updates immediately; the edit modal also has one.
- **Dashboard UI.** `scripts.js` runs `navigator.geolocation` with a 4s timeout; on success it calls Open-Meteo's forecast + reverse-geocoding APIs, on failure it falls back to `ipapi.co` for coordinates. No server-side weather proxy and no API keys. The clock updates roughly every 15 seconds, with a one-shot realignment at the next minute boundary so it doesn't drift visibly.
- **Mobile.** `styles.scss` is mobile-first with a breakpoint at 720px: dashboard grid collapses to one column, admin rows restack into a grid-template-areas layout, the edit modal becomes fullscreen, and the username chip is hidden below 420px. Drag handles use `touch-action: none` so touch drag works; ▲/▼ buttons exist for users who prefer taps.

## Route map

Public:
- `GET /` — dashboard
- `GET/POST /login` — sign in (redirects to `/setup` when no users exist)
- `POST /logout`
- `GET/POST /setup` — first-run admin creation (404-equivalent once a user exists)
- `GET /icons/{file}` — from `DATA_DIR/icons/`
- `GET /styles.css`, `/scripts.js`, `/admin.js` — embedded assets

Admin (`requireAuth`):
- `GET /admin` — panel page
- `GET /admin/api/services`
- `POST /admin/api/services`
- `PUT /admin/api/services/{id}`
- `DELETE /admin/api/services/{id}`
- `POST /admin/api/services/reorder` — body `{"order":[id,id,...]}`
- `POST /admin/api/services/{id}/icon` — multipart (field `icon`, ≤2MB, png/jpg/gif/svg/webp/ico)
- `POST /admin/api/settings` — body `{"page_title":"..."}`

## Env vars

- `BIND_URL` — listen address (e.g. `:8080`; default `:8080`).
- `DATA_DIR` — where `app.db` + `icons/` live (default `./data` locally, `/data` in container).
- `RELOAD_TEMPLATES` — `true` to reparse templates per request in dev.

`.env` is loaded via `godotenv` only when `BIND_URL` isn't already set in the process environment, which lets the container keep using its `ENV` defaults without a `.env` file.
