// Package media owns filesystem-backed assets: user images, thumbnails,
// and the favicon cache. Widgets and HTTP handlers depend on these helpers
// instead of knowing about directory layouts or image formats.
package media

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

var allowedImageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true,
	".webp": true, ".gif": true, ".avif": true,
}

// IsAllowedImage reports whether name has an extension the upload and
// picker paths accept.
func IsAllowedImage(name string) bool {
	return allowedImageExts[strings.ToLower(filepath.Ext(name))]
}

// ListImages returns the image files in dir. Missing directories yield nil.
func ListImages(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if IsAllowedImage(e.Name()) {
			files = append(files, e.Name())
		}
	}
	return files
}

// PickImage chooses a random image from dir and prefixes it with the URL
// path it is served under. An empty dir yields "".
func PickImage(dir, urlPrefix string, rng *rand.Rand) string {
	files := ListImages(dir)
	if len(files) == 0 {
		return ""
	}
	return urlPrefix + files[rng.Intn(len(files))]
}
