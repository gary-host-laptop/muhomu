package main

import (
	"log"
	"os"
	"path/filepath"

	"muhomu/widgets"
)

// faviconDir is set from appCfg at startup.
// Favicons are stored as <md5-of-source-url>.ext
// and served at /api/images/favicons/<filename>.

// cacheFaviconForLink resolves the favicon url for a bookmark link:
//  1. if link.Fav is already local and valid → keep it
//  2. if link.Fav is local but missing/corrupt → re-fetch from the
//     page's derived url and delete the stale file on success
//  3. if link.Fav is an explicit external url → fetch & cache that
//  4. if link.Fav is empty → derive from link.URL via duckduckgo → fetch & cache
//
// Returns the local path, or the original Fav if caching fails.
func cacheFaviconForLink(label, pageURL, favURL, faviconDir string) string {
	stalePath := ""
	if widgets.IsLocalFavicon(favURL) {
		// verify still on disk AND that it's actually an image — corrupt
		// files (e.g. an HTML page saved as .png) are re-fetched below.
		// Valid images with a wrong extension (e.g. WebP bytes saved as
		// .ico) pass ValidFaviconFile and are deliberately left alone:
		// browsers content-sniff <img> payloads so they render fine, and
		// re-saving them would churn files+DB for zero visible gain.
		p := filepath.Join(faviconDir, filepath.Base(favURL))
		if _, err := os.Stat(p); err == nil && widgets.ValidFaviconFile(p) {
			return favURL
		}
		stalePath = p
	}

	src := favURL
	if src == "" || widgets.IsLocalFavicon(src) {
		src = widgets.GuessFaviconURL(pageURL)
	}
	if src == "" {
		return favURL
	}

	local := widgets.CacheFavicon(src, faviconDir)
	if local == "" {
		// fetch failed — return original so browser can still try
		return favURL
	}
	// Remove the stale corrupt file only if it is a different file than the
	// one CacheFavicon just wrote. When the re-fetch source resolves to the
	// same md5 base (e.g. a corrupt icon originally cached from the same
	// duckduckgo url), CacheFavicon overwrites that exact path in place —
	// deleting it here would destroy the fresh icon.
	if stalePath != "" && filepath.Base(stalePath) != filepath.Base(local) {
		if err := os.Remove(stalePath); err == nil {
			log.Printf("favicon: removed stale corrupt file %s", filepath.Base(stalePath))
		}
	}
	return local
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
			// cacheFaviconForLink keeps valid local favicons as-is and
			// re-fetches missing or corrupt ones — so this pass also
			// self-heals broken cached files without user action.
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
