package widgets

import (
	"fmt"
	"html/template"
	"strings"
)

type BookmarksWidget struct{}

func (w *BookmarksWidget) ID() string { return "bookmarks" }

func (w *BookmarksWidget) Render(ctx RenderContext) (template.HTML, error) {
	folders, err := dbBookmarks(ctx.DB)
	if err != nil {
		return "", err
	}
	return wrap("bookmarks", "red", "ブックマーク",
		`<button class="wt-btn" id="bm-add-folder-btn"><i class="ph-light ph-plus"></i></button>`,
		string(renderBookmarks(folders))), nil
}

func renderBookmarks(folders []BookmarkFolder) template.HTML {
	var sb strings.Builder
	sb.WriteString(`<div class="widget-body"><div class="folder-list" id="folder-list">`)
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
  <div class="folder-links grid">`, fi, htmlEscape(folder.Folder))
		for li, link := range folder.Links {
			fav := link.Fav
			if fav == "" {
				fav = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='32' height='32'%3E%3Crect width='32' height='32' fill='%23333'/%3E%3C/svg%3E"
			}
			fmt.Fprintf(&sb,
				`<a href="%s" target="_blank" class="fav-tile" draggable="true" data-drag-index="%d">
  <img class="fav" src="%s" alt="">
  <span>%s</span>
  <button class="tile-edit"><i class="ph-light ph-pencil-simple"></i></button>
</a>`, link.URL, li, fav, htmlEscape(link.Label))
		}
		sb.WriteString(`</div></details>`)
	}
	sb.WriteString(`</div></div>`)
	return template.HTML(sb.String())
}
