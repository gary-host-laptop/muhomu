<div align="center">
  <h1>無ホム · muhomu</h1>
  <p>A self-hosted new-tab dashboard.</p>
</div>

---

<img src="./static/assets/images/screenshot.png" />

---

## About

muhomu replaces the browser's new-tab page with a dashboard: bookmarks, RSS feeds, notes, system stats and other widgets, arranged however you configure them.

It is a single Go binary. The interface is server-rendered HTML with [htmx](https://htmx.org/), so there is no frontend framework and no build step. Content is stored in a local SQLite database, and the layout is a YAML file you can edit by hand.

## Quick start

#### Pre-built image (GHCR)

```
docker run -d \
  --name muhomu \
  -p 4444:4444 \
  -v ./data:/data:z \
  ghcr.io/gary-host-laptop/muhomu:latest
```

Or with Docker Compose:

```yaml
services:
  muhomu:
    image: ghcr.io/gary-host-laptop/muhomu:latest
    ports:
      - "4444:4444"
    volumes:
      - ./data:/data:z
    restart: unless-stopped
```

#### Building from source

```
docker compose up -d --build
```

Or run it directly:

```
go run . -config ./data/config.yaml
```

Then open `http://localhost:4444`. To reconfigure, edit `data/config.yaml` and restart.

## Configuration

Layout and per-widget options live in `data/config.yaml`: the header strip, the columns and their sizes, widget order, and each widget's `options`. User content (bookmarks, notes, quotes, quick access, recently visited) is edited on the dashboard and stored in SQLite. Configuration is never stored in the database.

Full reference: [docs/configuration.md](docs/configuration.md)

## Documentation

- [docs/README.md](docs/README.md): index
- [docs/configuration.md](docs/configuration.md): `config.yaml` reference
- [docs/widgets.md](docs/widgets.md): each widget, its options, and where its data lives
- [docs/architecture.md](docs/architecture.md): how the server, widgets and htmx fit together
- [docs/data.md](docs/data.md): SQLite schema, image directories and data files
- [docs/deployment.md](docs/deployment.md): Docker, the GHCR image and command-line flags

## Development

```
go run . -config ./data/config.yaml
```

`main.go` wires up the server, `widgets/` holds the widget registry, and `static/` holds the frontend (CSS, htmx and per-widget JS). See [docs/architecture.md](docs/architecture.md) for details.

## License

[GNU AGPL v3](LICENSE)