# Data

All persistent state lives under the data directory (default `./data`), which is the volume you mount when running in Docker.

```
data/
├── config.yaml            # the whole configuration (see configuration.md)
├── quotes.yaml            # fallback quotes for the quote widget
├── muhomu.db              # SQLite database — all user content
├── images/
│   ├── profile/           # profile images
│   ├── bg/                # background images
│   └── favicons/          # cached favicons, served at /api/images/favicons/
├── widget-images/         # images for the image widget
└── themes/                # custom theme CSS files
```

## SQLite (`data/muhomu.db`)

Opened with WAL mode. The schema contains **only user-produced content** — configuration never lives in the database.

| Table | Purpose |
|---|---|
| `bookmarks` | Folders + links: `folder`, `label`, `url`, `favicon`, `position`, `folder_order` |
| `quick_access` | Quick-access links: `label`, `url`, `favicon`, `position` |
| `recent` | Recently visited entries: `name`, `url`, `added_at` |
| `notes` | The single note (one row, `id = 1`): `content` |
| `quotes` | Quotes for the quote widget: `text`, `author` |
| `bookmarks_fts` | An FTS5 full-text index over bookmarks, kept in sync by triggers; backs the `/api/bookmarks/search` endpoint |

There is no admin surface for the database — content is edited on the dashboard (edit mode) and the server writes it for you. Configuration is edited via the settings page at `/settings` (see [configuration.md](configuration.md)).

## Quotes (`data/quotes.yaml`)

The quote widget prefers quotes from the `quotes` table in the database; if the table is empty, it falls back to `quotes.yaml`:

```yaml
quotes:
  - text: "The tradition of all dead generations weighs like a nightmare on the brains of the living."
    author: "Marx"
```

## Images

Drop files into the host-mounted directories and they become available — or upload them from the settings page (`/settings`, data section), which writes into these same directories:

- `profile-images/` → `profile_images_dir` (default `./data/images/profile`)
- `bg-images/` → `bg_images_dir` (default `./data/images/bg`)
- `widget-images/` → `widget_images_dir` (default `./data/widget-images`)

Supported formats: jpg, jpeg, png, webp, gif, avif.

Favicons are written automatically into `data/images/favicons/` by the favicon cache (see [architecture.md](architecture.md)); you don't manage these by hand.

## Themes

Custom themes are CSS files in `themes_dir` (default `./data/themes`), applied with the `theme_file` config key or from the settings. The base `dark`/`light` themes live in the binary under `static/theme.css`.

## Backups

Because everything is files, a backup is just the data directory (plus your config if it lives elsewhere):

```
tar czf muhomu-backup.tgz ./data
```
