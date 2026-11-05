package widgets

import "muhomu/internal/store"

// Widgets use the shared store types directly so the JSON shape stays
// identical between server rendering, data-state, and POST persistence.
type BookmarkLink = store.BookmarkLink
type BookmarkFolder = store.BookmarkFolder
type QuickItem = store.QuickItem
type RecentItem = store.RecentItem
type Quote = store.Quote
