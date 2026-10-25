# Architecture

muhomu is a single Go binary that serves a server-rendered dashboard. The frontend is [htmx](https://htmx.org/) plus a small amount of vanilla JS; there is no build step and no frontend framework.

## The shape of a request

1. `serveIndex` (`main.go`) walks `columns` from `config.yaml`, renders every widget through `widgets.Registry()`, and assembles the page template.
2. The rendered page is cached in memory for 5 seconds (`pageCache`), so a reload burst doesn't re-render everything.
3. The browser receives plain HTML. Widgets with JS load their `static/js/widget-<id>.js`; everything else is pure server rendering + htmx.

## The widget interface

Every widget lives in `widgets/` and implements:

```go
type Widget interface {
    ID() string
    Render(ctx RenderContext) (template.HTML, error)
}
```

- `ID()` is the widget's `data-widget` attribute and its config key.
- `Render()` returns the complete widget HTML (or `""` to render nothing).

`RenderContext` carries everything a widget may need — the `*sql.DB`, a per-request RNG, the image/favicon directories, config values (`JLPTLevel`, location, quotes, `Options`), the latest system stats snapshot, and per-request query parameters (`Query`). Widgets take only what they need; the coordinator knows nothing about individual widgets.

`wrap()` (in `widgets/widget.go`) produces the shared chrome every widget is wrapped in: the root `<div class="widget" data-widget="...">`, the title bar (LED, label, action buttons), and the body container. An optional JSON state blob is embedded as `data-state` on the root so client JS can read the widget's data model after server-side re-renders — the Go renderer is always the single source of truth.

## htmx fragment endpoints

Two endpoints in `widget_api.go` power all interactivity:

- **`GET /api/widget/{id}`** — renders one widget fresh, honoring query params (`?current=` for quote/kotoba/image, `?folder=N` for bookmarks).
- **`POST /api/widget/{id}`** — persists the widget's full state (a JSON `state` form field) and returns the re-rendered widget. Used by bookmarks / quick-access / recently-visited after client-side mutations.

Both resolve the widget's `options` from the configured columns (including nested `split-column` children) via `widgetConfigFor`, so config changes apply to htmx renders too.

Examples:

- Bookmark folder navigation is a pure `hx-get` swap of the whole widget (`hx-target="closest .widget" hx-swap="outerHTML"`).
- The quote/kotoba/image "next" buttons are `hx-get` with the current value as a query param.
- `system-stats` polls itself every 2s (`hx-trigger="every 2s"`), selecting just its graph out of the response so the swap is content-only and the body never resizes or loses its polling attributes.

## Widget JS

Widgets that need client behavior (dialogs, drag-reorder, timers, audio) have a `static/js/widget-<id>.js` file, loaded only for active widgets. The files follow a common pattern: read state from the widget's `data-state`, collect dialog input, POST the new state via `htmx.ajax`, and let the server re-render.

### The off-screen swap technique (bookmarks)

Bookmark folder changes swap the whole widget. Firefox can paint the freshly-inserted markup in a pre-layout state (icons top-left, tiles unsized), and uncached favicons can pop in after the swap. `widget-bookmarks.js` handles this by:

1. intercepting `htmx:beforeSwap`,
2. rendering the response into an off-screen, hidden holder and forcing a layout pass there,
3. waiting for every favicon image to load (or fail, with a timeout),
4. swapping the settled widget into place atomically.

The result: no layout flash and no icon pop-in, even on first open.

## Caching layers

- **Page cache** — the rendered `/` page is cached for 5s in memory.
- **Feed cache** — RSS responses are cached per-feed with a TTL (default 2h) and refreshed with conditional requests (`ETag` / `Last-Modified`). Full-page renders never fetch: they serve cached entries (stale included) or a placeholder, while a background warmer (`startRSSWarmer` in `handlers.go`) refreshes all configured feeds every minute and open tabs poll the widget fragment for updates.
- **Weather cache** — cached until the start of the next hour.
- **Favicon cache** — remote favicons are downloaded once and stored under an md5-based filename in `data/images/favicons/`, served at `/api/images/favicons/` with `Cache-Control: public, max-age=86400` (safe because a filename always maps to the same bytes). A background migration on startup upgrades any legacy remote favicon URLs to local copies.

## System stats

A background goroutine (`startStatsCollector` in `handlers.go`) samples CPU, RAM, and network counters from `/proc` every 2 seconds into a ring buffer of 31 points. The system-stats widget renders them as two SVG sparklines (CPU on top, RAM below) with percentage gridlines at 0/25/50/75%. Each 2s poll swaps in a new frame; a CSS transform glide slides the newest point in from the right while the oldest exits left, so the graph reads as one continuous scroll rather than a tick.

## Settings API

`/settings` is a standalone page that edits the configuration files through a small API (`settings.go`):

- `GET /api/settings` returns the current config, the defaults, the quotes file, the available themes, and the image listings as JSON.
- `POST /api/settings` merges a (partial) config JSON over the running config, validates that the result parses as YAML, writes `config.yaml` atomically (temp file + rename), and applies it immediately via `setConfig` + cache invalidation.
- `POST /api/settings/reset`, `POST /api/settings/quotes`, and the image upload/delete endpoints cover the rest.

The page itself is a schema-driven vanilla JS app (`static/settings.js`): forms are declared in a schema object, and the layout editor reuses a nested widget-list editor for the header, columns, and `split-column` children. A new config key needs only a schema entry — no server change.

## Themes

The base theme (`dark`/`light`) sets CSS custom properties in `theme.css`; theme colors are inlined into the page as `:root` variables at render time (the `CriticalStyle` block). Additional themes are plain CSS files in `themes_dir` (default `./data/themes`) applied via the `theme_file` config key or the settings UI.
