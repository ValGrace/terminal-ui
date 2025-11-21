package cache

import (
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"sync"
	"time"
)

// Cache provides caching for command history data
type Cache struct {
	dirCache   map[string]*directoryCacheEntry
	mu         sync.RWMutex
	maxEntries int
	ttl        time.Duration
}

// directoryCacheEntry represents a cached directory entry
type directoryCacheEntry struct {
	commands  []history.CommandRecord
	timestamp time.Time
}

// New creates a new cache instance
func New(maxEntries int, ttl time.Duration) *Cache {
	return &Cache{
		dirCache:   make(map[string]*directoryCacheEntry),
		maxEntries: maxEntries,
		ttl:        ttl,
	}
}

// GetCommands retrieves commands for a directory from cache
func (c *Cache) GetCommands(directory string) ([]history.CommandRecord, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.dirCache[directory]
	if !exists {
		return nil, false
	}

	// Check if entry is expired
	if time.Since(entry.timestamp) > c.ttl {
		return nil, false
	}

	return entry.commands, true
}

// SetCommands stores commands for a directory in cache
func (c *Cache) SetCommands(directory string, commands []history.CommandRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest entry if cache is full
	if len(c.dirCache) >= c.maxEntries {
		c.evictOldest()
	}

	c.dirCache[directory] = &directoryCacheEntry{
		commands:  commands,
		timestamp: time.Now(),
	}
}

// Invalidate removes a directory from cache
func (c *Cache) Invalidate(directory string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.dirCache, directory)
}

// InvalidateAll clears the entire cache
func (c *Cache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.dirCache = make(map[string]*directoryCacheEntry)
}

// evictOldest removes the oldest entry from cache
func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.dirCache {
		if oldestKey == "" || entry.timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.timestamp
		}
	}

	if oldestKey != "" {
		delete(c.dirCache, oldestKey)
	}
}

// Size returns the current number of cached entries
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.dirCache)
}

// Stats returns cache statistics
func (c *Cache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := CacheStats{
		Entries:    len(c.dirCache),
		MaxEntries: c.maxEntries,
		TTL:        c.ttl,
	}

	// Count expired entries
	now := time.Now()
	for _, entry := range c.dirCache {
		if now.Sub(entry.timestamp) > c.ttl {
			stats.ExpiredEntries++
		}
	}

	return stats
}

// CacheStats represents cache statistics
type CacheStats struct {
	Entries        int           `json:"entries"`
	MaxEntries     int           `json:"max_entries"`
	ExpiredEntries int           `json:"expired_entries"`
	TTL            time.Duration `json:"ttl"`
}
