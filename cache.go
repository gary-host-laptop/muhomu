package main

import (
	"sync"
	"time"
)

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

var (
	pageCache = struct {
		sync.RWMutex
		data map[string]*cacheEntry
	}{data: make(map[string]*cacheEntry)}
	cacheTTL = 5 * time.Second
)

func invalidateCache() {
	pageCache.Lock()
	pageCache.data = make(map[string]*cacheEntry)
	pageCache.Unlock()
}
