# 無ホム · muhomu

A self-hosted new tab dashboard backed by a Go server, SQLite, and a yaml configuration file. Renders widgets server-side (SSR-first) with minimal client-side javascript.

## Requirements

- Docker + Docker Compose, or Go 1.22+

---

## Quick start with Docker

```bash
# create host directories for images and config
mkdir -p profile-images bg-images widget-images

# copy and edit the config
cp config.yaml.example config.yaml

docker compose up -d --build
```

Then open `http://localhost:4444`.

Data is persisted in a named Docker volume (`mutabu-data`). Image directories are bind-mounted from the host so you can drop files directly without restarting.

---

## Quick start without Docker

```bash
go mod tidy
go run . -config ./config.yaml
```

Then open `http://localhost:4444`.

### Flags

```
-port           string   port to listen on (default "8080")
-data           string   path to data directory (default "./data")
-static         string   path to static files (default "./static")
-config         string   path to config file (default "./config.yaml")
-widget-images  string   path to widget images directory (default: <data>/widget-images)
```

---

## Configuration

All layout and appearance is declared in `config.yaml`. Edit the file and restart to apply changes. No settings UI — configuration is infrastructure, not runtime state.

```yaml
theme: dark               # dark | light
font_latin: inter         # inter | share-tech-mono | vt323 | courier-prime
font_jp: dotgothic16      # dotgothic16 | biz-udgothic | noto-sans-jp
font_clock: orbitron      # orbitron | medodica | oxanium
clock_format: 24h         # 24h | 12h
clock_seconds: false
bg_blur: true
username: ""
search_target: _blank

location:
  city: ""
  lat: ""
  lon: ""

search_engines:
  - name: duckduckgo
    url: "https://duckduckgo.com/?q="
    default: true

jlpt_level: all           # n1 | n2 | n3 | n4 | n5 | all

widgets:
  - id: bookmarks
    col: center
    order: 1
  - id: calendar
    col: right
    order: 1
  # available: quick-access, bookmarks, notes, recently-visited,
  #            quote, kotoba, system-stats, timer, rain, image, calendar
```

A widget's presence in the list is the declaration of its existence — absent widgets produce no html, no javascript scope, no dom nodes.

---

## Images

Images are managed by dropping files into host directories. No upload UI.

| Directory | Purpose | Limit |
|-----------|---------|-------|
| `profile-images/` | Profile picture (one shown randomly per load) | unlimited |
| `bg-images/` | Background image (one shown randomly per load) | unlimited |
| `widget-images/` | Image widget (shuffle button cycles through) | unlimited |

Supported formats: jpg, jpeg, png, webp, gif, avif.

---

## User data

User-produced content (bookmarks, quick access, recent history, notes, quotes) is stored in SQLite and edited directly on the dashboard. Configuration is never stored in the database.

| Table | Description |
|-------|-------------|
| `bookmarks` | Folders and links, full-text search via FTS5 |
| `quick_access` | Quick access bar links |
| `recent` | Recently visited (last 8) |
| `notes` | Single persistent note |
| `quotes` | Quote collection for the quote widget |

Favicons are fetched and cached locally under `data/images/favicons/` on bookmark save, eliminating repeated external requests.

---

## API

| Method | Route | Description |
|--------|-------|-------------|
| GET | `/api/data` | All user data (bookmarks, notes, quick, recent, quotes) |
| POST | `/api/data` | Save user data mutations |
| GET | `/api/bookmarks/search?q=` | Full-text bookmark search |
| GET | `/api/quotes` | List quotes |
| POST | `/api/quotes` | Add quote `{text, author}` |
| DELETE | `/api/quotes/{id}` | Delete quote |
| GET | `/api/stats` | CPU, RAM, network stats (system-stats widget) |
| GET | `/api/weather?lat=&lon=` | Proxied weather from open-meteo |
| GET | `/api/widget-images/next?current=` | Random widget image (excluding current) |
| GET | `/api/images/profile/` | Serve profile images |
| GET | `/api/images/bg/` | Serve background images |
| GET | `/api/images/favicons/` | Serve cached favicons |
| GET | `/api/widget-images/files/` | Serve widget images |

---

## Backup

Copy the `data/` directory. Contains the SQLite database and cached favicons. Image directories (`profile-images/`, `bg-images/`, `widget-images/`) are on the host and not inside the volume.

---

## New tab redirect

Use a browser extension to redirect new tab to `http://localhost:4444`. With [New Tab Override](https://addons.mozilla.org/en-US/firefox/addon/new-tab-override/) (Firefox), set the url to `http://localhost:4444`.
