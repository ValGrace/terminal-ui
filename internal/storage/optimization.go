package storage

import (
	"fmt"
	"time"

	"github.com/ValGrace/command-history-tracker/pkg/history"
)

// OptimizationEngine provides storage optimization features
type OptimizationEngine struct {
	storage StorageEngine
}

// NewOptimizationEngine creates a new optimization engine
func NewOptimizationEngine(storage StorageEngine) *OptimizationEngine {
	return &OptimizationEngine{
		storage: storage,
	}
}

// CleanupPolicy defines rules for command cleanup
type CleanupPolicy struct {
	MaxAge        time.Duration       // Maximum age of commands to keep
	MaxCommands   int                 // Maximum number of commands per directory
	ExcludeShells []history.ShellType // Shells to exclude from cleanup
	KeepPatterns  []string            // Command patterns to always keep
}

// DefaultCleanupPolicy returns a sensible default cleanup policy
func DefaultCleanupPolicy() *CleanupPolicy {
	return &CleanupPolicy{
		MaxAge:        90 * 24 * time.Hour,                  // 90 days
		MaxCommands:   1000,                                 // 1000 commands per directory
		ExcludeShells: []history.ShellType{},                // Don't exclude any shells by default
		KeepPatterns:  []string{"git", "docker", "kubectl"}, // Keep important commands
	}
}

// ApplyCleanupPolicy applies the cleanup policy to the storage
func (o *OptimizationEngine) ApplyCleanupPolicy(policy *CleanupPolicy) error {
	if policy == nil {
		policy = DefaultCleanupPolicy()
	}

	// Get all directories with history
	directories, err := o.storage.GetDirectoriesWithHistory()
	if err != nil {
		return fmt.Errorf("failed to get directories: %w", err)
	}

	var totalCleaned int

	for _, dir := range directories {
		cleaned, err := o.cleanupDirectory(dir, policy)
		if err != nil {
			fmt.Printf("Warning: failed to cleanup directory %s: %v\n", dir, err)
			continue
		}
		totalCleaned += cleaned
	}

	fmt.Printf("Cleanup completed: removed %d commands across %d directories\n", totalCleaned, len(directories))
	return nil
}

// cleanupDirectory applies cleanup policy to a specific directory
func (o *OptimizationEngine) cleanupDirectory(dir string, policy *CleanupPolicy) (int, error) {
	// Get all commands for the directory
	commands, err := o.storage.GetCommandsByDirectory(dir)
	if err != nil {
		return 0, err
	}

	if len(commands) <= policy.MaxCommands {
		// No cleanup needed based on count
		return 0, nil
	}

	// Sort commands by timestamp (newest first) - they should already be sorted
	// Keep the most recent MaxCommands, but respect other policy rules
	var toDelete []history.CommandRecord
	cutoffTime := time.Now().Add(-policy.MaxAge)

	for i, cmd := range commands {
		shouldDelete := false

		// Check if command is too old
		if cmd.Timestamp.Before(cutoffTime) {
			shouldDelete = true
		}

		// Check if we have too many commands (keep the first MaxCommands)
		if i >= policy.MaxCommands {
			shouldDelete = true
		}

		// Check if shell should be excluded
		for _, excludeShell := range policy.ExcludeShells {
			if cmd.Shell == excludeShell {
				shouldDelete = false
				break
			}
		}

		// Check if command matches keep patterns
		for _, pattern := range policy.KeepPatterns {
			if containsPattern(cmd.Command, pattern) {
				shouldDelete = false
				break
			}
		}

		if shouldDelete {
			toDelete = append(toDelete, cmd)
		}
	}

	// For now, we'll use the existing CleanupOldCommands method
	// In a real implementation, we'd want a more granular delete method
	if len(toDelete) > 0 {
		// Calculate the age threshold that would delete the oldest commands we want to delete
		if len(toDelete) > 0 {
			oldestToDelete := toDelete[len(toDelete)-1].Timestamp
			daysSince := int(time.Since(oldestToDelete).Hours() / 24)
			return len(toDelete), o.storage.CleanupOldCommands(daysSince)
		}
	}

	return 0, nil
}

// containsPattern checks if a command contains a specific pattern
func containsPattern(command, pattern string) bool {
	// Simple substring match - could be enhanced with regex
	return len(command) >= len(pattern) &&
		(command[:len(pattern)] == pattern ||
			command[len(command)-len(pattern):] == pattern ||
			containsSubstring(command, pattern))
}

// containsSubstring checks if command contains pattern as substring
func containsSubstring(command, pattern string) bool {
	for i := 0; i <= len(command)-len(pattern); i++ {
		if command[i:i+len(pattern)] == pattern {
			return true
		}
	}
	return false
}

// OptimizeStorage performs various storage optimizations
func (o *OptimizationEngine) OptimizeStorage() error {
	// Apply default cleanup policy
	if err := o.ApplyCleanupPolicy(nil); err != nil {
		return fmt.Errorf("failed to apply cleanup policy: %w", err)
	}

	// If storage supports stats, refresh them
	if statsStorage, ok := o.storage.(StatsStorageEngine); ok {
		if sqliteStorage, ok := statsStorage.(*SQLiteStorage); ok {
			if err := sqliteStorage.refreshDirectoryStats(); err != nil {
				fmt.Printf("Warning: failed to refresh directory stats: %v\n", err)
			}
		}
	}

	return nil
}

// GetStorageStats returns statistics about storage usage
func (o *OptimizationEngine) GetStorageStats() (*StorageStats, error) {
	directories, err := o.storage.GetDirectoriesWithHistory()
	if err != nil {
		return nil, fmt.Errorf("failed to get directories: %w", err)
	}

	stats := &StorageStats{
		TotalDirectories: len(directories),
		DirectoryStats:   make(map[string]DirectoryStats),
	}

	for _, dir := range directories {
		commands, err := o.storage.GetCommandsByDirectory(dir)
		if err != nil {
			continue
		}

		dirStats := DirectoryStats{
			CommandCount: len(commands),
		}

		if len(commands) > 0 {
			dirStats.OldestCommand = commands[len(commands)-1].Timestamp
			dirStats.NewestCommand = commands[0].Timestamp
		}

		// Count by shell type
		shellCounts := make(map[history.ShellType]int)
		for _, cmd := range commands {
			shellCounts[cmd.Shell]++
		}
		dirStats.ShellCounts = shellCounts

		stats.DirectoryStats[dir] = dirStats
		stats.TotalCommands += len(commands)
	}

	return stats, nil
}

// StorageStats represents storage usage statistics
type StorageStats struct {
	TotalDirectories int                       `json:"total_directories"`
	TotalCommands    int                       `json:"total_commands"`
	DirectoryStats   map[string]DirectoryStats `json:"directory_stats"`
}

// DirectoryStats represents statistics for a single directory
type DirectoryStats struct {
	CommandCount  int                       `json:"command_count"`
	OldestCommand time.Time                 `json:"oldest_command"`
	NewestCommand time.Time                 `json:"newest_command"`
	ShellCounts   map[history.ShellType]int `json:"shell_counts"`
}

// RetentionManager handles automatic cleanup based on retention policies
type RetentionManager struct {
	optimization *OptimizationEngine
	policy       *CleanupPolicy
	ticker       *time.Ticker
	stopChan     chan bool
}

// NewRetentionManager creates a new retention manager
func NewRetentionManager(storage StorageEngine, policy *CleanupPolicy) *RetentionManager {
	if policy == nil {
		policy = DefaultCleanupPolicy()
	}

	return &RetentionManager{
		optimization: NewOptimizationEngine(storage),
		policy:       policy,
		stopChan:     make(chan bool),
	}
}

// Start begins automatic cleanup based on the specified interval
func (r *RetentionManager) Start(interval time.Duration) {
	r.ticker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-r.ticker.C:
				if err := r.optimization.ApplyCleanupPolicy(r.policy); err != nil {
					fmt.Printf("Automatic cleanup failed: %v\n", err)
				}
			case <-r.stopChan:
				return
			}
		}
	}()
}

// Stop stops the automatic cleanup
func (r *RetentionManager) Stop() {
	if r.ticker != nil {
		r.ticker.Stop()
	}
	r.stopChan <- true
}

// UpdatePolicy updates the cleanup policy
func (r *RetentionManager) UpdatePolicy(policy *CleanupPolicy) {
	r.policy = policy
}
