package render

import (
	"fmt"
	"html/template"
	"strings"
)

// ── PageHead ────────────────────────────────────────────────────
// Emits a <style> block with all user-personalized CSS variables
// so fonts, clock theme, bg blur etc. are applied before first paint.

type PageHeadData struct {
	Theme      string // e.g. "dark"
	FontPixel  string // full CSS font-family value
	FontDoto   string // clock font
	FontClass  string // body class for clock font sizing
	BgImageURL string // random pick from bg images, empty if none
	BgBlur     bool
	ClockTheme string // "theme"|"light"|"dark"
	UILang     string
	TitleLang  string
	Username   string
}

func PageHead(d PageHeadData) template.HTML {
	var sb strings.Builder

	// Inline style overrides for personalized vars
	sb.WriteString("<style>:root{")
	if d.FontPixel != "" {
		fmt.Fprintf(&sb, "--font-pixel:%s;", d.FontPixel)
	}
	if d.FontDoto != "" {
		fmt.Fprintf(&sb, "--font-doto:%s;", d.FontDoto)
	}
	sb.WriteString("}</style>")

	// Body class string (theme + optional font class)
	classes := "theme-" + d.Theme
	if d.FontClass != "" {
		classes += " " + d.FontClass
	}
	// Clock theme class on body via data attr — applied by template directly
	_ = d.ClockTheme

	// Background image inline style
	if d.BgImageURL != "" {
		blur := ""
		if d.BgBlur {
			blur = ` class="blurred"`
		}
		fmt.Fprintf(&sb, `<div id="bg"%s><img src="%s" alt=""></div>`, blur, d.BgImageURL)
	} else {
		sb.WriteString(`<div id="bg"></div>`)
	}

	return template.HTML(sb.String())
}

// ── Quick Access Widget ─────────────────────────────────────────

func QuickAccess(items []QuickItem) template.HTML {
	var sb strings.Builder
	sb.WriteString(`<div class="quick-links">`)
	for i, item := range items {
		fav := item.Favicon
		if fav == "" {
			fav = item.Fav
		}
		fmt.Fprintf(&sb, `<a href="%s" target="_blank" draggable="true" data-drag-index="%d">
            <img class="ql-fav" src="%s" alt="">
            %s
            <span class="qa-btn-group"><button class="qa-edit"><i class="ph-light ph-pencil-simple"></i></button></span>
        </a>`, item.URL, i, fav, template.HTMLEscapeString(item.Label))
	}
	sb.WriteString(`</div>`)
	return template.HTML(sb.String())
}

// ── Bookmarks Widget ────────────────────────────────────────────

func Bookmarks(folders []BookmarkFolder) template.HTML {
	var sb strings.Builder
	sb.WriteString(`<div class="folder-list" id="folder-list">`)
	for fi, folder := range folders {
		fmt.Fprintf(&sb, `<details class="folder" draggable="true" data-drag-index="%d">
            <summary class="folder-head">
                <span class="folder-icon">◈</span>
                %s
                <span class="folder-head-btns">
                    <button class="folder-head-btn"><i class="ph-light ph-pencil-simple"></i></button>
                    <button class="folder-head-btn"><i class="ph-light ph-plus"></i></button>
                </span>
            </summary>
            <div class="folder-links grid">`, fi, template.HTMLEscapeString(folder.Folder))
		for li, link := range folder.Links {
			fav := link.Fav
			if fav == "" {
				fav = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='32' height='32'%3E%3Crect width='32' height='32' fill='%23333'/%3E%3C/svg%3E"
			}
			fmt.Fprintf(&sb, `
                <a href="%s" target="_blank" class="fav-tile" draggable="true" data-drag-index="%d">
                    <img class="fav" src="%s" alt="">
                    <span>%s</span>
                    <button class="tile-edit"><i class="ph-light ph-pencil-simple"></i></button>
                </a>`, link.URL, li, fav, template.HTMLEscapeString(link.Label))
		}
		sb.WriteString(`</div></details>`)
	}
	sb.WriteString(`</div>`)
	return template.HTML(sb.String())
}

// ── Quotes Widget ──────────────────────────────────────────────

func Quotes(quotes []Quote) template.HTML {
	if len(quotes) == 0 {
		return template.HTML(`<blockquote class="quote-block"><p id="quote-text">no quotes</p><footer id="quote-author"></footer></blockquote>`)
	}
	q := quotes[0]
	return template.HTML(fmt.Sprintf(`
        <blockquote class="quote-block">
            <p id="quote-text">%s</p>
            <footer id="quote-author">%s</footer>
        </blockquote>`, template.HTMLEscapeString(q.Text), template.HTMLEscapeString(q.Author)))
}

// ── Notes Widget ───────────────────────────────────────────────

func Notes(content string) template.HTML {
	return template.HTML(fmt.Sprintf(`<textarea id="notes" placeholder="// type here. auto-saves.">%s</textarea>`,
		template.HTMLEscapeString(content)))
}

// ── Recently Visited Widget ────────────────────────────────────

func Recent(items []RecentItem) template.HTML {
	var sb strings.Builder
	sb.WriteString(`<div class="recent-grid" id="recent-grid">`)
	for _, item := range items {
		fmt.Fprintf(&sb, `
            <div class="recent-tile">
                <div class="rt-name">%s</div>
                <div class="rt-url">%s</div>
                <button class="recent-x"><i class="ph-light ph-x"></i></button>
            </div>`, template.HTMLEscapeString(item.Name), template.HTMLEscapeString(item.URL))
	}
	sb.WriteString(`</div>`)
	return template.HTML(sb.String())
}

// ── Image Widget ───────────────────────────────────────────────

func WidgetImages(images []Image) template.HTML {
	var sb strings.Builder
	sb.WriteString(`<div class="widget-img-wrap" id="widget-img-wrap">`)
	for i, img := range images {
		active := ""
		if i == 0 {
			active = ` active`
		}
		fmt.Fprintf(&sb, `<img src="%s" alt="" class="%s">`, img.URL, active)
	}
	sb.WriteString(`</div>`)
	return template.HTML(sb.String())
}

// ── Status Widget ──────────────────────────────────────────────

func Status(day, week, year, temp string) template.HTML {
	return template.HTML(fmt.Sprintf(`
        <div class="status-rows">
            <div class="status-row"><span class="label">day</span><span class="value" id="day-name">%s</span></div>
            <div class="status-row"><span class="label">week</span><span class="value" id="week-num">%s</span></div>
            <div class="status-row"><span class="label">year</span><span class="value" id="year-progress">%s</span></div>
            <div class="status-row"><span class="label">temp</span><span class="value" id="temperature">%s</span></div>
        </div>`, day, week, year, temp))
}

// ── System Stats Widget ────────────────────────────────────────

func SystemStats() template.HTML {
	return template.HTML(`<canvas id="stats-canvas" width="300" height="150"></canvas>`)
}

// ── Kotoba Widget ──────────────────────────────────────────────

func Kotoba(word Word) template.HTML {
	return template.HTML(fmt.Sprintf(`
        <div class="kanji-block">
            <div class="kanji-char" id="kanji-char"><a href="https://jisho.org/search/%s%%20%%23kanji" target="_blank" style="color:inherit;text-decoration:none;">%s</a></div>
            <div class="kanji-reading" id="kanji-reading">%s</div>
            <div class="kanji-divider"></div>
            <div class="kanji-meta"><span class="kanji-level %s" id="kanji-level">%s</span></div>
            <div class="kanji-meaning" id="kanji-meaning">%s</div>
        </div>`, word.K, word.K, word.R, word.L, word.L, word.M))
}

// ── Search Engines ─────────────────────────────────────────────

type Engine struct {
	Name      string
	URL       string
	IsDefault bool
}

func SearchEngines(engines []Engine) template.HTML {
	if len(engines) == 0 {
		return template.HTML(`<button class="engine-btn active" data-url="https://duckduckgo.com/?q=">duckduckgo</button>`)
	}
	var sb strings.Builder
	for _, eng := range engines {
		active := ""
		if eng.IsDefault {
			active = " active"
		}
		fmt.Fprintf(&sb, `<button class="engine-btn%s" data-url="%s">%s</button>`,
			active, template.HTMLEscapeString(eng.URL), template.HTMLEscapeString(eng.Name))
	}
	return template.HTML(sb.String())
}

// ── Timer Widget ───────────────────────────────────────────────

func Timer() template.HTML {
	return template.HTML(`
        <div class="timer-display" id="timer-display">00:00</div>
        <div class="timer-inputs">
            <div>
                <input type="number" class="timer-input" id="timer-min" value="0" min="0" max="99" placeholder="00">
                <div class="timer-label">min</div>
            </div>
            <span class="timer-sep">:</span>
            <div>
                <input type="number" class="timer-input" id="timer-sec" value="0" min="0" max="59" placeholder="00">
                <div class="timer-label">sec</div>
            </div>
        </div>
        <button id="timer-reset" style="display:none;"></button>
    `)
}

// ── Rain Player Widget ─────────────────────────────────────────

func RainPlayer() template.HTML {
	return template.HTML(`
        <div class="rain-player">
            <div class="rain-track rain-master">
                <span class="rain-track-label">vol</span>
                <input type="range" class="rain-slider rain-slider-master" id="vol-master" min="0" max="1" step="0.01" value="1">
            </div>
            <div class="rain-divider"></div>
            <div class="rain-track">
                <span class="rain-track-label">雨</span>
                <input type="range" class="rain-slider" id="vol-rain" min="0" max="1" step="0.01" value="0.7">
            </div>
            <div class="rain-track">
                <span class="rain-track-label">風</span>
                <input type="range" class="rain-slider" id="vol-wind" min="0" max="1" step="0.01" value="0.4">
            </div>
            <div class="rain-track">
                <span class="rain-track-label">雷</span>
                <input type="range" class="rain-slider" id="vol-thunder" min="0" max="1" step="0.01" value="0.5">
            </div>
        </div>
    `)
}

// ── Full widget wrappers ────────────────────────────────────────
// Each function returns the complete <div class="widget"> block
// so the template can render them in any order via the layout map.

func WidgetQuickAccess(items []QuickItem) template.HTML {
	inner := QuickAccess(items)
	return template.HTML(fmt.Sprintf(`
<div class="widget" data-widget="quick-access">
  <div class="widget-title">
    <div class="wt-bar blue"></div>
    <span class="wt-label" data-widget-label="quick-access">クイックアクセス</span>
    <button class="wt-btn" id="qa-add-btn"><i class="ph-light ph-plus"></i></button>
  </div>
  <div class="widget-body" style="padding:6px 10px">%s</div>
</div>`, inner))
}

func WidgetBookmarks(folders []BookmarkFolder) template.HTML {
	inner := Bookmarks(folders)
	return template.HTML(fmt.Sprintf(`
<div class="widget" data-widget="bookmarks">
  <div class="widget-title">
    <div class="wt-bar red"></div>
    <span class="wt-label" data-widget-label="bookmarks">ブックマーク</span>
    <button class="wt-btn" id="bm-add-folder-btn"><i class="ph-light ph-plus"></i></button>
  </div>
  <div class="widget-body">%s</div>
</div>`, inner))
}

func WidgetNotes(content string) template.HTML {
	inner := Notes(content)
	return template.HTML(fmt.Sprintf(`
<div class="widget" data-widget="notes">
  <div class="widget-title">
    <div class="wt-bar green"></div>
    <span class="wt-label" data-widget-label="notes">メモ帳</span>
    <button class="wt-act" id="notes-clear"><i class="ph-light ph-trash"></i></button>
  </div>
  <div class="widget-body">%s</div>
</div>`, inner))
}

func WidgetRecent(items []RecentItem) template.HTML {
	inner := Recent(items)
	return template.HTML(fmt.Sprintf(`
<div class="widget" data-widget="recently-visited">
  <div class="widget-title">
    <div class="wt-bar pink"></div>
    <span class="wt-label" data-widget-label="recently-visited">後で見る</span>
  </div>
  <div class="widget-body">
    %s
    <div class="recent-quick-add">
      <input type="text" id="rqa-name" placeholder="label" autocomplete="off" />
      <input type="text" id="rqa-url" placeholder="https://..." autocomplete="off" />
      <button id="rqa-btn"><span>+</span></button>
    </div>
  </div>
</div>`, inner))
}

func WidgetImage(imageURL string) template.HTML {
	imgHTML := ""
	filename := ""
	if imageURL != "" {
		filename = imageURL[strings.LastIndex(imageURL, "/")+1:]
		imgHTML = fmt.Sprintf(`<img src="%s" alt="" id="widget-img" class="active" data-filename="%s">`, imageURL, filename)
	}
	return template.HTML(fmt.Sprintf(`
<div class="widget image-widget" data-widget="image">
  <div class="widget-title">
    <div class="wt-bar red"></div>
    <span class="wt-label" data-widget-label="image">イメージ</span>
    <button class="wt-act" id="widget-img-next" title="random image"><i class="ph-light ph-shuffle"></i></button>
  </div>
  <div class="widget-img-inner">
    <div class="widget-img-wrap" id="widget-img-wrap">%s</div>
  </div>
</div>`, imgHTML))
}

func WidgetStatus(day, week, year, temp string) template.HTML {
	inner := Status(day, week, year, temp)
	return template.HTML(fmt.Sprintf(`
<div class="widget" data-widget="status">
  <div class="widget-title">
    <div class="wt-bar red"></div>
    <span class="wt-label" data-widget-label="status">状態</span>
  </div>
  <div class="widget-body">%s</div>
</div>`, inner))
}

func WidgetSystemStats() template.HTML {
	return template.HTML(`
<div class="widget" data-widget="system-stats">
  <div class="widget-title">
    <div class="wt-bar blue"></div>
    <span class="wt-label" data-widget-label="system-stats">システム</span>
  </div>
  <div class="widget-body" style="padding:6px">
    <canvas id="stats-canvas" width="300" height="150"></canvas>
  </div>
</div>`)
}

func WidgetKotoba(word Word) template.HTML {
	inner := Kotoba(word)
	return template.HTML(fmt.Sprintf(`
<div class="widget" data-widget="kotoba">
  <div class="widget-title">
    <div class="wt-bar pink"></div>
    <span class="wt-label" data-widget-label="kotoba">言葉</span>
    <button class="wt-act" id="kanji-next"><i class="ph-light ph-caret-right"></i></button>
  </div>
  %s
</div>`, inner))
}

func WidgetTimer() template.HTML {
	inner := Timer()
	return template.HTML(fmt.Sprintf(`
<div class="widget" data-widget="timer">
  <div class="widget-title">
    <div class="wt-bar green"></div>
    <span class="wt-label" data-widget-label="timer">タイマー</span>
    <button class="wt-act" id="timer-trash" title="clear timer"><i class="ph-light ph-trash"></i></button>
    <button class="wt-act" id="timer-start"><i class="ph-light ph-play"></i></button>
  </div>
  %s
</div>`, inner))
}

func WidgetRain() template.HTML {
	inner := RainPlayer()
	return template.HTML(fmt.Sprintf(`
<div class="widget" data-widget="rain">
  <div class="widget-title">
    <div class="wt-bar blue"></div>
    <span class="wt-label" data-widget-label="rain">雨音</span>
    <button class="wt-act" id="rain-btn"><i class="ph-light ph-play"></i></button>
  </div>
  %s
</div>`, inner))
}

func WidgetQuote(q Quote) template.HTML {
	inner := Quotes([]Quote{q})
	return template.HTML(fmt.Sprintf(`
<div class="widget" data-widget="quote">
  <div class="widget-title">
    <div class="wt-bar blue"></div>
    <span class="wt-label" data-widget-label="quote">名言</span>
    <button class="wt-act" id="quote-next"><i class="ph-light ph-caret-right"></i></button>
  </div>
  <div class="widget-body" style="padding:8px">%s</div>
</div>`, inner))
}
