package widgets

import (
	"fmt"
	"html/template"
	"strings"
)

// ── Header widgets ────────────────────────────────────────────
// The header strip is configured like the body: a list of widget ids
// in config's `header` section (default: clock, identity, profile).
// Header widgets implement HeaderWidget — their RenderHeader returns
// content WITHOUT the standard widget chrome (title bar, LED, box),
// because the header is a bare horizontal strip. They also implement
// the ordinary Widget interface, so the same widget can be placed in a
// body column, where it renders with the normal chrome.

// ClockWidget renders the big clock and the date line. The client
// (newtab.js) fills both by id every second.
type ClockWidget struct{}

func (w *ClockWidget) ID() string { return "clock" }

func clockInner() template.HTML {
	return template.HTML(`<div class="clock-block">
    <div id="clock">00:00:00</div>
    <div id="date">loading...</div>
</div>`)
}

func (w *ClockWidget) RenderHeader(ctx RenderContext) (template.HTML, error) {
	return clockInner(), nil
}

func (w *ClockWidget) Render(ctx RenderContext) (template.HTML, error) {
	return wrap(ctx, "clock", "時計", "", `<div class="widget-body">`+string(clockInner())+`</div>`), nil
}

// IdentityWidget renders the user's identity block: the edit/settings
// buttons, the greeting + username, and the barcode deco.
// The client (newtab.js / i18n.js) fills the greeting by id; the
// username comes from config.
type IdentityWidget struct{}

func (w *IdentityWidget) ID() string { return "identity" }

func identityInner(username string) template.HTML {
	return template.HTML(`<div class="header-right-text">
    <div class="header-btns">
        <button id="edit-toggle" title="edit mode">
            <i class="ph-light ph-pencil-simple"></i>
        </button>
        <a href="/settings" id="settings-btn" title="settings" aria-label="settings">
            <i class="ph-light ph-gear-six"></i>
        </a>
    </div>
    <div class="header-name-block">
        <span id="greeting">こんにちは</span>
        <span id="username">` + htmlEscape(username) + `</span>
    </div>
    <span class="header-deco">▌│█│▌│█ ▌│▌█</span>
</div>`)
}

func (w *IdentityWidget) RenderHeader(ctx RenderContext) (template.HTML, error) {
	return identityInner(ctx.Username), nil
}

func (w *IdentityWidget) Render(ctx RenderContext) (template.HTML, error) {
	return wrap(ctx, "identity", "アイデンティティ", "", `<div class="widget-body">`+string(identityInner(ctx.Username))+`</div>`), nil
}

// ProfileWidget renders the profile picture (avatar) with its corner
// decorations. The image URL is picked by the coordinator (serveIndex)
// from the profile images directory.
type ProfileWidget struct{}

func (w *ProfileWidget) ID() string { return "profile" }

func profileInner(imgURL string) template.HTML {
	img := ""
	if imgURL != "" {
		img = `<img src="` + htmlEscape(imgURL) + `" alt="" style="width:100%;height:100%;object-fit:cover;">`
	}
	return template.HTML(`<div id="header-avatar-wrap">
    <div id="header-avatar">` + img + `</div>
    <span class="av-corner tl"></span>
    <span class="av-corner tr"></span>
    <span class="av-corner bl"></span>
    <span class="av-corner br"></span>
</div>`)
}

func (w *ProfileWidget) RenderHeader(ctx RenderContext) (template.HTML, error) {
	return profileInner(ctx.ProfileImageURL), nil
}

func (w *ProfileWidget) Render(ctx RenderContext) (template.HTML, error) {
	return wrap(ctx, "profile", "プロフィール", "", `<div class="widget-body">`+string(profileInner(ctx.ProfileImageURL))+`</div>`), nil
}

// SearchWidget renders the search bar and the engine buttons. The
// client (newtab.js) wires the input and the engine selection by
// class/id. It can live in the header strip or in the body (its
// classic full-width position below the header is a page-level strip
// that renders this widget's header content).
type SearchWidget struct{}

func (w *SearchWidget) ID() string { return "search" }

func searchInner(engines []SearchEngine) template.HTML {
	var sb strings.Builder
	if len(engines) == 0 {
		sb.WriteString(`<button class="engine-btn active" data-url="https://duckduckgo.com/?q=">duckduckgo</button>`)
	}
	for _, e := range engines {
		active := ""
		if e.Default {
			active = " active"
		}
		fmt.Fprintf(&sb, `<button class="engine-btn%s" data-url="%s">%s</button>`,
			active, htmlEscape(e.URL), htmlEscape(e.Name))
	}
	return template.HTML(`<div class="search-bar">
    <input type="text" id="search-input" placeholder="search..." autocomplete="off" />
</div>
<div class="search-engines">
    ` + sb.String() + `
</div>`)
}

func (w *SearchWidget) RenderHeader(ctx RenderContext) (template.HTML, error) {
	return searchInner(ctx.SearchEngines), nil
}

func (w *SearchWidget) Render(ctx RenderContext) (template.HTML, error) {
	return wrap(ctx, "search", "検索", "", `<div class="widget-body">`+string(searchInner(ctx.SearchEngines))+`</div>`), nil
}
