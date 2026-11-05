package widgets

import "html/template"

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

func clockInner() (template.HTML, error) {
	return renderWidgetTemplate("clock.tmpl", nil)
}

func (w *ClockWidget) RenderHeader(ctx RenderContext) (template.HTML, error) {
	return clockInner()
}

func (w *ClockWidget) Render(ctx RenderContext) (template.HTML, error) {
	inner, err := clockInner()
	if err != nil {
		return "", err
	}
	return wrap(ctx, "clock", "時計", "", `<div class="widget-body">`+string(inner)+`</div>`), nil
}

// IdentityWidget renders the user's identity block: the edit/settings
// buttons, the greeting + username, and the barcode deco.
// The client (newtab.js / i18n.js) fills the greeting by id; the
// username comes from config.
type IdentityWidget struct{}

func (w *IdentityWidget) ID() string { return "identity" }

func identityInner(username string) (template.HTML, error) {
	return renderWidgetTemplate("identity.tmpl", username)
}

func (w *IdentityWidget) RenderHeader(ctx RenderContext) (template.HTML, error) {
	return identityInner(ctx.Username)
}

func (w *IdentityWidget) Render(ctx RenderContext) (template.HTML, error) {
	inner, err := identityInner(ctx.Username)
	if err != nil {
		return "", err
	}
	return wrap(ctx, "identity", "アイデンティティ", "", `<div class="widget-body">`+string(inner)+`</div>`), nil
}

// ProfileWidget renders the profile picture (avatar) with its corner
// decorations. The image URL is picked by the coordinator (serveIndex)
// from the profile images directory.
type ProfileWidget struct{}

func (w *ProfileWidget) ID() string { return "profile" }

func profileInner(imgURL string) (template.HTML, error) {
	return renderWidgetTemplate("profile.tmpl", imgURL)
}

func (w *ProfileWidget) RenderHeader(ctx RenderContext) (template.HTML, error) {
	return profileInner(ctx.ProfileImageURL)
}

func (w *ProfileWidget) Render(ctx RenderContext) (template.HTML, error) {
	inner, err := profileInner(ctx.ProfileImageURL)
	if err != nil {
		return "", err
	}
	return wrap(ctx, "profile", "プロフィール", "", `<div class="widget-body">`+string(inner)+`</div>`), nil
}

// SearchWidget renders the search bar and the engine buttons. The
// client (newtab.js) wires the input and the engine selection by
// class/id. It can live in the header strip or in the body (its
// classic full-width position below the header is a page-level strip
// that renders this widget's header content).
type SearchWidget struct{}

func (w *SearchWidget) ID() string { return "search" }

func searchInner(engines []SearchEngine) (template.HTML, error) {
	return renderWidgetTemplate("search.tmpl", engines)
}

func (w *SearchWidget) RenderHeader(ctx RenderContext) (template.HTML, error) {
	return searchInner(ctx.SearchEngines)
}

func (w *SearchWidget) Render(ctx RenderContext) (template.HTML, error) {
	inner, err := searchInner(ctx.SearchEngines)
	if err != nil {
		return "", err
	}
	return wrap(ctx, "search", "検索", "", `<div class="widget-body">`+string(inner)+`</div>`), nil
}
