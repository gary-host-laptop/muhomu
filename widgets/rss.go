package widgets

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mmcdole/gofeed"
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

// mergeRoundRobin interleaves the fetched feeds one article at a time in
// config order (feed by feed, each feed's articles in their own order), so
// a busy feed can't starve the others: every working feed contributes
// evenly until limit is reached. Failed feeds are logged and skipped;
// duplicate titles across feeds are dropped. limit <= 0 means no limit.
func mergeRoundRobin(results []feedResult, limit int) []feedArticle {
	for _, res := range results {
		if res.err != nil {
			log.Printf("rss: feed fetch error: %v", res.err)
		}
	}

	var all []feedArticle
	seen := make(map[string]bool)
	next := make([]int, len(results)) // per-feed cursor

	for {
		added := 0
		for i, res := range results {
			if res.err != nil {
				continue
			}
			if limit > 0 && len(all) >= limit {
				break
			}
			for next[i] < len(res.articles) {
				a := res.articles[next[i]]
				next[i]++
				if seen[a.Title] {
					continue
				}
				seen[a.Title] = true
				all = append(all, a)
				added++
				break
			}
		}
		if added == 0 || (limit > 0 && len(all) >= limit) {
			break
		}
	}

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all
}

// sortByNewest orders articles newest-first by publish date. Items
// without a parseable date sink below dated ones; ties and undated
// items keep their prior relative order.
func sortByNewest(articles []feedArticle) {
	sort.SliceStable(articles, func(i, j int) bool {
		ti, tj := articles[i].publishedAt, articles[j].publishedAt
		if ti.Equal(tj) {
			return false
		}
		if ti.IsZero() {
			return false // undated never sorts above anything
		}
		if tj.IsZero() {
			return true // dated always sorts above undated
		}
		return ti.After(tj)
	})
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
// .htmx-request while in flight — newtab.css spins the icon.
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

func (w *RSSWidget) Render(ctx RenderContext) (template.HTML, error) {
	// ── Parse config options ──
	style := "vertical-list"
	if s, ok := ctx.Options["style"].(string); ok {
		style = s
	}

	limit := 25
	if v, ok := ctx.Options["limit"]; ok {
		switch n := v.(type) {
		case float64:
			limit = int(n)
		case int:
			limit = n
		}
	}

	itemsPerPage := 5
	if v, ok := ctx.Options["items-per-page"]; ok {
		switch n := v.(type) {
		case float64:
			itemsPerPage = int(n)
		case int:
			itemsPerPage = n
		}
	} else if v, ok := ctx.Options["collapse-after"]; ok {
		switch n := v.(type) {
		case float64:
			itemsPerPage = int(n)
		case int:
			itemsPerPage = n
		}
	}
	if itemsPerPage < 1 {
		itemsPerPage = 5
	}

	// ── Parse cache duration, feeds, poll interval ──
	cacheTTL := parseRSSCacheTTL(ctx.Options)
	feeds := parseRSSFeeds(ctx.Options, cacheTTL)
	if len(feeds) == 0 {
		return "", nil
	}
	poll := parseRSSPoll(ctx.Options)

	var allArticles []feedArticle

	if ctx.Fragment {
		// ?refresh=1 (the update button) drops cached entries so every
		// feed is re-downloaded unconditionally.
		if ctx.Query["refresh"] == "1" {
			urls := make([]string, len(feeds))
			for i, fc := range feeds {
				urls[i] = fc.URL
			}
			dropFeedCache(urls)
		}

		// ── Fetch feeds in parallel ──
		results := make([]feedResult, len(feeds))
		var wg sync.WaitGroup
		for i := range feeds {
			wg.Add(1)
			go func(idx int, fc feedConfig) {
				defer wg.Done()
				articles, err := fetchFeed(fc, ctx.FaviconDir)
				results[idx] = feedResult{articles: articles, err: err}
			}(i, feeds[i])
		}
		wg.Wait()

		allArticles = finalizeArticles(results, limit)
	} else {
		// ── Full-page render: never touch the network ──
		// Serve whatever is resident in the cache, stale entries
		// included; the poll and the background warmer keep it honest.
		// With nothing cached at all, render a placeholder that htmx
		// fills in asynchronously so page paint never waits on feeds.
		var results []feedResult
		for _, fc := range feeds {
			if arts, ok := cachedArticles(fc.URL); ok {
				results = append(results, feedResult{articles: arts})
			}
		}
		allArticles = finalizeArticles(results, limit)

		if len(allArticles) == 0 {
			body := fmt.Sprintf(`<div class="widget-body rss-body rss-loading"%s>`+
				`<div class="rss-list" id="%s"><div class="rss-placeholder">フィードを取得中…</div></div></div>`,
				rssBodyAttrs(poll, true), rssListID)
			return wrap(ctx, "rss", "フィード", rssRefreshButton, body), nil
		}
	}

	if len(allArticles) == 0 {
		return "", nil
	}

	// ── Render paginated pages ──
	totalArticles := len(allArticles)
	totalPages := (totalArticles + itemsPerPage - 1) / itemsPerPage
	if totalPages < 1 {
		totalPages = 1
	}

	var sb strings.Builder
	containerClass := "rss-list"
	if style == "horizontal-cards" {
		containerClass = "rss-horizontal"
	}

	sb.WriteString(`<div data-pager-wrap>`)

	for page := 0; page < totalPages; page++ {
		start := page * itemsPerPage
		end := start + itemsPerPage
		if end > totalArticles {
			end = totalArticles
		}
		activeCls := ""
		if page == 0 {
			activeCls = " active"
		}
		sb.WriteString(fmt.Sprintf(`<div data-page="%d" class="rss-page%s">`, page, activeCls))

		for _, a := range allArticles[start:end] {
			switch style {
			case "horizontal-cards":
				sb.WriteString(fmt.Sprintf(`<a href="%s" rel="noopener" class="rss-card">`,
					htmlEscape(a.URL)))
				if a.ImageURL != "" {
					sb.WriteString(fmt.Sprintf(`<div class="rss-card-img"><img src="%s" alt="" loading="lazy"></div>`,
						htmlEscape(a.ImageURL)))
				}
				sb.WriteString(`<div class="rss-card-body">`)
				sb.WriteString(fmt.Sprintf(`<div class="rss-card-title">%s</div>`, htmlEscape(a.Title)))
				sb.WriteString(fmt.Sprintf(`<div class="rss-card-meta">%s · %s</div>`,
					htmlEscape(a.FeedTitle), a.Published))
				sb.WriteString(`</div></a>`)

			default: // vertical-list
				sb.WriteString(fmt.Sprintf(`<a href="%s" rel="noopener" class="rss-item">`,
					htmlEscape(a.URL)))
				if a.FaviconURL != "" {
					sb.WriteString(fmt.Sprintf(`<img class="rss-fav" src="%s" alt="" loading="lazy">`, a.FaviconURL))
				}
				sb.WriteString(`<div class="rss-item-body">`)
				sb.WriteString(fmt.Sprintf(`<div class="rss-item-title">%s</div>`, htmlEscape(a.Title)))
				sb.WriteString(fmt.Sprintf(`<div class="rss-item-meta">%s · %s</div>`,
					htmlEscape(a.FeedTitle), a.Published))
				sb.WriteString(`</div></a>`)
			}
		}
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</div>`)

	// Pagination lives in the titlebar (prev / 1·N / next) so the body is
	// just the current page of items and can hold a fixed height. The
	// pager buttons are scoped from the widget root by pager.js. The
	// refresh button re-fetches every feed unconditionally via the
	// fragment endpoint.
	acts := `<button class="wt-btn" data-widget-btn="rss"><i class="ph-light ph-pencil-simple"></i></button>` +
		rssRefreshButton
	if totalPages > 1 {
		// Same caret arrows as the image / quote / kotoba next buttons.
		acts += fmt.Sprintf(`<button class="wt-act rss-pager" data-pager="prev" disabled title="previous page"><i class="ph-light ph-caret-left"></i></button>`+
			`<span class="wt-act rss-pager-ind" data-pager-indicator>1 / %d</span>`+
			`<button class="wt-act rss-pager" data-pager="next" title="next page"><i class="ph-light ph-caret-right"></i></button>`, totalPages)
	}

	// The poll attrs live on the body div and swap the whole widget, so the
	// pager in the titlebar is never stale relative to the list; pager.js
	// re-locks the indicator width on each htmx:load.
	inner := fmt.Sprintf(`<div class="widget-body rss-body"%s><div class="%s" id="%s">%s</div></div>`,
		rssBodyAttrs(poll, false), containerClass, rssListID, sb.String())
	return wrap(ctx, "rss", "フィード", acts, inner), nil
}

// ── feed fetching with TTL + ETag/Last-Modified caching ───────

// faviconSources lists the candidate favicon urls for an item link in
// priority order: the site's duckduckgo-derived icon first, then the
// feed channel's own logo. The first source that yields a valid cached
// image wins.
func faviconSources(link, feedIconURL string) []string {
	var srcs []string
	if g := GuessFaviconURL(link); g != "" {
		srcs = append(srcs, g)
	}
	if feedIconURL != "" {
		srcs = append(srcs, feedIconURL)
	}
	return srcs
}

// fetchFeed returns articles for a feed. If a valid cached entry
// exists within the TTL window, no network call is made.
// After TTL expiry, a conditional GET uses ETag/Last-Modified
// to avoid re-downloading if the feed hasn't changed.
func fetchFeed(fc feedConfig, faviconDir string) ([]feedArticle, error) {
	now := time.Now()

	// ── Check cache: return immediately if within TTL ──
	feedCacheMu.Lock()
	cached := feedCache[fc.URL]
	if cached != nil && now.Before(cached.expiresAt) {
		feedCacheMu.Unlock()
		return cached.items, nil
	}
	feedCacheMu.Unlock()

	// ── Build request with conditional headers if expired data exists ──
	req, err := http.NewRequest("GET", fc.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "muhomu/1.0")

	if cached != nil {
		if cached.etag != "" {
			req.Header.Add("If-None-Match", cached.etag)
		}
		if cached.lastModified != "" {
			req.Header.Add("If-Modified-Since", cached.lastModified)
		}
	}

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 304 — feed hasn't changed, re-use cached items with fresh TTL
	if resp.StatusCode == http.StatusNotModified && cached != nil {
		feedCacheMu.Lock()
		feedCache[fc.URL] = &cachedFeed{
			etag:         cached.etag,
			lastModified: cached.lastModified,
			items:        cached.items,
			expiresAt:    now.Add(fc.CacheDuration),
		}
		feedCacheMu.Unlock()
		return cached.items, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, fc.URL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	fp := gofeed.NewParser()
	parsed, err := fp.ParseString(string(body))
	if err != nil {
		return nil, err
	}

	sourceTitle := fc.Title
	if sourceTitle == "" {
		sourceTitle = parsed.Title
	}

	// Channel-level logo as fallback favicon source: substack and
	// friends publish one here, and duckduckgo's icon service often
	// lacks their subdomain hosts.
	feedIconURL := ""
	if parsed.Image != nil {
		feedIconURL = parsed.Image.URL
	}
	// Resolved icons are memoized per host: items of a feed share the
	// host, and a failing lookup (e.g. duckduckgo 404) would otherwise
	// be repeated per item.
	iconByHost := map[string]string{}
	itemFavicon := func(link string) string {
		host := link
		if u, err := url.Parse(link); err == nil && u.Host != "" {
			host = u.Host
		}
		if v, ok := iconByHost[host]; ok {
			return v
		}
		var v string
		for _, src := range faviconSources(link, feedIconURL) {
			if v = CacheFavicon(src, faviconDir); v != "" {
				break
			}
		}
		iconByHost[host] = v
		return v
	}

	max := fc.Limit
	if max <= 0 || max > len(parsed.Items) {
		max = len(parsed.Items)
	}

	articles := make([]feedArticle, 0, max)
	for i := 0; i < max; i++ {
		item := parsed.Items[i]
		if item.Title == "" {
			continue
		}

		link := item.Link
		if link == "" && len(item.Links) > 0 {
			link = item.Links[0]
		}

		publishedAt := time.Time{}
		if item.PublishedParsed != nil {
			publishedAt = *item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			publishedAt = *item.UpdatedParsed
		}

		published := ""
		if !publishedAt.IsZero() {
			published = publishedAt.Format("2006-01-02")
		}

		desc := ""
		if item.Description != "" {
			desc = stripTags(item.Description)
			if len(desc) > 200 {
				desc = desc[:200] + "…"
			}
		}

		imgURL := ""
		if item.Image != nil && item.Image.URL != "" {
			imgURL = item.Image.URL
		}

		faviconURL := itemFavicon(link)

		articles = append(articles, feedArticle{
			Title:       item.Title,
			URL:         link,
			Published:   published,
			Description: desc,
			FeedTitle:   sourceTitle,
			ImageURL:    imgURL,
			FaviconURL:  faviconURL,
			publishedAt: publishedAt,
		})
	}

	// ── Cache the result ──
	etag := resp.Header.Get("ETag")
	lastModified := resp.Header.Get("Last-Modified")

	feedCacheMu.Lock()
	feedCache[fc.URL] = &cachedFeed{
		etag:         etag,
		lastModified: lastModified,
		items:        articles,
		expiresAt:    now.Add(fc.CacheDuration),
	}
	feedCacheMu.Unlock()

	return articles, nil
}
