package widgets

import (
	"encoding/json"
	"html/template"
	"strconv"

	"muhomu/internal/media"
)

type BookmarksWidget struct{}

func (w *BookmarksWidget) ID() string { return "bookmarks" }

type bookmarksRootView struct {
	Rows    int
	Folders []BookmarkFolder
}

type bookmarkLinkView struct {
	Index   int
	Label   string
	URL     string
	Favicon string
	Letter  string
}

type bookmarksFolderView struct {
	Rows      int
	FolderIdx int
	Links     []bookmarkLinkView
}

func (w *BookmarksWidget) Render(ctx RenderContext) (template.HTML, error) {
	folders, err := ctx.Store.Bookmarks()
	if err != nil {
		return "", err
	}
	state, _ := json.Marshal(folders)

	// Fixed widget height, declared in rows of tiles (config option
	// "rows", default 3). The server emits --bm-rows on the body so the
	// CSS derives the exact pixel height; both views (root listing and
	// inside a folder) share that one fixed box and scroll internally
	// when content overflows.
	rows := 3
	if v, ok := ctx.Options["rows"]; ok {
		switch n := v.(type) {
		case float64:
			rows = int(n)
		case int:
			rows = n
		}
	}
	if rows < 1 {
		rows = 1
	}
	if rows > 10 {
		rows = 10
	}

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
		inner, err := renderBookmarksFolder(folders[folderIdx], folderIdx, rows)
		if err != nil {
			return "", err
		}
		return wrap(ctx, "bookmarks", label, acts,
			string(inner), string(state)), nil
	}

	inner, err := renderBookmarksRoot(folders, rows)
	if err != nil {
		return "", err
	}
	return wrap(ctx, "bookmarks", "ブックマーク", acts,
		string(inner), string(state)), nil
}

// renderBookmarksRoot renders the root listing: a grid of folder tiles.
// Each tile is clickable (hx-get) to enter the folder; the edit pencil
// and drag-reorder live here, mirroring a file manager's parent view.
func renderBookmarksRoot(folders []BookmarkFolder, rows int) (template.HTML, error) {
	return renderWidgetTemplate("bookmarks-root.tmpl", bookmarksRootView{
		Rows:    rows,
		Folders: folders,
	})
}

// renderBookmarksFolder renders the "inside folder" view: the folder's
// links as fav tiles, with a ".." parent tile as the first grid item —
// same size as a fav tile, so going back costs no extra vertical space
// (and the grid can be locked to the root's height exactly).
func renderBookmarksFolder(folder BookmarkFolder, folderIdx, rows int) (template.HTML, error) {
	links := make([]bookmarkLinkView, len(folder.Links))
	for li, link := range folder.Links {
		v := bookmarkLinkView{
			Index: li,
			Label: link.Label,
			URL:   link.URL,
		}
		// Remote favicons (e.g. icons.duckduckgo.com) are skipped by the
		// renderer: the tile shows a letter tile instead, so painting a
		// folder never blocks on (or 404s against) a third-party host. The
		// server caches reachable ones into /api/images/favicons/ at startup,
		// after which the real icon renders here.
		//
		// The template emits width/height attributes on local favicons, not
		// just CSS: when htmx swaps in this view, the browser starts loading
		// the favicons before the stylesheet is applied to the fresh subtree —
		// without attributes an uncached large favicon would flash at its
		// intrinsic (natural) size, filling the widget. Attribute sizes apply
		// at parse time, so the img is 32px from the very first frame, CSS or
		// no CSS.
		if media.IsExternalFavicon(link.Fav) || link.Fav == "" {
			v.Letter = media.FavLetter(link.Label)
		} else {
			v.Favicon = link.Fav
		}
		links[li] = v
	}

	return renderWidgetTemplate("bookmarks-folder.tmpl", bookmarksFolderView{
		Rows:      rows,
		FolderIdx: folderIdx,
		Links:     links,
	})
}
