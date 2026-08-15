package widgets

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strconv"
	"strings"
)

type BookmarksWidget struct{}

func (w *BookmarksWidget) ID() string { return "bookmarks" }

func (w *BookmarksWidget) Render(ctx RenderContext) (template.HTML, error) {
	folders, err := dbBookmarks(ctx.DB)
	if err != nil {
		return "", err
	}
	state, _ := json.Marshal(folders)

	// A ?folder=N query (from htmx navigation and CRUD posts) switches the
	// widget into "inside folder N" mode: the body shows that folder's
	// links with a ".." row to return to the root listing. The root
	// listing stays the default render.
	folderIdx := -1
	if f, ok := ctx.Query["folder"]; ok && f != "" {
		if n, err := strconv.Atoi(f); err == nil && n >= 0 && n < len(folders) {
			folderIdx = n
		}
	}

	acts := `<button class="wt-btn" id="bm-add-folder-btn"><i class="ph-light ph-plus"></i></button>`

	if folderIdx >= 0 {
		label := "ブックマーク › " + htmlEscape(folders[folderIdx].Folder)
		return wrap(ctx, "bookmarks", label, acts,
			string(renderBookmarksFolder(folders[folderIdx], folderIdx)), string(state)), nil
	}

	return wrap(ctx, "bookmarks", "ブックマーク", acts,
		string(renderBookmarksRoot(folders)), string(state)), nil
}

// renderBookmarksRoot renders the root listing: a grid of folder tiles.
// Each tile is clickable (hx-get) to enter the folder; the edit pencil
// and drag-reorder live here, mirroring a file manager's parent view.
func renderBookmarksRoot(folders []BookmarkFolder) template.HTML {
	var sb strings.Builder
	sb.WriteString(`<div class="widget-body"><div class="folder-tiles" id="folder-list" data-folder="-1">`)
	for fi, folder := range folders {
		fmt.Fprintf(&sb, `<div class="folder-tile" tabindex="0" role="button" draggable="true" data-drag-index="%d" title="open folder"
  hx-get="/api/widget/bookmarks?folder=%d" hx-target="closest .widget" hx-swap="outerHTML">
  <i class="ph-light ph-folder" aria-hidden="true"></i>
  <span class="folder-name">%s</span>
  <button class="tile-edit" title="edit folder"><i class="ph-light ph-pencil-simple"></i></button>
</div>`, fi, fi, htmlEscape(folder.Folder))
	}
	if len(folders) == 0 {
		sb.WriteString(`<div class="folder-empty">no folders yet</div>`)
	}
	sb.WriteString(`</div></div>`)
	return template.HTML(sb.String())
}

// renderBookmarksFolder renders the "inside folder" view: the folder's
// links as fav tiles, with a ".." parent tile as the first grid item —
// same size as a fav tile, so going back costs no extra vertical space
// (and the grid can be locked to the root's height exactly).
func renderBookmarksFolder(folder BookmarkFolder, folderIdx int) template.HTML {
	var sb strings.Builder
	fmt.Fprintf(&sb, `<div class="widget-body"><div class="folder-links grid" id="folder-list" data-folder="%d">
<div class="folder-up fav-tile" tabindex="0" role="button" title="up one level"
  hx-get="/api/widget/bookmarks" hx-target="closest .widget" hx-swap="outerHTML">
  <i class="ph-light ph-folder" aria-hidden="true"></i>
  <span>..</span>
</div>`, folderIdx)
	if len(folder.Links) == 0 {
		sb.WriteString(`<div class="folder-empty">empty folder</div>`)
	} else {
		for li, link := range folder.Links {
			sb.WriteString(favTile(link, li))
		}
	}
	sb.WriteString(`</div></div>`)
	return template.HTML(sb.String())
}

// favTile renders one bookmark link as a draggable favicon tile.
func favTile(link BookmarkLink, li int) string {
	fav := link.Fav
	if fav == "" {
		fav = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='32' height='32'%3E%3Crect width='32' height='32' fill='%23333'/%3E%3C/svg%3E"
	}
	// width/height attributes, not just CSS: when htmx swaps in this
	// view, the browser starts loading the favicons before the stylesheet
	// is applied to the fresh subtree — without attributes an uncached
	// large favicon would flash at its intrinsic (natural) size, filling
	// the widget. Attribute sizes apply at parse time, so the img is
	// 32px from the very first frame, CSS or no CSS.
	return fmt.Sprintf(
		`<a href="%s" class="fav-tile" title="%s" draggable="true" data-drag-index="%d">
  <img class="fav" src="%s" alt="" width="32" height="32">
  <span>%s</span>
  <button class="tile-edit"><i class="ph-light ph-pencil-simple"></i></button>
</a>`, link.URL, htmlEscape(link.Label), li, fav, htmlEscape(link.Label))
}
