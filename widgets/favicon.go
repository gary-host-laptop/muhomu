// Package widgets — shared favicon caching.
//
// Favicons are stored as <md5-of-source-url><detected-ext>
// and served at /api/images/favicons/<filename>.
package widgets

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// maxFaviconBytes caps how much of a favicon response we are willing to
// store. Real favicons are tens of kilobytes; anything bigger is a
// misbehaving server, not an icon.
const maxFaviconBytes = 2 << 20 // 2 MiB

// FaviconKey returns a stable filename for a given source url,
// using the url's own extension as a hint. New files are actually
// written with the extension detected from the image content (see
// CacheFavicon), so this only matters for backward compatibility.
func FaviconKey(srcURL string) string {
	sum := md5.Sum([]byte(srcURL))
	ext := filepath.Ext(srcURL)
	if ext == "" || len(ext) > 5 {
		ext = ".ico"
	}
	// strip query strings from ext
	if idx := strings.Index(ext, "?"); idx != -1 {
		ext = ext[:idx]
	}
	return fmt.Sprintf("%x%s", sum, ext)
}

// faviconBase returns the stable md5 prefix shared by all files cached
// from one source url (regardless of extension).
func faviconBase(srcURL string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(srcURL)))
}

// IsLocalFavicon returns true if the url is already a local path.
func IsLocalFavicon(u string) bool {
	return strings.HasPrefix(u, "/api/images/favicons/")
}

// GuessFaviconURL derives the duckduckgo favicon url from a page url.
func GuessFaviconURL(pageURL string) string {
	u, err := url.Parse(pageURL)
	if err != nil || u.Host == "" {
		return ""
	}
	return "https://icons.duckduckgo.com/ip3/" + u.Hostname() + ".ico"
}

// sniffImageFormat identifies the image format of data from its magic
// bytes and returns the matching file extension. Returns ("", false) for
// anything that is not a recognized image — including HTML pages, JSON
// error bodies, or truncated files.
func sniffImageFormat(b []byte) (string, bool) {
	switch {
	case len(b) >= 8 && bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		return ".png", true
	case len(b) >= 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF:
		return ".jpg", true
	case len(b) >= 6 && (bytes.HasPrefix(b, []byte("GIF87a")) || bytes.HasPrefix(b, []byte("GIF89a"))):
		return ".gif", true
	case len(b) >= 12 && bytes.HasPrefix(b, []byte("RIFF")) && string(b[8:12]) == "WEBP":
		return ".webp", true
	case len(b) >= 4 && b[0] == 0 && b[1] == 0 && b[2] == 1 && b[3] == 0:
		return ".ico", true
	case len(b) >= 12 && bytes.HasPrefix(b[4:8], []byte("ftyp")) &&
		(bytes.HasPrefix(b[8:12], []byte("avif")) || bytes.HasPrefix(b[8:12], []byte("avis"))):
		return ".avif", true
	}
	// SVG is text: allow an XML/DOCTYPE document (with an optional UTF-8
	// BOM) that actually declares an <svg> root inside the first 300 bytes.
	s := b
	if bytes.HasPrefix(s, []byte{0xEF, 0xBB, 0xBF}) {
		s = s[3:]
	}
	if len(s) > 0 && s[0] == '<' {
		head := s
		if len(head) > 300 {
			head = head[:300]
		}
		if strings.Contains(strings.ToLower(string(head)), "<svg") {
			return ".svg", true
		}
	}
	return "", false
}

// ValidFaviconFile reports whether the file at path is a recognized
// image format (checked by magic bytes). Corrupt files — e.g. an HTML
// page that was saved as a favicon — fail this check.
func ValidFaviconFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	head := make([]byte, 512)
	n, err := io.ReadFull(f, head)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return false
	}
	_, ok := sniffImageFormat(head[:n])
	return ok
}

// CacheFavicon fetches srcURL, validates that the response really is an
// image, saves it to the favicons directory under the extension detected
// from its content, and returns the local serving path. Returns "" on
// failure — and, crucially, never writes non-image bytes to disk.
func CacheFavicon(srcURL, faviconDir string) string {
	if srcURL == "" {
		return ""
	}
	if IsLocalFavicon(srcURL) {
		// already cached — verify the file is present and valid
		filename := filepath.Base(srcURL)
		p := filepath.Join(faviconDir, filename)
		if _, err := os.Stat(p); err == nil && ValidFaviconFile(p) {
			return srcURL
		}
		// missing or corrupt — we have no source to re-fetch from here
		return ""
	}

	base := faviconBase(srcURL)

	// cache hit: any valid file for this source (whatever extension a
	// previous fetch stored it under) is good.
	if matches, err := filepath.Glob(filepath.Join(faviconDir, base+".*")); err == nil {
		for _, m := range matches {
			if ValidFaviconFile(m) {
				return "/api/images/favicons/" + filepath.Base(m)
			}
		}
	}

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(srcURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ""
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxFaviconBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxFaviconBytes {
		return ""
	}

	ext, ok := sniffImageFormat(data)
	if !ok {
		// Not an image (HTML page, error body, ...) — refuse to store it.
		return ""
	}

	destPath := filepath.Join(faviconDir, base+ext)
	if err := os.WriteFile(destPath, data, 0o644); err != nil {
		return ""
	}

	// Drop stale files cached for the same source under a different
	// (wrong) extension — e.g. an HTML page previously saved as .png.
	if matches, err := filepath.Glob(filepath.Join(faviconDir, base+".*")); err == nil {
		for _, m := range matches {
			if m != destPath {
				os.Remove(m)
			}
		}
	}

	return "/api/images/favicons/" + filepath.Base(destPath)
}
