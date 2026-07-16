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

	collapseAfter := 5
	if v, ok := ctx.Options["collapse-after"]; ok {
		switch n := v.(type) {
		case float64:
			collapseAfter = int(n)
		case int:
			collapseAfter = n
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

	// ── Render ──
	var sb strings.Builder
	showCollapse := collapseAfter > 0 && len(allArticles) > collapseAfter

	switch style {
	case "horizontal-cards":
		sb.WriteString(`<div class="rss-horizontal">`)
		for i, a := range allArticles {
			cls := ""
			if showCollapse && i >= collapseAfter {
				cls = " rss-hidden"
			}
			sb.WriteString(fmt.Sprintf(`<a href="%s" target="_blank" rel="noopener" class="rss-card%s">`,
				htmlEscape(a.URL), cls))
			if a.ImageURL != "" {
				sb.WriteString(fmt.Sprintf(`<div class="rss-card-img"><img src="%s" alt="" loading="lazy"></div>`,
					htmlEscape(a.ImageURL)))
			}
			sb.WriteString(`<div class="rss-card-body">`)
			sb.WriteString(fmt.Sprintf(`<div class="rss-card-title">%s</div>`, htmlEscape(a.Title)))
			sb.WriteString(fmt.Sprintf(`<div class="rss-card-meta">%s · %s</div>`,
				htmlEscape(a.FeedTitle), a.Published))
			sb.WriteString(`</div></a>`)
		}
		if showCollapse {
			sb.WriteString(`<button class="rss-show-more" data-rss-target="rss-horizontal">+ show more</button>`)
		}
		sb.WriteString(`</div>`)

	default: // vertical-list
		sb.WriteString(`<div class="rss-list">`)
		for i, a := range allArticles {
			cls := ""
			if showCollapse && i >= collapseAfter {
				cls = " rss-hidden"
			}
			sb.WriteString(fmt.Sprintf(`<a href="%s" target="_blank" rel="noopener" class="rss-item%s">`,
				htmlEscape(a.URL), cls))
			if a.FaviconURL != "" {
				sb.WriteString(fmt.Sprintf(`<img class="rss-fav" src="%s" alt="" loading="lazy">`, a.FaviconURL))
			}
			sb.WriteString(`<div class="rss-item-body">`)
			sb.WriteString(fmt.Sprintf(`<div class="rss-item-title">%s</div>`, htmlEscape(a.Title)))
			sb.WriteString(fmt.Sprintf(`<div class="rss-item-meta">%s · %s</div>`,
				htmlEscape(a.FeedTitle), a.Published))
			sb.WriteString(`</div></a>`)
		}
		if showCollapse {
			sb.WriteString(`<button class="rss-show-more" data-rss-target="rss-list">+ show more</button>`)
		}
		sb.WriteString(`</div>`)
	}

	inner := fmt.Sprintf(`<div class="widget-body rss-body">%s</div>`, sb.String())
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
  document.querySelectorAll(".rss-show-more").forEach(function(btn){
    btn.addEventListener("click",function(){
      var parent=this.closest(".rss-list, .rss-horizontal");
      if(!parent)return;
      parent.querySelectorAll(".rss-hidden").forEach(function(el){el.classList.remove("rss-hidden");});
      this.style.display="none";
    });
  });
})();`
}


