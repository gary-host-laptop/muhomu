package widgets

import (
	"crypto/md5"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

type RSSWidget struct{}

func (w *RSSWidget) ID() string { return "rss" }

type feedConfig struct {
	URL   string
	Title string
	Limit int // per-feed limit; 0 = no limit
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
		// fallback to old option name
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
		fc := feedConfig{URL: url}
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
		feeds = append(feeds, fc)
	}

	if len(feeds) == 0 {
		return "", nil
	}

	// ── Fetch and parse feeds ──
	fp := gofeed.NewParser()
	client := &http.Client{Timeout: 8 * time.Second}
	fp.Client = client

	type article struct {
		Title       string
		URL         string
		Published   string
		Description string
		FeedTitle   string
		ImageURL    string
		FaviconURL  string
	}

	var allArticles []article
	seen := make(map[string]bool)

	for _, fc := range feeds {
		parsed, err := fp.ParseURL(fc.URL)
		if err != nil {
			continue
		}
		sourceTitle := fc.Title
		if sourceTitle == "" {
			sourceTitle = parsed.Title
		}

		max := fc.Limit
		if max <= 0 || max > len(parsed.Items) {
			max = len(parsed.Items)
		}

		for i := 0; i < max; i++ {
			item := parsed.Items[i]
			if item.Title == "" {
				continue
			}
			// deduplicate by title
			if seen[item.Title] {
				continue
			}
			seen[item.Title] = true

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
				// strip tags for plain text
				desc = stripTags(item.Description)
				if len(desc) > 200 {
					desc = desc[:200] + "…"
				}
			}

			imgURL := ""
			if item.Image != nil && item.Image.URL != "" {
				imgURL = item.Image.URL
			}

			faviconURL := cacheFeedFavicon(link, ctx.FaviconDir)

			allArticles = append(allArticles, article{
				Title:       item.Title,
				URL:         link,
				Published:   published,
				Description: desc,
				FeedTitle:   sourceTitle,
				ImageURL:    imgURL,
				FaviconURL:  faviconURL,
			})
		}

		// Apply global limit
		if limit > 0 && len(allArticles) >= limit {
			allArticles = allArticles[:limit]
			break
		}
	}

	if len(allArticles) == 0 {
		return "", nil
	}

	// Apply global limit
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
		sb.WriteString(fmt.Sprintf(`<div class="rss-page%s" data-rss-page="%d">`, activeCls, page))

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
		sb.WriteString(fmt.Sprintf(`<div class="rss-pagination">`))
		sb.WriteString(`<button class="rss-nav-btn" data-rss-dir="prev" disabled>‹ prev</button>`)
		sb.WriteString(fmt.Sprintf(`<span class="rss-page-indicator">1 / %d</span>`, totalPages))
		sb.WriteString(`<button class="rss-nav-btn" data-rss-dir="next">next ›</button>`)
		sb.WriteString(`</div>`)
	}

	inner := fmt.Sprintf(`<div class="widget-body rss-body"><div class="%s">%s</div></div>`, containerClass, sb.String())
	return wrap("rss", "green", "フィード",
		`<button class="wt-btn" data-widget-btn="rss"><i class="ph-light ph-pencil-simple"></i></button>`,
		inner), nil
}

// cacheFeedFavicon fetches a feed's favicon from DuckDuckGo, caches it
// to disk, and returns a local serving path. Returns "" on failure.
func cacheFeedFavicon(feedURL, faviconDir string) string {
	parsed, err := url.Parse(feedURL)
	if err != nil || parsed.Host == "" {
		return ""
	}
	srcURL := "https://icons.duckduckgo.com/ip3/" + parsed.Hostname() + ".ico"

	// Cache key: MD5 of the source URL (same scheme as favicon.go)
	key := fmt.Sprintf("%x.ico", md5.Sum([]byte(srcURL)))
	destPath := filepath.Join(faviconDir, key)

	// Already cached
	if _, err := os.Stat(destPath); err == nil {
		return "/api/images/favicons/" + key
	}

	// Fetch and cache
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(srcURL)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return ""
	}
	defer resp.Body.Close()

	f, err := os.Create(destPath)
	if err != nil {
		return ""
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(destPath)
		return ""
	}

	return "/api/images/favicons/" + key
}

func (w *RSSWidget) Script() string {
	return `(function(){
  document.querySelectorAll(".rss-nav-btn").forEach(function(btn){
    btn.addEventListener("click",function(){
      var wrap=this.closest(".rss-body");
      if(!wrap)return;
      var container=wrap.querySelector(".rss-list, .rss-horizontal");
      if(!container)return;
      var pages=container.querySelectorAll(".rss-page");
      var current=container.querySelector(".rss-page.active");
      if(!current)return;
      var curIdx=parseInt(current.dataset.rssPage);
      var dir=this.dataset.rssDir;
      var nextIdx=dir==="next"?curIdx+1:curIdx-1;
      if(nextIdx<0||nextIdx>=pages.length)return;
      current.classList.remove("active");
      pages[nextIdx].classList.add("active");
      // update indicator
      var ind=wrap.querySelector(".rss-page-indicator");
      if(ind)ind.textContent=(nextIdx+1)+" / "+pages.length;
      // update prev disabled state
      var prev=wrap.querySelector('[data-rss-dir="prev"]');
      if(prev)prev.disabled=(nextIdx===0);
    });
  });
})();`
}


