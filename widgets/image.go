package widgets

import (
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ImageWidget struct{}

var allowedImgExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true,
	".webp": true, ".gif": true, ".avif": true,
}

func listWidgetImages(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && allowedImgExts[strings.ToLower(filepath.Ext(e.Name()))] {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files
}

// nextWidgetImage returns the serving URL and filename of the image that
// follows current alphabetically, wrapping around at the end. With an
// empty or unknown current it returns the first image.
func nextWidgetImage(dir, current string) (string, string) {
	files := listWidgetImages(dir)
	if len(files) == 0 {
		return "", ""
	}
	idx := 0
	if current != "" {
		for i, f := range files {
			if f == current {
				idx = (i + 1) % len(files)
				break
			}
		}
	}
	return "/api/widget-images/files/" + files[idx], files[idx]
}

func (w *ImageWidget) ID() string { return "image" }

func (w *ImageWidget) Render(ctx RenderContext) (template.HTML, error) {
	imageURL, filename := nextWidgetImage(ctx.WidgetImageDir, ctx.Query["current"])

	inner := `<div class="widget-img-inner">
  <div class="widget-img-wrap" id="widget-img-wrap"></div>
</div>`
	acts := `<button class="wt-act" id="widget-img-next" title="next image" disabled><i class="ph-light ph-caret-right"></i></button>`

	if imageURL != "" {
		inner = fmt.Sprintf(`<div class="widget-img-inner">
  <div class="widget-img-wrap" id="widget-img-wrap"><img src="%s" alt="" id="widget-img" class="active" data-filename="%s"></div>
</div>`, htmlEscape(imageURL), filename)
		// The next button carries the current filename so the server can
		// step to the following image. A prefetch link warms the cache for
		// the image after that, keeping clicks instant without client JS.
		acts = fmt.Sprintf(
			`<button class="wt-act" id="widget-img-next" title="next image" hx-get="/api/widget/image?current=%s" hx-target="closest .widget" hx-swap="outerHTML"><i class="ph-light ph-caret-right"></i></button>`,
			url.QueryEscape(filename),
		)
		if nextURL, _ := nextWidgetImage(ctx.WidgetImageDir, filename); nextURL != "" {
			inner += fmt.Sprintf(`<link rel="prefetch" href="%s">`, htmlEscape(nextURL))
		}
	}

	return wrap(ctx, "image", "イメージ", acts, inner), nil
}
