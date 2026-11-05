package widgets

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/mmcdole/gofeed"

	"muhomu/internal/media"
)

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

// ── feed fetching with TTL + ETag/Last-Modified caching ───────

// faviconSources lists the candidate favicon urls for an item link in
// priority order: the site's duckduckgo-derived icon first, then the
// feed channel's own logo. The first source that yields a valid cached
// image wins.
func faviconSources(link, feedIconURL string) []string {
	var srcs []string
	if g := media.GuessFaviconURL(link); g != "" {
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
			if v = media.CacheFavicon(src, faviconDir); v != "" {
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
