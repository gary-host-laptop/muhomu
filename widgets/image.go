package widgets

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

type ImageWidget struct{}

func (w *ImageWidget) ID() string { return "image" }

func (w *ImageWidget) Render(ctx RenderContext) (template.HTML, error) {
	imageURL := pickWidgetImageDir(ctx.WidgetImageDir, ctx.RNG)
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
		`<button class="wt-act" id="widget-img-next" title="random image"><i class="ph-light ph-shuffle"></i></button>`,
		inner), nil
}

var allowedExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true,
	".webp": true, ".gif": true, ".avif": true,
}

func pickWidgetImageDir(dir string, rng interface{ Intn(int) int }) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && allowedExts[strings.ToLower(filepath.Ext(e.Name()))] {
			files = append(files, e.Name())
		}
	}
	if len(files) == 0 {
		return ""
	}
	return "/api/widget-images/files/" + files[rng.Intn(len(files))]
}

func (w *ImageWidget) Script() string {
	return `(function(){
  const btn=document.getElementById("widget-img-next");
  const wrap=document.getElementById("widget-img-wrap");
  if(!btn||!wrap)return;
  let currentFilename=wrap.querySelector("img")?.dataset.filename||"";
  let preloaded=null;
  function setImage(url,filename){
    const img=document.createElement("img");
    img.src=url;img.id="widget-img";img.className="active";img.dataset.filename=filename;
    wrap.innerHTML="";wrap.appendChild(img);currentFilename=filename;
  }
  function preloadNext(after){
    fetch("/api/widget-images/next?current="+encodeURIComponent(after))
      .then(r=>r.json()).then(d=>{if(!d.url)return;const i=new Image();i.src=d.url;preloaded={url:d.url,filename:d.filename};})
      .catch(()=>{});
  }
  btn.addEventListener("click",()=>{
    if(!preloaded){
      fetch("/api/widget-images/next?current="+encodeURIComponent(currentFilename))
        .then(r=>r.json()).then(d=>{if(!d.url)return;setImage(d.url,d.filename);preloadNext(d.filename);})
        .catch(()=>{});
      return;
    }
    setImage(preloaded.url,preloaded.filename);preloaded=null;preloadNext(currentFilename);
  });
  if(currentFilename)preloadNext(currentFilename);
})();`
}
