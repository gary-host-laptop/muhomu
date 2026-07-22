package widgets

import (
	"fmt"
	"html/template"
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

func firstWidgetImage(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && allowedImgExts[strings.ToLower(filepath.Ext(e.Name()))] {
			files = append(files, e.Name())
		}
	}
	if len(files) == 0 {
		return ""
	}
	sort.Strings(files)
	return "/api/widget-images/files/" + files[0]
}

func (w *ImageWidget) ID() string { return "image" }

func (w *ImageWidget) Render(ctx RenderContext) (template.HTML, error) {
	imageURL := firstWidgetImage(ctx.WidgetImageDir)
	imgHTML := ""
	if imageURL != "" {
		filename := imageURL[strings.LastIndex(imageURL, "/")+1:]
		imgHTML = fmt.Sprintf(
			`<img src="%s" alt="" id="widget-img" class="active" data-filename="%s">`,
			imageURL, filename,
		)
	}
	inner := fmt.Sprintf(`<div class="widget-img-inner">
  <div class="widget-img-wrap" id="widget-img-wrap">%s</div>
</div>`, imgHTML)
	return wrap(ctx, "image", "イメージ",
		`<button class="wt-act" id="widget-img-next" title="next image"><i class="ph-light ph-caret-right"></i></button>`,
		inner), nil
}


