package widgets

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func titles(articles []feedArticle) []string {
	out := make([]string, len(articles))
	for i, a := range articles {
		out[i] = a.Title
	}
	return out
}

func TestMergeRoundRobinInterleavesFeeds(t *testing.T) {
	results := []feedResult{
		{articles: []feedArticle{{Title: "A1"}, {Title: "A2"}, {Title: "A3"}, {Title: "A4"}}},
		{articles: []feedArticle{{Title: "B1"}, {Title: "B2"}}},
		{articles: []feedArticle{{Title: "C1"}}},
	}
	got := titles(mergeRoundRobin(results, 0)) // unlimited
	want := []string{"A1", "B1", "C1", "A2", "B2", "A3", "A4"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-robin order wrong:\n got %v\nwant %v", got, want)
	}
}

func TestMergeRoundRobinRespectsLimit(t *testing.T) {
	results := []feedResult{
		{articles: []feedArticle{{Title: "A1"}, {Title: "A2"}, {Title: "A3"}, {Title: "A4"}}},
		{articles: []feedArticle{{Title: "B1"}, {Title: "B2"}}},
		{articles: []feedArticle{{Title: "C1"}}},
	}
	got := titles(mergeRoundRobin(results, 4))
	want := []string{"A1", "B1", "C1", "A2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("limit not respected:\n got %v\nwant %v", got, want)
	}
}

func TestMergeRoundRobinSkipsFailedFeed(t *testing.T) {
	results := []feedResult{
		{articles: []feedArticle{{Title: "A1"}, {Title: "A2"}}},
		{err: errors.New("boom")},
		{articles: []feedArticle{{Title: "C1"}}},
	}
	got := titles(mergeRoundRobin(results, 0))
	want := []string{"A1", "C1", "A2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("failed feed not skipped:\n got %v\nwant %v", got, want)
	}
}

func TestMergeRoundRobinDropsDuplicateTitles(t *testing.T) {
	results := []feedResult{
		{articles: []feedArticle{{Title: "Same"}}},
		{articles: []feedArticle{{Title: "Same"}, {Title: "B1"}}},
	}
	got := titles(mergeRoundRobin(results, 0))
	want := []string{"Same", "B1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("duplicates not dropped:\n got %v\nwant %v", got, want)
	}
}

func TestMergeRoundRobinEmpty(t *testing.T) {
	if got := mergeRoundRobin(nil, 25); len(got) != 0 {
		t.Fatalf("expected no articles, got %v", got)
	}
	if got := mergeRoundRobin([]feedResult{{err: errors.New("down")}}, 25); len(got) != 0 {
		t.Fatalf("expected no articles from a failed feed, got %v", got)
	}
}

func TestSortByNewestOrdersNewestFirst(t *testing.T) {
	day := func(n int) time.Time { return time.Date(2026, 8, n, 0, 0, 0, 0, time.UTC) }
	articles := []feedArticle{
		{Title: "old", publishedAt: day(1)},
		{Title: "newest", publishedAt: day(20)},
		{Title: "middle", publishedAt: day(10)},
	}
	sortByNewest(articles)
	want := []string{"newest", "middle", "old"}
	if got := titles(articles); !reflect.DeepEqual(got, want) {
		t.Fatalf("not sorted newest-first:\n got %v\nwant %v", got, want)
	}
}

func TestSortByNewestSinksUndatedItems(t *testing.T) {
	day := func(n int) time.Time { return time.Date(2026, 8, n, 0, 0, 0, 0, time.UTC) }
	articles := []feedArticle{
		{Title: "undated"},
		{Title: "old", publishedAt: day(1)},
		{Title: "newest", publishedAt: day(20)},
	}
	sortByNewest(articles)
	want := []string{"newest", "old", "undated"}
	if got := titles(articles); !reflect.DeepEqual(got, want) {
		t.Fatalf("undated items not sunk to the bottom:\n got %v\nwant %v", got, want)
	}
}

func TestSortByNewestKeepsTiesStable(t *testing.T) {
	same := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	articles := []feedArticle{
		{Title: "first", publishedAt: same},
		{Title: "second", publishedAt: same},
		{Title: "third", publishedAt: same},
	}
	sortByNewest(articles)
	want := []string{"first", "second", "third"}
	if got := titles(articles); !reflect.DeepEqual(got, want) {
		t.Fatalf("stable order not preserved:\n got %v\nwant %v", got, want)
	}
}

func TestFinalizeArticlesTruncatesAfterSorting(t *testing.T) {
	day := func(n int) time.Time { return time.Date(2026, 8, n, 0, 0, 0, 0, time.UTC) }
	results := []feedResult{
		// feed A is listed first but holds older articles
		{articles: []feedArticle{{Title: "old", publishedAt: day(1)}, {Title: "mid", publishedAt: day(5)}}},
		{articles: []feedArticle{{Title: "new1", publishedAt: day(20)}, {Title: "new2", publishedAt: day(15)}}},
	}
	got := titles(finalizeArticles(results, 2))
	want := []string{"new1", "new2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("limit applied before sorting:\n got %v\nwant %v", got, want)
	}
}

// ── render modes ──────────────────────────────────────────────

// seedFeed plants articles in the global feed cache and removes them
// when the test finishes.
func seedFeed(t *testing.T, url string, items []feedArticle, ttl time.Duration) {
	t.Helper()
	feedCacheMu.Lock()
	feedCache[url] = &cachedFeed{items: items, expiresAt: time.Now().Add(ttl)}
	feedCacheMu.Unlock()
	t.Cleanup(func() {
		feedCacheMu.Lock()
		delete(feedCache, url)
		feedCacheMu.Unlock()
	})
}

func feedsOption(urls ...string) map[string]interface{} {
	var fs []interface{}
	for _, u := range urls {
		fs = append(fs, map[string]interface{}{"url": u})
	}
	return map[string]interface{}{"feeds": fs}
}

func TestIndexRenderServesStaleCacheWithoutNetwork(t *testing.T) {
	seedFeed(t, "https://example.com/feed", []feedArticle{
		{Title: "cached item", URL: "https://example.com/a"},
	}, -1*time.Hour) // expired — must still be served (stale-while-revalidate)

	html, err := (&RSSWidget{}).Render(RenderContext{Options: feedsOption("https://example.com/feed")})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "cached item") {
		t.Fatalf("stale cached article not rendered: %s", html)
	}
	if strings.Contains(string(html), `hx-trigger="load`) {
		t.Fatalf("warm index render must not carry the load placeholder trigger: %s", html)
	}
}

func TestIndexRenderColdShowsPlaceholder(t *testing.T) {
	html, err := (&RSSWidget{}).Render(RenderContext{Options: feedsOption("https://never-fetched.example/feed")})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), `hx-trigger="load`) {
		t.Fatalf("cold index render missing load trigger: %s", html)
	}
	if !strings.Contains(string(html), "rss-placeholder") {
		t.Fatalf("cold index render missing placeholder body: %s", html)
	}
	if !strings.Contains(string(html), "?refresh=1") {
		t.Fatalf("placeholder titlebar missing refresh button: %s", html)
	}
	if !strings.Contains(string(html), `hx-target="closest .widget"`) {
		t.Fatalf("placeholder update must swap the whole widget: %s", html)
	}
}

func TestFragmentRenderServesFreshCacheWithoutNetwork(t *testing.T) {
	seedFeed(t, "https://example.com/feed", []feedArticle{
		{Title: "fresh item", URL: "https://example.com/b"},
	}, time.Hour)

	html, err := (&RSSWidget{}).Render(RenderContext{
		Options:  feedsOption("https://example.com/feed"),
		Fragment: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "fresh item") {
		t.Fatalf("fragment render did not serve fresh cache: %s", html)
	}
	if !strings.Contains(string(html), `hx-trigger="every `) {
		t.Fatalf("fragment render missing poll trigger: %s", html)
	}
}

func TestDropFeedCacheClearsOnlyGivenURLs(t *testing.T) {
	seedFeed(t, "https://a.example/feed", []feedArticle{{Title: "a"}}, time.Hour)
	seedFeed(t, "https://b.example/feed", []feedArticle{{Title: "b"}}, time.Hour)

	dropFeedCache([]string{"https://a.example/feed"})

	if _, ok := cachedArticles("https://a.example/feed"); ok {
		t.Fatalf("dropped url still cached")
	}
	if _, ok := cachedArticles("https://b.example/feed"); !ok {
		t.Fatalf("unrelated cache entry was removed")
	}
}

// ── htmx attrs + option parsing ───────────────────────────────

func TestRSSBodyAttrs(t *testing.T) {
	if got := rssBodyAttrs(0, false); got != "" {
		t.Fatalf("disabled poll should emit no attrs, got %q", got)
	}
	got := rssBodyAttrs(90*time.Minute, false)
	if !strings.Contains(got, `hx-trigger="every 5400000ms"`) {
		t.Fatalf("poll duration not emitted in ms: %q", got)
	}
	if !strings.Contains(got, `hx-target="closest .widget"`) || !strings.Contains(got, `hx-swap="outerHTML"`) {
		t.Fatalf("poll must swap the whole widget root: %q", got)
	}
	if strings.Contains(got, "hx-select") {
		t.Fatalf("poll must use the full fragment, not a selected sub-element: %q", got)
	}
	if !strings.Contains(rssRefreshButton, `hx-target="closest .widget"`) {
		t.Fatalf("refresh button must swap the whole widget root: %q", rssRefreshButton)
	}
	load := rssBodyAttrs(time.Hour, true)
	if !strings.Contains(load, `load, every `) {
		t.Fatalf("placeholder attrs missing one-shot load trigger: %q", load)
	}
}

func TestParseRSSPoll(t *testing.T) {
	if got := parseRSSPoll(map[string]interface{}{}); got != defaultPollInterval {
		t.Fatalf("default poll = %v, want %v", got, defaultPollInterval)
	}
	if got := parseRSSPoll(map[string]interface{}{"poll": "15m"}); got != 15*time.Minute {
		t.Fatalf("custom poll = %v, want 15m", got)
	}
	if got := parseRSSPoll(map[string]interface{}{"poll": "0"}); got != 0 {
		t.Fatalf("explicit zero should disable polling, got %v", got)
	}
}

// ── favicon fallback ──────────────────────────────────────────

func TestFaviconSourcesPrefersDDGThenFeedLogo(t *testing.T) {
	logo := "https://substackcdn.com/image/logo.png"
	srcs := faviconSources("https://dialecticaldispatches.substack.com/p/x", logo)
	want := []string{
		"https://icons.duckduckgo.com/ip3/dialecticaldispatches.substack.com.ico",
		logo,
	}
	if !reflect.DeepEqual(srcs, want) {
		t.Fatalf("wrong fallback order:\n got %v\nwant %v", srcs, want)
	}
	if srcs := faviconSources("", ""); len(srcs) != 0 {
		t.Fatalf("no sources expected for empty link/logo, got %v", srcs)
	}
}
