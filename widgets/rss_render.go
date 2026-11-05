package widgets

import (
	"html/template"
	"sync"
)

type rssPageView struct {
	Index  int
	Active bool
	Items  []feedArticle
}

type rssPagesView struct {
	Horizontal bool
	Pages      []rssPageView
}

type rssWidgetView struct {
	Attrs          template.HTMLAttr
	ContainerClass string
	ListID         string
	Pages          template.HTML
}

type rssLoadingView struct {
	Attrs  template.HTMLAttr
	ListID string
}

type rssActionsView struct {
	RefreshButton template.HTML
	TotalPages    int
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
			body, err := renderWidgetTemplate("rss-loading.tmpl", rssLoadingView{
				Attrs:  template.HTMLAttr(rssBodyAttrs(poll, true)),
				ListID: rssListID,
			})
			if err != nil {
				return "", err
			}
			return wrap(ctx, "rss", "フィード", rssRefreshButton, string(body)), nil
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

	horizontal := style == "horizontal-cards"
	containerClass := "rss-list"
	if horizontal {
		containerClass = "rss-horizontal"
	}

	pages := make([]rssPageView, totalPages)
	for page := 0; page < totalPages; page++ {
		start := page * itemsPerPage
		end := start + itemsPerPage
		if end > totalArticles {
			end = totalArticles
		}
		pages[page] = rssPageView{
			Index:  page,
			Active: page == 0,
			Items:  allArticles[start:end],
		}
	}

	pagesHTML, err := renderWidgetTemplate("rss-pages.tmpl", rssPagesView{
		Horizontal: horizontal,
		Pages:      pages,
	})
	if err != nil {
		return "", err
	}

	// Pagination lives in the titlebar (prev / 1·N / next) so the body is
	// just the current page of items and can hold a fixed height. The
	// pager buttons are scoped from the widget root by pager.js. The
	// refresh button re-fetches every feed unconditionally via the
	// fragment endpoint.
	acts, err := renderWidgetTemplate("rss-actions.tmpl", rssActionsView{
		RefreshButton: template.HTML(rssRefreshButton),
		TotalPages:    totalPages,
	})
	if err != nil {
		return "", err
	}

	// The poll attrs live on the body div and swap the whole widget, so the
	// pager in the titlebar is never stale relative to the list; pager.js
	// re-locks the indicator width on each htmx:load.
	inner, err := renderWidgetTemplate("rss-widget.tmpl", rssWidgetView{
		Attrs:          template.HTMLAttr(rssBodyAttrs(poll, false)),
		ContainerClass: containerClass,
		ListID:         rssListID,
		Pages:          pagesHTML,
	})
	if err != nil {
		return "", err
	}
	return wrap(ctx, "rss", "フィード", string(acts), string(inner)), nil
}
