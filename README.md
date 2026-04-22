# homelab-browser

A small, self-hosted dashboard for the services running on your homelab. One page of tiles, a gated admin panel to manage them, and a top bar with a live clock and the local weather. SQLite-backed, single binary, single Docker volume.

## Introduction

A single page listing every service you self-host — Plex, Sonarr, Home Assistant, your NAS, whatever — as clean clickable cards. The whole dashboard sits behind a login, so it's just for you (and anyone you hand credentials to).

- **Managed from the browser.** Add services, rearrange them, upload custom icons — works just as well on your phone.
- **Everything in one folder.** The database and your uploaded icons share a single directory. Moving to a new host is copying that folder.
- **Private by default.** No telemetry, no third-party accounts, no API keys. The weather shown in the top bar is fetched by your browser directly from [Open-Meteo](https://open-meteo.com/) — the server never sees it.

## Features

- 🧭 **One page, all your services.** Every app you self-host shows up as a clickable tile. Click, go.
- 🔐 **Private to you.** The whole dashboard is behind a username and password — nobody else sees your list of services. One login lasts 90 days and refreshes itself as you use the app.
- 🛠 **Manage everything from the browser.** Add, rename, and delete services right from the admin panel.
- 🖼 **Upload your own icons.** PNG, SVG, JPG, WebP, GIF, or ICO, up to 2 MB.
- 🔀 **Reorder your way.** Drag tiles around on desktop, tap ▲/▼ on mobile. Both save instantly.
- 🌤 **Clock and local weather up top**, refreshed live.
- 🪟 **Open tabs the way you want.** Each service can open in a new tab or replace the current page — set per service.
- 📱 **Made for phones too.** The admin panel works one-handed, with a full-screen edit dialog and chunky tap targets.
- 🛡 **Your data stays yours.** No telemetry, no cloud account, no API keys. Weather comes straight from your browser to [Open-Meteo](https://open-meteo.com/) — the server never sees your location.

## Installation

### Docker Compose (recommended)

```bash
git clone https://github.com/Azmekk/homelab-browser.git
cd homelab-browser
cp docker-compose.yml.example docker-compose.yml
docker compose up -d
```

Open `http://localhost:8080` and you'll be redirected to `/setup` to create the initial admin account.

The `./data` directory next to the compose file will hold `app.db` and `icons/` — that's your entire backup surface.

### Docker (standalone)

```bash
docker build -t homelab-browser .
docker run -d \
    --name homelab-browser \
    -p 8080:8080 \
    -v "$(pwd)/data:/data" \
    --restart unless-stopped \
    homelab-browser
```

### From source

Requires Go 1.23 or newer.

```bash
git clone https://github.com/Azmekk/homelab-browser.git
cd homelab-browser/src
go build -o ../bin/homelab-browser .
BIND_URL=:8080 DATA_DIR=./data ../bin/homelab-browser
```

Build artifacts are written to `/bin/` at the repo root (gitignored).

## Configuration

All configuration is via environment variables. A `.env` file in `src/` is read automatically when `BIND_URL` isn't already set in the process environment — handy for local dev, irrelevant in containers.

| Variable           | Default (source) | Default (Docker) | Purpose |
|--------------------|------------------|------------------|---------|
| `BIND_URL`         | `:8080`          | `:8080`          | Listen address, e.g. `:8080` or `127.0.0.1:9000`. |
| `DATA_DIR`         | `./data`         | `/data`          | Where `app.db` and `icons/` live. |
| `RELOAD_TEMPLATES` | unset            | unset            | Set to `true` to reparse HTML templates on every request (dev only). |

### Data directory layout

```
<DATA_DIR>/
  app.db            SQLite database (services, users, sessions, settings)
  app.db-wal        WAL file (SQLite journaling)
  app.db-shm        Shared memory index
  icons/
    1-a3f9…png      Uploaded service icons (named `<id>-<random>.<ext>`)
    2-b7c2…svg
```

To back up, stop the container and `tar czf homelab.tar.gz ./data`. To migrate to a new host, copy the directory.

## First-run setup

The app ships with **no default credentials** — it refuses to be useful until you create an admin account.

1. Boot the container / binary.
2. Visit `http://<host>:8080` — you'll be redirected straight to the setup page.
3. Choose a username (≥ 2 characters) and a password (≥ 8 characters). Confirm the password.
4. You're dropped into the admin panel, logged in.

From there: set your page title, click **+ Add service**, and upload some icons.

Once an admin exists, `/setup` redirects back to `/login` — it's a one-shot endpoint.

## Development

All commands run from the `src/` directory.

```bash
# Run locally (needs go 1.23+)
go run .

# Rebuild the binary
go build -o homelab-browser .

# Regenerate sqlc code after editing db/schema.sql or db/queries.sql
#   one-time install:  go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
sqlc generate
```

For template hot-reload during UI work, run with `RELOAD_TEMPLATES=true` so every request re-parses `wwwroot/templates/*.html`.

### Project layout

```
src/
  main.go              chi router wiring + env loading
  auth.go              bcrypt + session cookie + requireAuth middleware
  store.go             SQLite bootstrap (embedded schema.sql) + thin wrapper
  templates.go         embedded template loader + static asset handler
  handlers_public.go   /, /login, /setup, /logout
  handlers_admin.go    /admin and /admin/api/* (JSON)
  sqlc.yaml            sqlc config
  db/
    schema.sql         source of truth for tables
    queries.sql        hand-written queries with sqlc annotations
    *.go               generated by `sqlc generate` (checked in)
  wwwroot/
    styles.css         plain CSS with :root custom properties — no build step
    scripts.js         dashboard clock + weather (vanilla JS)
    admin.js           admin Alpine.js component
    templates/         html/template sources (embedded at build time)
```

See `CLAUDE.md` for deeper architecture notes.

## Tech stack & credits

Server:

- [Go](https://go.dev/) + [chi](https://github.com/go-chi/chi) for routing and middleware.
- [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) — pure-Go SQLite driver, so no CGO, no libsqlite to install.
- [sqlc](https://sqlc.dev/) for generating type-safe Go from SQL.
- [`golang.org/x/crypto/bcrypt`](https://pkg.go.dev/golang.org/x/crypto/bcrypt) for password hashing.
- [godotenv](https://github.com/joho/godotenv) for `.env` in local dev.

Client (all loaded from CDN on the admin page only):

- [Alpine.js](https://alpinejs.dev/) for the admin panel reactivity.
- [SortableJS](https://sortablejs.github.io/Sortable/) for drag-to-reorder.

External data:

- [Open-Meteo](https://open-meteo.com/) for weather and reverse geocoding (no API key required).
- [ipapi.co](https://ipapi.co/) as an IP-based coordinate fallback when geolocation is unavailable.
