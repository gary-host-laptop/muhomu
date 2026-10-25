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

// widgetThumbPath returns the thumbnail path on disk for a widget image.
func widgetThumbPath(dir, filename string) string {
	return filepath.Join(dir, "thumbs", filename+".thumb.jpg")
}

// widgetServeURL returns the URL to display for a widget image file: the
// pre-generated thumbnail when it exists, the original file otherwise.
func widgetServeURL(dir, filename string) string {
	if _, err := os.Stat(widgetThumbPath(dir, filename)); err == nil {
		return "/api/widget-images/thumbs/" + url.PathEscape(filename+".thumb.jpg")
	}
	return "/api/widget-images/files/" + url.PathEscape(filename)
}

// widgetImageAfter returns the filename that follows current
// alphabetically, wrapping around at the end. With an empty or unknown
// current it returns the first image.
func widgetImageAfter(dir, current string) string {
	files := listWidgetImages(dir)
	if len(files) == 0 {
		return ""
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
	return files[idx]
}

func (w *ImageWidget) ID() string { return "image" }

func (w *ImageWidget) Render(ctx RenderContext) (template.HTML, error) {
	files := listWidgetImages(ctx.WidgetImageDir)

	var imageURL, filename string
	if len(files) > 0 {
		current := ctx.Query["current"]
		idx := 0
		if current != "" {
			// Detached "next" clicks step alphabetically from ?current=.
			for i, f := range files {
				if f == current {
					idx = (i + 1) % len(files)
					break
				}
			}
		} else if ctx.RNG != nil {
			// Full-page renders start from a random image, so new tabs
			// are never pinned to one heavy file (and once every image has
			// been cached, every new tab is served from cache).
			idx = ctx.RNG.Intn(len(files))
		}
		filename = files[idx]
		imageURL = widgetServeURL(ctx.WidgetImageDir, filename)
	}

	inner := `<div class="widget-img-inner">
  <div class="widget-img-wrap" id="widget-img-wrap"></div>
</div>`
	acts := `<button class="wt-act" id="widget-img-next" title="next image" disabled><i class="ph-light ph-caret-right"></i></button>`

	if imageURL != "" {
		// width/height stay CSS-driven; lazy+async keep a big first image
		// from blocking first paint of the tab.
		inner = fmt.Sprintf(`<div class="widget-img-inner">
  <div class="widget-img-wrap" id="widget-img-wrap"><img src="%s" alt="" id="widget-img" class="active" data-filename="%s" loading="lazy" decoding="async"></div>
</div>`, htmlEscape(imageURL), filename)
		// The next button carries the current filename so the server can
		// step to the following image. A prefetch link warms the cache for
		// the image after that, keeping clicks instant without client JS.
		acts = fmt.Sprintf(
			`<button class="wt-act" id="widget-img-next" title="next image" hx-get="/api/widget/image?current=%s" hx-target="closest .widget" hx-swap="outerHTML"><i class="ph-light ph-caret-right"></i></button>`,
			url.QueryEscape(filename),
		)
		if next := widgetImageAfter(ctx.WidgetImageDir, filename); next != "" {
			inner += fmt.Sprintf(`<link rel="prefetch" href="%s">`, htmlEscape(widgetServeURL(ctx.WidgetImageDir, next)))
		}
	}

	return wrap(ctx, "image", "イメージ", acts, inner), nil
}
