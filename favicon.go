package main

import (
	"log"

	"muhomu/internal/media"
)

// migrateFavicons runs in the background on startup, scanning all bookmarks
// and quick access items for external favicon urls and caching them locally.
// This handles existing data without requiring user action — the system
// takes on the labour of migration rather than offloading it to the user.
func migrateFavicons(faviconDir string) {
	log.Println("favicon: starting background migration")

	folders, err := dataStore.Bookmarks()
	if err != nil {
		log.Printf("favicon: migration failed to load bookmarks: %v", err)
		return
	}

	changed := false
	for fi, folder := range folders {
		for li, link := range folder.Links {
			// CacheFaviconForLink keeps valid local favicons as-is and
			// re-fetches missing or corrupt ones — so this pass also
			// self-heals broken cached files without user action.
			local := media.CacheFaviconForLink(link.Label, link.URL, link.Fav, faviconDir)
			if local != link.Fav {
				folders[fi].Links[li].Fav = local
				changed = true
				log.Printf("favicon: cached %s → %s", link.URL, local)
			}
		}
	}
	if changed {
		if err := dataStore.SetBookmarks(folders); err != nil {
			log.Printf("favicon: migration failed to save bookmarks: %v", err)
		} else {
			invalidateCache()
			log.Println("favicon: bookmark migration complete")
		}
	} else {
		log.Println("favicon: bookmarks already up to date")
	}

	// quick access
	items, err := dataStore.QuickAccess()
	if err != nil {
		log.Printf("favicon: migration failed to load quick access: %v", err)
		return
	}
	qaChanged := false
	for i, item := range items {
		local := media.CacheFaviconForLink(item.Label, item.URL, item.Favicon, faviconDir)
		if local != item.Favicon {
			items[i].Favicon = local
			qaChanged = true
			log.Printf("favicon: cached qa %s → %s", item.URL, local)
		}
	}
	if qaChanged {
		if err := dataStore.SetQuickAccess(items); err != nil {
			log.Printf("favicon: migration failed to save quick access: %v", err)
		} else {
			invalidateCache()
			log.Println("favicon: quick access migration complete")
		}
	} else {
		log.Println("favicon: quick access already up to date")
	}
}
