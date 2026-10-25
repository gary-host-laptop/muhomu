# Deployment

muhomu is a single static binary — no database server, no runtime dependencies beyond the data directory.

## Pre-built image (GHCR)

The official image is published at `ghcr.io/gary-host-laptop/muhomu`. It listens on port **4444** and expects your data in `/data`:

```bash
docker run -d \
  --name muhomu \
  -p 4444:4444 \
  -v ./data:/data \
  ghcr.io/gary-host-laptop/muhomu:latest
```

Docker Compose:

```yaml
services:
  muhomu:
    image: ghcr.io/gary-host-laptop/muhomu:latest
    ports:
      - "4444:4444"
    volumes:
      - ./data:/data
    restart: unless-stopped
```

Then open `http://localhost:4444`.

> The `:z` suffix in some examples is only needed on SELinux-enabled hosts (e.g. Fedora).

### The data volume

Everything that persists lives in `/data` (config, SQLite, images, themes — see [data.md](data.md)). Back up the directory, not the container. On first run, muhomu starts with the default config if `config.yaml` is missing — and with no widgets until you declare a `columns` layout.

## Building from source

With Docker:

```bash
docker compose up -d --build
```

Or build and run the binary directly:

```bash
go build -o muhomu .
./muhomu -config ./data/config.yaml
```

Requires Go 1.25+ (see `go.mod`).

## Command-line flags

| Flag | Default | Purpose |
|---|---|---|
| `-port` | `8080` | Port to listen on |
| `-data` | `./data` | Data directory (SQLite, images, themes) |
| `-config` | `./data/config.yaml` | Path to the config file |
| `-static` | `./static` | Static files directory (only needed when running outside the repo) |

The Docker image sets `-port 4444`, `-data /data`, and `-static /app/static` for you (via `PORT`, `DATA_DIR`, and `STATIC_DIR` environment variables).

## Notes

- The page is served at `/`; widget fragments live under `/api/widget/...` and are swapped in by htmx — there is no separate API token or auth, so run it on a trusted network or behind a reverse proxy if you want access control.
- The image is built from the `Dockerfile` (Alpine final stage); it installs CA certificates and timezone data, which the RSS and weather widgets need for outbound HTTPS.
- muhomu is [AGPL-3.0](LICENSE) licensed.
