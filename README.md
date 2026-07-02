# 無ホム · muhomu

A self-hosted version of the mutabu new tab page, backed by a Go API server and SQLite.

## Requirements

- Docker + Docker Compose, or Go 1.22+

---

## Quick start with Docker

```bash
docker compose up -d
```

Then open `http://localhost:8080` in your browser.

Data is persisted in a Docker volume (`mutabu-data`). Profile and background images are stored on disk inside that volume.

---

## Quick start without Docker

```bash
go mod tidy
go run .
```

Then open `http://localhost:8080`.

By default:
- Data is stored in `./data/mutabu.db`
- Images are stored in `./data/images/`
- Static files are served from `./static/`

### Flags

```
-port   string   port to listen on (default "8080")
-data   string   path to data directory (default "./data")
-static string   path to static files (default "./static")
```

---

## API

| Method | Route | Description |
|--------|-------|-------------|
| GET | `/api/data` | All settings and data |
| POST | `/api/data` | Save settings (partial or full) |
| GET | `/api/bookmarks/search?q=` | Full-text bookmark search |
| POST | `/api/images/upload?type=profile\|bg` | Upload image (multipart) |
| DELETE | `/api/images/:filename` | Delete image |
| GET | `/api/images/:filename` | Serve image file |
| GET | `/api/weather?lat=&lon=` | Proxied weather from open-meteo |

---

## Data

All settings are stored as key-value pairs in SQLite. Bookmarks, quick access, recent, and notes have dedicated tables. Images are stored as files under `data/images/` and referenced by filename in the database.

To back up: copy the `data/` directory.
