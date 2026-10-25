# Configuration (`data/config.yaml`)

Everything that is *configuration* lives in one YAML file, `data/config.yaml` (or whatever you pass with `-config`). Configuration is never stored in the database — user content is.

If the file does not exist, muhomu starts with its defaults and renders **no widgets** (an empty `columns` list). If the file fails to parse, defaults are used.

The file is **hot-reloaded**: save an edit and it is picked up within ~2 seconds, no restart needed (a failed parse — e.g. mid-save — keeps the running config and retries). Exceptions: the data-directory paths (`profile_images_dir`, `bg_images_dir`, `widget_images_dir`, `themes_dir`) and the CLI flags (`-port`, `-data`, `-static`, `-config`) are resolved only at startup.

## Settings page

There is also a settings page at **`/settings`** (reachable via the gear icon in the header). It is a GUI over this file and the other config files — it never touches YAML itself; the server does all reading and writing (`settings.go`):

- **`GET /api/settings`** — returns the current config, the defaults, the quotes file, the available themes, and the image listings as JSON.
- **`POST /api/settings`** — merges a (partial) config JSON over the running config, validates that the result still parses as YAML, writes `config.yaml` atomically (temp file + rename), and applies it immediately.
- **`POST /api/settings/reset`** — writes the built-in default config.
- **`POST /api/settings/quotes`** — saves `quotes.yaml`.
- **`POST` / **`DELETE /api/settings/images`** — upload/delete files in the profile/bg/widget image directories.

The page is a schema-driven vanilla JS app: it covers every key in this reference as a form — including the layout editor (header, columns, per-widget options such as `rss` feeds and `split-column` children), search engines, footer items, quotes, themes, and image uploads. The YAML reference below remains the full truth; the page is just an alternate way to edit it.

## Top-level keys

| Key | Type | Default | Purpose |
|---|---|---|---|
| `theme` | string | `dark` | Base theme: `dark` or `light` |
| `font_latin` | string | `inter` | Latin font family |
| `font_jp` | string | `dotgothic16` | Japanese font family |
| `font_clock` | string | `orbitron` | Clock font family |
| `clock_format` | string | `24h` | `24h` or `12h` |
| `clock_seconds` | bool | `false` | Show seconds in the clock |
| `clock_theme` | string | `theme` | Clock styling |
| `bg_blur` | bool | `true` | Blur the background image |
| `ui_lang` | string | `en` | Interface language (`en`/`ja`) |
| `title_lang` | string | `ja` | Language for the page title |
| `username` | string | — | Shown in the header |
| `search_target` | string | `_self` | `_self` or `_blank` for search results |
| `default_led` | string | `var(--accent)` | Default widget LED color (CSS color) |
| `jlpt_level` | string | `all` | JLPT level filter (`all`, `n5`–`n1`) |
| `location.city` | string | — | City name (weather) |
| `location.lat` / `location.lon` | string | — | Coordinates (weather fallback) |
| `search_engines` | list | DDG/YouTube/GitHub | `{name, url, default}` entries |
| `footer_items` | list | — | Footer stats shown left-to-right in list order: `rx`, `tx`, `cpu`, `ram`. The muhomu version and GitHub link always render at the far right. |
| `profile_images_dir` | string | `./data/images/profile` | Profile images |
| `bg_images_dir` | string | `./data/images/bg` | Background images |
| `widget_images_dir` | string | `./data/widget-images` | Image widget images |
| `themes_dir` | string | `./data/themes` | Custom theme CSS files |
| `theme_file` | string | — | Load a specific theme file |
| `header` | list | defaults | The header strip (see below) |
| `columns` | list | defaults | The layout (see below) |

## Header

`header` is the ordered list of widgets shown in the header strip — the same `{id, options}` shape as column widgets. Header widgets render *chrome-less*: no title bar, no LED, no box; they sit directly in the strip.

The header is **always one row** and widgets are **fixed-size** — except `search`, which stretches to fill the leftover horizontal space. With the default order, the layout reads: clock left, search filling the middle, identity + profile grouped right.

```yaml
header:
  - id: clock
  - id: search
  - id: identity
  - id: profile
```

Available header widgets:

| Widget | Shows |
|---|---|
| `clock` | The big clock + date line |
| `search` | The search bar + engine buttons (stretches to fill space) |
| `identity` | Edit/settings buttons, greeting, username, barcode |
| `profile` | The profile picture (avatar) with corner deco |

If `header` is absent, the default is `clock`, `identity`, `profile`.

The `search` widget is special: if it is **not** placed in the header and not in any column, it renders in its classic position — a full-width strip below the header (with the accent divider). Placing `search` in the header or in a column moves it there and suppresses the strip.

Any body widget can also be placed in the header; it will render there with its full chrome (title bar + box) instead of chrome-less.

## Columns

`columns` is the authoritative declaration of layout. Each column declares a `size` and the ordered list of widgets it contains. Order in the list is display order.

```yaml
columns:
  - size: small          # 250px fixed
    widgets:
      - id: image
      - id: quote
  - size: full           # remaining space (1fr)
    widgets:
      - id: bookmarks
        options:
          rows: 3
      - id: notes
  - size: small
    widgets:
      - id: system-stats
      - id: calendar
```

- `size`: `small` (250px) or `full` (the remaining width).
- `widgets`: an ordered list of `{id, options}` entries.

### Widget options

Every widget accepts an `options` map. Two are common to all widgets:

- `led` — the LED dot color in the title bar, as a CSS color (e.g. `#ff5555`). Falls back to `default_led`, then to `var(--accent)`.

Widget-specific options:

| Widget | Option | Type | Default | Description |
|---|---|---|---|---|
| `bookmarks` | `rows` | int | `3` | Fixed body height in rows of folder tiles (1–10). Scrolls internally when content overflows. |
| `rss` | `style` | string | `vertical-list` | `vertical-list` or `horizontal-cards` |
| `rss` | `limit` | int | `25` | Max articles per feed (0 = no limit) |
| `rss` | `items-per-page` | int | `5` | Articles per page (`collapse-after` is an alias) |
| `rss` | `cache` | string | `2h` | Feed cache TTL (Go duration, e.g. `30m`, `4h`) — how often the server refetches |
| `rss` | `poll` | string | `1h` | How often an open tab re-requests the widget fragment (Go duration, `0` disables) |
| `rss` | `feeds` | list | — | `{url, title, limit, cache}` entries |
| `split-column` | `max-columns` | int | — | Cap on the number of inner columns |
| `split-column` | `widgets` | list | — | Nested `{id, options}` widget list |

The RSS widget renders nothing until at least one feed is configured:

```yaml
- id: rss
  options:
    style: vertical-list
    limit: 10
    items-per-page: 5
    feeds:
      - url: https://example.com/feed.xml
        title: example
      - url: https://other.example/rss
        title: other
        limit: 20        # per-feed override
        cache: 30m       # per-feed override
```

`split-column` nests other widgets side by side:

```yaml
- id: split-column
  options:
    max-columns: 2
    widgets:
      - id: timer
      - id: quick-access
```

## Available widgets

Body widgets: `quick-access`, `bookmarks`, `notes`, `recently-visited`, `rss`, `quote`, `kotoba`, `system-stats`, `timer`, `rain`, `weather`, `image`, `calendar`, `split-column`.

Header widgets: `clock`, `identity`, `profile`, `search`.

See [widgets.md](widgets.md) for what each one does and where its data lives.
