package storage

import (
	"github.com/ValGrace/command-history-tracker/internal/cache"
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"time"
)

// CachedStorage wraps a storage engine with caching capabilities
type CachedStorage struct {
	storage history.StorageEngine
	cache   *cache.Cache
}

// NewCachedStorage creates a new cached storage wrapper
func NewCachedStorage(storage history.StorageEngine, maxCacheEntries int, cacheTTL time.Duration) history.StorageEngine {
	return &CachedStorage{
		storage: storage,
		cache:   cache.New(maxCacheEntries, cacheTTL),
	}
}

// SaveCommand saves a command and invalidates the cache for its directory
func (cs *CachedStorage) SaveCommand(cmd history.CommandRecord) error {
	if err := cs.storage.SaveCommand(cmd); err != nil {
		return err
	}

	// Invalidate cache for this directory
	cs.cache.Invalidate(cmd.Directory)
	return nil
}

// GetCommandsByDirectory retrieves commands with caching
func (cs *CachedStorage) GetCommandsByDirectory(dir string) ([]history.CommandRecord, error) {
	// Try to get from cache first
	if commands, found := cs.cache.GetCommands(dir); found {
		return commands, nil
	}

	// Cache miss - fetch from storage
	commands, err := cs.storage.GetCommandsByDirectory(dir)
	if err != nil {
		return nil, err
	}

	// Store in cache
	cs.cache.SetCommands(dir, commands)

	return commands, nil
}

// GetDirectoriesWithHistory delegates to underlying storage
func (cs *CachedStorage) GetDirectoriesWithHistory() ([]string, error) {
	return cs.storage.GetDirectoriesWithHistory()
}

// SearchCommands delegates to underlying storage (no caching for search results)
func (cs *CachedStorage) SearchCommands(pattern string, dir string) ([]history.CommandRecord, error) {
	return cs.storage.SearchCommands(pattern, dir)
}

// CleanupOldCommands cleans up old commands and invalidates cache
func (cs *CachedStorage) CleanupOldCommands(retentionDays int) error {
	if err := cs.storage.CleanupOldCommands(retentionDays); err != nil {
		return err
	}

	// Invalidate entire cache after cleanup
	cs.cache.InvalidateAll()
	return nil
}

// Close closes the underlying storage
func (cs *CachedStorage) Close() error {
	return cs.storage.Close()
}

// GetCacheStats returns cache statistics
func (cs *CachedStorage) GetCacheStats() cache.CacheStats {
	return cs.cache.Stats()
}

// InvalidateCache invalidates the entire cache
func (cs *CachedStorage) InvalidateCache() {
	cs.cache.InvalidateAll()
}
