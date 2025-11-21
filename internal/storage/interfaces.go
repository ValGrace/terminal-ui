package storage

import (
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"time"
)

// StorageEngine defines the interface for command history storage
type StorageEngine interface {
	// Initialize sets up the storage engine
	Initialize() error

	// SaveCommand stores a command record
	SaveCommand(cmd history.CommandRecord) error

	// GetCommandsByDirectory retrieves commands for a specific directory
	GetCommandsByDirectory(dir string) ([]history.CommandRecord, error)

	// GetDirectoriesWithHistory returns all directories that have command history
	GetDirectoriesWithHistory() ([]string, error)

	// CleanupOldCommands removes commands older than specified retention period
	CleanupOldCommands(retentionDays int) error

	// SearchCommands finds commands matching a pattern
	SearchCommands(pattern string, dir string) ([]history.CommandRecord, error)

	// Close closes the storage connection
	Close() error
}

// FilterableStorageEngine extends StorageEngine with advanced filtering capabilities
type FilterableStorageEngine interface {
	StorageEngine

	// GetCommandsByTimeRange retrieves commands within a specific time range
	GetCommandsByTimeRange(startTime, endTime time.Time, dir string) ([]history.CommandRecord, error)

	// GetCommandsByShell retrieves commands filtered by shell type
	GetCommandsByShell(shellType history.ShellType, dir string) ([]history.CommandRecord, error)
}

// BatchStorageEngine extends StorageEngine with batch operations
type BatchStorageEngine interface {
	StorageEngine

	// BatchSaveCommands saves multiple commands in a single transaction
	BatchSaveCommands(commands []history.CommandRecord) error
}

// StatsStorageEngine extends StorageEngine with statistics operations
type StatsStorageEngine interface {
	StorageEngine

	// GetDirectoryStats returns directory statistics
	GetDirectoryStats() ([]history.DirectoryIndex, error)
}

// NewStorageEngine creates a new storage engine based on the storage type
func NewStorageEngine(storageType string, dbPath string) (StorageEngine, error) {
	switch storageType {
	case "sqlite", "":
		return NewSQLiteStorage(dbPath), nil
	default:
		return NewSQLiteStorage(dbPath), nil // Default to SQLite
	}
}
