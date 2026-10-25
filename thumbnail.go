package main

import (
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
)

// Thumbnails for the image widget. Originals stay untouched where the
// user put them; the widget serves a pre-scaled JPEG from the thumbs
// subdir instead, so a new tab doesn't drag multi-megabyte files back
// and forth just to show a ~250px preview.

const (
	thumbMaxDim  = 700 // longest thumbnail side in pixels
	thumbQuality = 85  // JPEG quality
)

// ensureWidgetThumbs regenerates thumbnails for every widget image whose
// thumbnail is missing or older than the original. Decodes (and skips)
// unsupported formats gracefully — the widget falls back to the
// original file for anything that has no thumbnail.
func ensureWidgetThumbs(dir string) {
	thumbsDir := filepath.Join(dir, "thumbs")
	if err := os.MkdirAll(thumbsDir, 0755); err != nil {
		log.Printf("thumbnail: cannot create %s: %v", thumbsDir, err)
		return
	}
	for _, name := range listImagesDir(dir) {
		ensureWidgetThumb(dir, thumbsDir, name)
	}
}

func ensureWidgetThumb(dir, thumbsDir, name string) {
	origPath := filepath.Join(dir, name)
	thumbPath := filepath.Join(thumbsDir, name+".thumb.jpg")

	origInfo, err := os.Stat(origPath)
	if err != nil {
		return
	}
	if info, err := os.Stat(thumbPath); err == nil && info.Size() > 0 &&
		info.ModTime().After(origInfo.ModTime()) {
		return // thumbnail already up to date
	}

	f, err := os.Open(origPath)
	if err != nil {
		return
	}
	src, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		log.Printf("thumbnail: skipping %s: %v", name, err)
		return
	}

	dst := scaleImage(src, thumbMaxDim)

	out, err := os.Create(thumbPath)
	if err != nil {
		log.Printf("thumbnail: write %s failed: %v", thumbPath, err)
		return
	}
	defer out.Close()
	if err := jpeg.Encode(out, flatten(dst), &jpeg.Options{Quality: thumbQuality}); err != nil {
		log.Printf("thumbnail: encode %s failed: %v", thumbPath, err)
		os.Remove(thumbPath)
	}
}

// scaleImage shrinks src to fit within thumbMaxDim on its longest side,
// preserving aspect ratio, using bilinear interpolation.
func scaleImage(src image.Image, maxDim int) *image.RGBA {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= 0 || sh <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}

	w, h := sw, sh
	if sw > maxDim || sh > maxDim {
		scale := float64(maxDim) / float64(max(sw, sh))
		w = max(1, int(float64(sw)*scale))
		h = max(1, int(float64(sh)*scale))
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	if sw == w && sh == h {
		draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
		return dst
	}

	xs := float64(sw) / float64(w)
	ys := float64(sh) / float64(h)
	for y := 0; y < h; y++ {
		fy := float64(y)*ys + 0.5
		y0 := int(fy)
		y1 := y0 + 1
		if y1 >= sh {
			y1 = y0
		}
		ty := fy - float64(y0)
		for x := 0; x < w; x++ {
			fx := float64(x)*xs + 0.5
			x0 := int(fx)
			x1 := x0 + 1
			if x1 >= sw {
				x1 = x0
			}
			tx := fx - float64(x0)

			c00 := toNRGBA(src.At(b.Min.X+x0, b.Min.Y+y0))
			c01 := toNRGBA(src.At(b.Min.X+x0, b.Min.Y+y1))
			c10 := toNRGBA(src.At(b.Min.X+x1, b.Min.Y+y0))
			c11 := toNRGBA(src.At(b.Min.X+x1, b.Min.Y+y1))
			dst.Set(x, y, color.NRGBA{
				R: lerp(lerp(c00.R, c10.R, tx), lerp(c01.R, c11.R, tx), ty),
				G: lerp(lerp(c00.G, c10.G, tx), lerp(c01.G, c11.G, tx), ty),
				B: lerp(lerp(c00.B, c10.B, tx), lerp(c01.B, c11.B, tx), ty),
				A: lerp(lerp(c00.A, c10.A, tx), lerp(c01.A, c11.A, tx), ty),
			})
		}
	}
	return dst
}

// flatten composites an RGBA image onto the widget panel background, so
// JPEG encoding of a semi-transparent source has no transparent holes.
func flatten(src *image.RGBA) *image.RGBA {
	out := image.NewRGBA(src.Bounds())
	draw.Draw(out, out.Bounds(),
		image.NewUniform(color.NRGBA{R: 0x16, G: 0x15, B: 0x1f, A: 0xff}),
		image.Point{}, draw.Src)
	draw.Draw(out, out.Bounds(), src, src.Bounds().Min, draw.Over)
	return out
}

func toNRGBA(c color.Color) color.NRGBA {
	r, g, b, a := c.RGBA()
	return color.NRGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}

func lerp(a, b uint8, t float64) uint8 {
	return uint8(float64(a) + (float64(b)-float64(a))*t)
}
