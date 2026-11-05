package widgets

import (
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ImageWidget struct{}

type imageView struct {
	ImageURL string
	Filename string
	NextURL  string
}

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

	view := imageView{ImageURL: imageURL, Filename: filename}
	if imageURL != "" {
		// The next button carries the current filename so the server can
		// step to the following image. A prefetch link warms the cache for
		// the image after that, keeping clicks instant without client JS.
		if next := widgetImageAfter(ctx.WidgetImageDir, filename); next != "" {
			view.NextURL = widgetServeURL(ctx.WidgetImageDir, next)
		}
	}

	inner, err := renderWidgetTemplate("image.tmpl", view)
	if err != nil {
		return "", err
	}
	acts, err := renderWidgetTemplate("image-actions.tmpl", view)
	if err != nil {
		return "", err
	}
	return wrap(ctx, "image", "イメージ", string(acts), string(inner)), nil
}
