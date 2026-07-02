package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// faviconDir is set from appCfg at startup.
// Favicons are stored as <md5-of-source-url>.ext
// and served at /api/images/favicons/<filename>.

// guessFaviconURL derives the duckduckgo favicon url from a page url,
// mirroring the js guessFavicon function.
func guessFaviconURL(pageURL string) string {
	u, err := url.Parse(pageURL)
	if err != nil || u.Host == "" {
		return ""
	}
	return "https://icons.duckduckgo.com/ip3/" + u.Hostname() + ".ico"
}

// faviconKey returns a stable filename for a given source url.
func faviconKey(srcURL string) string {
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

// isLocalFavicon returns true if the url is already a local path.
func isLocalFavicon(u string) bool {
	return strings.HasPrefix(u, "/api/images/favicons/")
}

// cacheFavicon fetches srcURL, saves it to the favicons directory,
// and returns the local serving path. Returns "" on failure.
func cacheFavicon(srcURL, faviconDir string) string {
	if srcURL == "" {
		return ""
	}
	if isLocalFavicon(srcURL) {
		// already cached — verify file still exists
		filename := filepath.Base(srcURL)
		if _, err := os.Stat(filepath.Join(faviconDir, filename)); err == nil {
			return srcURL
		}
		// file missing, re-fetch
	}

	key := faviconKey(srcURL)
	destPath := filepath.Join(faviconDir, key)

	// already on disk
	if _, err := os.Stat(destPath); err == nil {
		return "/api/images/favicons/" + key
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(srcURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ""
	}

	f, err := os.Create(destPath)
	if err != nil {
		return ""
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(destPath)
		return ""
	}

	return "/api/images/favicons/" + key
}

// cacheFaviconForLink resolves the favicon url for a bookmark link:
//  1. if link.Fav is already local → keep it
//  2. if link.Fav is an explicit external url → fetch & cache that
//  3. if link.Fav is empty → derive from link.URL via duckduckgo → fetch & cache
//
// Returns the local path, or the original Fav if caching fails.
func cacheFaviconForLink(label, pageURL, favURL, faviconDir string) string {
	if isLocalFavicon(favURL) {
		// verify still on disk
		filename := filepath.Base(favURL)
		if _, err := os.Stat(filepath.Join(faviconDir, filename)); err == nil {
			return favURL
		}
	}

	src := favURL
	if src == "" || isLocalFavicon(src) {
		src = guessFaviconURL(pageURL)
	}
	if src == "" {
		return favURL
	}

	local := cacheFavicon(src, faviconDir)
	if local != "" {
		return local
	}
	// fetch failed — return original so browser can still try
	return favURL
}

// migrateFavicons runs in the background on startup, scanning all bookmarks
// and quick access items for external favicon urls and caching them locally.
// This handles existing data without requiring user action — the system
// takes on the labour of migration rather than offloading it to the user.
func migrateFavicons(faviconDir string) {
	log.Println("favicon: starting background migration")

	folders, err := getBookmarks()
	if err != nil {
		log.Printf("favicon: migration failed to load bookmarks: %v", err)
		return
	}

	changed := false
	for fi, folder := range folders {
		for li, link := range folder.Links {
			if isLocalFavicon(link.Fav) {
				continue
			}
			local := cacheFaviconForLink(link.Label, link.URL, link.Fav, faviconDir)
			if local != link.Fav {
				folders[fi].Links[li].Fav = local
				changed = true
				log.Printf("favicon: cached %s → %s", link.URL, local)
			}
		}
	}
	if changed {
		if err := setBookmarks(folders); err != nil {
			log.Printf("favicon: migration failed to save bookmarks: %v", err)
		} else {
			invalidateCache()
			log.Println("favicon: bookmark migration complete")
		}
	} else {
		log.Println("favicon: bookmarks already up to date")
	}

	// quick access
	items, err := getQuickAccess()
	if err != nil {
		log.Printf("favicon: migration failed to load quick access: %v", err)
		return
	}
	qaChanged := false
	for i, item := range items {
		if isLocalFavicon(item.Favicon) {
			continue
		}
		local := cacheFaviconForLink(item.Label, item.URL, item.Favicon, faviconDir)
		if local != item.Favicon {
			items[i].Favicon = local
			qaChanged = true
			log.Printf("favicon: cached qa %s → %s", item.URL, local)
		}
	}
	if qaChanged {
		if err := setQuickAccess(items); err != nil {
			log.Printf("favicon: migration failed to save quick access: %v", err)
		} else {
			invalidateCache()
			log.Println("favicon: quick access migration complete")
		}
	} else {
		log.Println("favicon: quick access already up to date")
	}
}
