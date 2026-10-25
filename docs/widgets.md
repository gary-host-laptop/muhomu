# Widgets

Every widget is an independent, server-rendered unit with a title bar (LED + label + actions) and a body. Widgets are activated simply by listing them in `columns` in `data/config.yaml` (see [configuration.md](configuration.md)). This page describes what each one shows, how it is configured, and where its data lives.

## Header widgets

Header widgets render in the header strip via the `header` config section, chrome-less (no title bar / box). They also implement the normal widget interface, so any of them can instead be placed in a body column, where they render with the standard chrome.

### clock
The big clock and the date line. The client (`newtab.js`) fills both by id every second, honoring `clock_format` / `clock_seconds`.

- **Options:** `led`
- **JS:** none (driven by `static/newtab.js`)

### identity
The user's identity block: edit/settings buttons, the greeting + username, and the barcode decoration. The greeting is filled by the client (`newtab.js`); the username comes from config.

- **Data:** `username` from `config.yaml`
- **Options:** `led`
- **JS:** none (driven by `static/newtab.js`)

### profile
The profile picture (avatar) with corner decorations, picked randomly from the profile images directory.

- **Data:** files in `profile_images_dir` (default `./data/images/profile`)
- **Options:** `led`

### search
The search bar + engine buttons. In the header it stretches to fill the leftover horizontal space. If it is not placed in the header and not in any column, it renders in its classic full-width strip below the header (with the accent divider).

- **Data:** `search_engines` from `config.yaml`
- **Options:** `led`
- **JS:** none (driven by `static/newtab.js`)

## Content widgets

### quick-access
A vertical list of links with favicons. Links are added, edited, removed, and reordered from the page (edit mode).

- **Data:** `quick_access` table in `data/muhomu.db`
- **Options:** `led`
- **JS:** `static/js/widget-quick-access.js`

### bookmarks
A folder-based bookmark manager: folders at the root level, links inside folders. Folder tiles are draggable to reorder; entering a folder is a pure htmx swap (`/api/widget/bookmarks?folder=N`), and the inside-folder grid scrolls when the folder outgrows its fixed height.

- **Data:** `bookmarks` table in `data/muhomu.db`
- **Options:** `rows` (fixed body height in rows of tiles, default 3), `led`
- **JS:** `static/js/widget-bookmarks.js`

### notes
A single persistent note, edited in place.

- **Data:** `notes` table in `data/muhomu.db` (a single row, `id = 1`)
- **Options:** `led`
- **JS:** `static/js/widget-notes.js`

### recently-visited
A list of recently visited sites, managed from the page.

- **Data:** `recent` table in `data/muhomu.db`
- **Options:** `led`
- **JS:** `static/js/widget-recently-visited.js`

### rss
A feed reader with pagination (prev / page counter / next live in the title bar). Feeds are fetched server-side with a TTL cache and conditional requests (ETag / Last-Modified), so it is polite to the sources.

Fetching is fully decoupled from page load: the full-page render only ever serves cached articles (stale entries included) and shows a "loading" placeholder that htmx fills in when nothing is cached yet. A background warmer refreshes all configured feeds on a ticker, and an open tab picks up new articles by polling the widget fragment (`poll`, default `1h`). The refresh button in the title bar re-downloads every feed immediately, bypassing the cache.

The two timing knobs: `cache` controls how often the *server* refetches a feed (the update latency), `poll` controls how often an *open tab* checks the server. With defaults (`2h` / `1h`) new items surface within ~2h of publication; set e.g. `cache: 30m` + `poll: 10m` for snappier behavior.

- **Data:** feeds configured in `config.yaml`; no database involvement
- **Options:** `style`, `limit`, `items-per-page`, `cache`, `poll`, `feeds`, `led`
- **JS:** none — pagination (`static/js/pager.js`, shared), polling, and refresh are pure htmx

### quote
Shows a rotating quote with a "next" button that avoids repeating the current one.

- **Data:** the `quotes` table in `data/muhomu.db` takes precedence; otherwise quotes from `data/quotes.yaml`
- **Options:** `led`
- **JS:** none (next button is pure htmx)

### kotoba
Shows a Japanese word (kanji, reading, meaning) drawn from an embedded JLPT vocabulary list (N5–N1), with a "next word" button.

- **Data:** embedded word list in `widgets/kotoba.go`
- **Options:** `led`
- **JS:** none (next button is pure htmx)

## Ambient / system widgets

### system-stats
Live CPU and RAM sparklines with percentage gridlines (0/25/50/75%), sampled from `/proc` every 2 seconds and re-rendered via htmx polling. The bottom band is RAM, the top band is CPU; 100% is deliberately not drawn.

- **Data:** sampled from the host (`/proc/stat`, `/proc/meminfo`, network counters)
- **Options:** `led`
- **JS:** none (htmx polling)

### timer
A simple countdown timer. A `vol` slider at the top of the body sets the alarm's volume (persisted in `localStorage`); it is the same control row the rain widget uses, and both widgets share a fixed body height (`--audio-body-h`).

- **Options:** `led`
- **JS:** `static/js/widget-timer.js`

### rain
An ambient rain widget (sound + visual), with play controls.

- **Options:** `led`
- **JS:** `static/js/widget-rain.js`

### weather
Current weather from [open-meteo](https://open-meteo.com/), fetched server-side and cached until the next hour.

- **Data:** open-meteo API, located by `location` in `config.yaml`
- **Options:** `led`

### image
Cycles through images in the widget images directory, stepping sequentially with an htmx "next" button.

- **Data:** files in `widget_images_dir` (default `./data/widget-images`)
- **Options:** `led`
- **JS:** none (next button is pure htmx)

### calendar
A server-rendered calendar for the current month.

- **Options:** `led`

## Layout widget

### split-column
Not a content widget — it nests other widgets side by side in a configurable number of columns.

- **Options:** `max-columns`, `widgets` (a nested list of widget configs), `led`

See [configuration.md](configuration.md) for option types and defaults.
