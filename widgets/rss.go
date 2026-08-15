package widgets

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
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
}

// feedResult carries the outcome of fetching one feed.
type feedResult struct {
	articles []feedArticle
	err      error
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

	// ── Parse cache duration ──
	cacheTTL := defaultCacheTTL
	if v, ok := ctx.Options["cache"]; ok {
		if s, ok := v.(string); ok {
			if d, err := time.ParseDuration(s); err == nil && d > 0 {
				cacheTTL = d
			}
		}
	}

	// ── Parse feeds ──
	rawFeeds, ok := ctx.Options["feeds"].([]interface{})
	if !ok || len(rawFeeds) == 0 {
		return "", nil
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

	if len(feeds) == 0 {
		return "", nil
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

	// ── Collect articles, skip failed feeds ──
	var allArticles []feedArticle
	seen := make(map[string]bool)

	for _, res := range results {
		if res.err != nil {
			log.Printf("rss: feed fetch error: %v", res.err)
			continue
		}
		for _, a := range res.articles {
			if seen[a.Title] {
				continue
			}
			seen[a.Title] = true
			allArticles = append(allArticles, a)
		}
		if limit > 0 && len(allArticles) >= limit {
			allArticles = allArticles[:limit]
			break
		}
	}

	if len(allArticles) == 0 {
		return "", nil
	}

	if limit > 0 && len(allArticles) > limit {
		allArticles = allArticles[:limit]
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

	// ── Pagination nav ──
	if totalPages > 1 {
		sb.WriteString(`<div class="rss-pagination">`)
		sb.WriteString(`<button class="rss-nav-btn" data-pager="prev" disabled>‹ prev</button>`)
		sb.WriteString(fmt.Sprintf(`<span class="rss-page-indicator" data-pager-indicator>1 / %d</span>`, totalPages))
		sb.WriteString(`<button class="rss-nav-btn" data-pager="next">next ›</button>`)
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</div>`)

	inner := fmt.Sprintf(`<div class="widget-body rss-body"><div class="%s">%s</div></div>`, containerClass, sb.String())
	return wrap(ctx, "rss", "フィード",
		`<button class="wt-btn" data-widget-btn="rss"><i class="ph-light ph-pencil-simple"></i></button>`,
		inner), nil
}

// ── feed fetching with TTL + ETag/Last-Modified caching ───────

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

		published := ""
		if item.PublishedParsed != nil {
			published = item.PublishedParsed.Format("2006-01-02")
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

		faviconURL := CacheFavicon(GuessFaviconURL(link), faviconDir)

		articles = append(articles, feedArticle{
			Title:       item.Title,
			URL:         link,
			Published:   published,
			Description: desc,
			FeedTitle:   sourceTitle,
			ImageURL:    imgURL,
			FaviconURL:  faviconURL,
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
