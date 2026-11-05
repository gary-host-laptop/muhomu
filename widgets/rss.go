package widgets

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// ── Feed cache ────────────────────────────────────────────────
// Each feed is cached with a TTL (default 2h). Within the window
// no network call is made. After expiry, a conditional GET via
// ETag/Last-Modified tries for a 304 before re-downloading.
type cachedFeed struct {
	etag         string
	lastModified string
	items        []feedArticle
	expiresAt    time.Time
}

var (
	feedCache   = map[string]*cachedFeed{}
	feedCacheMu sync.Mutex
)

// defaultCacheTTL is used when no custom "cache" option is set.
const defaultCacheTTL = 2 * time.Hour

// defaultPollInterval is how often an open tab re-requests the rss
// fragment when no custom "poll" option is set.
const defaultPollInterval = time.Hour

// rssListID anchors htmx swaps: the poll/refresh/load requests all
// select this element out of the response and swap it into the widget
// body, so the titlebar (pager + refresh button) survives every update.
const rssListID = "rss-list"

type RSSWidget struct{}

func (w *RSSWidget) ID() string { return "rss" }

type feedConfig struct {
	URL           string
	Title         string
	Limit         int // per-feed limit; 0 = no limit
	CacheDuration time.Duration
}

type feedArticle struct {
	Title       string
	URL         string
	Published   string
	Description string
	FeedTitle   string
	ImageURL    string
	FaviconURL  string

	// publishedAt is the raw publish/update timestamp used for date
	// ordering; zero for items without a parseable date.
	publishedAt time.Time
}

// feedResult carries the outcome of fetching one feed.
type feedResult struct {
	articles []feedArticle
	err      error
}

// ── option parsing (shared by Render and WarmRSSCache) ────────

func parseRSSCacheTTL(opts map[string]interface{}) time.Duration {
	ttl := defaultCacheTTL
	if s, ok := opts["cache"].(string); ok {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			ttl = d
		}
	}
	return ttl
}

func parseRSSFeeds(opts map[string]interface{}, cacheTTL time.Duration) []feedConfig {
	rawFeeds, ok := opts["feeds"].([]interface{})
	if !ok {
		return nil
	}
	var feeds []feedConfig
	for _, raw := range rawFeeds {
		m, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		url, _ := m["url"].(string)
		if url == "" {
			continue
		}
		fc := feedConfig{URL: url, CacheDuration: cacheTTL}
		if t, ok := m["title"].(string); ok {
			fc.Title = t
		}
		if v, ok := m["limit"]; ok {
			switch n := v.(type) {
			case float64:
				fc.Limit = int(n)
			case int:
				fc.Limit = n
			}
		}
		// Per-feed cache override
		if v, ok := m["cache"]; ok {
			if s, ok := v.(string); ok {
				if d, err := time.ParseDuration(s); err == nil && d > 0 {
					fc.CacheDuration = d
				}
			}
		}
		feeds = append(feeds, fc)
	}
	return feeds
}

func parseRSSPoll(opts map[string]interface{}) time.Duration {
	poll := defaultPollInterval
	if s, ok := opts["poll"].(string); ok {
		if d, err := time.ParseDuration(s); err == nil {
			poll = d // an explicit 0 / negative disables polling
		}
	}
	return poll
}

// ── feed cache accessors ──────────────────────────────────────

// cachedArticles returns the articles resident in the feed cache for
// url regardless of TTL expiry (stale-while-revalidate). ok is false
// when nothing has ever been fetched for url.
func cachedArticles(url string) (articles []feedArticle, ok bool) {
	feedCacheMu.Lock()
	defer feedCacheMu.Unlock()
	c := feedCache[url]
	if c == nil || len(c.items) == 0 {
		return nil, false
	}
	return c.items, true
}

// dropFeedCache forgets the cached entry for each url so the next
// fetchFeed call re-downloads unconditionally — the ?refresh=1 path.
func dropFeedCache(urls []string) {
	feedCacheMu.Lock()
	for _, u := range urls {
		delete(feedCache, u)
	}
	feedCacheMu.Unlock()
}

// finalizeArticles merges feed results round-robin, orders them
// newest-first and truncates to the global limit. Merging runs without
// a limit so truncation happens AFTER sorting — otherwise older
// entries from an early feed could crowd out genuinely newer ones.
func finalizeArticles(results []feedResult, limit int) []feedArticle {
	all := mergeRoundRobin(results, 0)
	sortByNewest(all)
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all
}

// ── htmx plumbing ─────────────────────────────────────────────

// rssBodyAttrs drives live updates on the widget. Each update re-fetches
// the whole widget and swaps the widget root (closest .widget + outerHTML)
// — the full-widget swap used everywhere (see the convention above wrap in
// widget.go). The attrs sit on the body div, which is re-rendered with the
// widget, so each poll re-arms exactly one timer. includeLoad adds a
// one-shot load trigger, used by the placeholder to pull its first real
// content.
func rssBodyAttrs(poll time.Duration, includeLoad bool) string {
	triggers := ""
	if includeLoad {
		triggers = "load"
	}
	if poll > 0 {
		if triggers != "" {
			triggers += ", "
		}
		triggers += fmt.Sprintf("every %dms", poll.Milliseconds())
	}
	if triggers == "" {
		return ""
	}
	return fmt.Sprintf(` hx-get="/api/widget/rss" hx-trigger="%s" hx-target="closest .widget" hx-swap="outerHTML"`,
		triggers)
}

// rssRefreshButton re-fetches every feed unconditionally (?refresh=1)
// and swaps the re-rendered widget into place. htmx tags the button with
// .htmx-request while in flight — static/css/rss.css spins the icon.
const rssRefreshButton = `<button class="wt-btn rss-refresh" title="refresh now" ` +
	`hx-get="/api/widget/rss?refresh=1" hx-target="closest .widget" hx-swap="outerHTML">` +
	`<i class="ph-light ph-arrows-clockwise"></i></button>`

// WarmRSSCache refreshes any of the widget's feeds whose cached copy is
// missing or past its TTL — exactly what a fragment render would do,
// minus rendering. A periodic caller keeps feeds fresh even while no
// browser is open, so full-page renders almost always find warm cache.
// Overlapping invocations are skipped rather than queued.
var rssWarming atomic.Bool

func WarmRSSCache(opts map[string]interface{}, faviconDir string) {
	if !rssWarming.CompareAndSwap(false, true) {
		return
	}
	defer rssWarming.Store(false)

	feeds := parseRSSFeeds(opts, parseRSSCacheTTL(opts))
	var wg sync.WaitGroup
	for _, fc := range feeds {
		wg.Add(1)
		go func(fc feedConfig) {
			defer wg.Done()
			if _, err := fetchFeed(fc, faviconDir); err != nil {
				log.Printf("rss: warm fetch error: %v", err)
			}
		}(fc)
	}
	wg.Wait()
}
