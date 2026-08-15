package widgets

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSniffImageFormat(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		ext  string
		ok   bool
	}{
		{"png", []byte("\x89PNG\r\n\x1a\n" + "rest-of-png"), ".png", true},
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00}, ".jpg", true},
		{"gif87a", []byte("GIF87a........"), ".gif", true},
		{"gif89a", []byte("GIF89a........"), ".gif", true},
		{"webp", []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), ".webp", true},
		{"ico", []byte{0x00, 0x00, 0x01, 0x00, 0x01, 0x00}, ".ico", true},
		{"avif", []byte("\x00\x00\x00\x1cftypavif\x00\x00\x00\x00"), ".avif", true},
		{"svg-decl", []byte("<?xml version=\"1.0\"?><svg xmlns=\"http://www.w3.org/2000/svg\">"), ".svg", true},
		{"svg-bom", []byte("\xef\xbb\xbf<svg>"), ".svg", true},
		{"html-page", []byte("<!DOCTYPE html><html><head><title>x</title></head><body>hi</body></html>"), "", false},
		{"github-html", []byte("<!DOCTYPE html>\n<html lang=\"en\" data-color-mode=\"auto\">"), "", false},
		{"json-error", []byte(`{"error":"boom"}`), "", false},
		{"empty", nil, "", false},
		{"short", []byte{0x00, 0x00}, "", false},
	}
	for _, c := range cases {
		ext, ok := sniffImageFormat(c.data)
		if ok != c.ok || ext != c.ext {
			t.Errorf("%s: got (%q, %v), want (%q, %v)", c.name, ext, ok, c.ext, c.ok)
		}
	}
}

func TestCacheFaviconRejectsNonImage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		w.Write([]byte("<!DOCTYPE html><html><body>not an icon</body></html>"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	got := CacheFavicon(srv.URL+"/favicon.png", dir)
	if got != "" {
		t.Fatalf("expected failure for non-image response, got %q", got)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("non-image response must not be written to disk, found %d files", len(entries))
	}
}

func TestCacheFaviconSavesWithDetectedExt(t *testing.T) {
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(png)
	}))
	defer srv.Close()

	dir := t.TempDir()
	// the source url claims .ico — the stored file must still be .png
	src := srv.URL + "/icon.ico"
	base := fmt.Sprintf("%x", md5.Sum([]byte(src)))
	// stale corrupt file from a previous (buggy) fetch:
	os.WriteFile(filepath.Join(dir, base+".ico"), []byte("<!DOCTYPE html>"), 0o644)

	got := CacheFavicon(src, dir)
	want := "/api/images/favicons/" + base + ".png"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	data, err := os.ReadFile(filepath.Join(dir, base+".png"))
	if err != nil || !strings.HasPrefix(string(data), "\x89PNG\r\n\x1a\n") {
		t.Fatalf("png file missing or wrong content: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, base+".ico")); err == nil {
		t.Fatal("stale corrupt .ico file was not removed")
	}
}

func TestCacheFaviconReusesValidCache(t *testing.T) {
	dir := t.TempDir()
	src := "https://example.com/favicon.png"
	base := fmt.Sprintf("%x", md5.Sum([]byte(src)))
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)
	os.WriteFile(filepath.Join(dir, base+".png"), png, 0o644)

	got := CacheFavicon(src, dir)
	if want := "/api/images/favicons/" + base + ".png"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestCacheFaviconOverwritesCorruptSamePath(t *testing.T) {
	// A corrupt file cached under the md5 base of the fetch source must be
	// replaced IN PLACE (same path returned), so callers that re-fetch from
	// the same source never end up deleting the fresh icon.
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(png)
	}))
	defer srv.Close()

	dir := t.TempDir()
	src := srv.URL + "/favicon.ico"
	base := fmt.Sprintf("%x", md5.Sum([]byte(src)))
	// previous fetch stored garbage (an html page) at the exact path we
	// will write: base + .png
	badPath := filepath.Join(dir, base+".png")
	os.WriteFile(badPath, []byte("<!DOCTYPE html>"), 0o644)

	got := CacheFavicon(src, dir)
	want := "/api/images/favicons/" + base + ".png"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	data, err := os.ReadFile(badPath)
	if err != nil {
		t.Fatalf("in-place overwrite left no file: %v", err)
	}
	if !strings.HasPrefix(string(data), "\x89PNG\r\n\x1a\n") {
		t.Fatalf("corrupt file was not replaced by valid image")
	}
}

func TestValidFaviconFile(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.ico")
	os.WriteFile(good, []byte{0x00, 0x00, 0x01, 0x00, 0x01, 0x00}, 0o644)
	if !ValidFaviconFile(good) {
		t.Fatal("valid ico file reported invalid")
	}

	bad := filepath.Join(dir, "bad.png")
	os.WriteFile(bad, []byte("<!DOCTYPE html><html>"), 0o644)
	if ValidFaviconFile(bad) {
		t.Fatal("html file reported as a valid image")
	}

	if ValidFaviconFile(filepath.Join(dir, "missing.ico")) {
		t.Fatal("missing file reported valid")
	}
}
