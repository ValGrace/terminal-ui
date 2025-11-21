package browser

import (
	"github.com/ValGrace/command-history-tracker/pkg/history"
	tea "github.com/charmbracelet/bubbletea"
)

// Message types for bubbletea commands

// directoryHistoryMsg contains commands for a specific directory
type directoryHistoryMsg struct {
	commands []history.CommandRecord
}

// directoryTreeMsg contains directory tree information
type directoryTreeMsg struct {
	directories []history.DirectoryIndex
}

// errorMsg contains error information
type errorMsg struct {
	error error
}

// Commands for loading data asynchronously

// loadDirectoryHistory loads command history for a specific directory
func loadDirectoryHistory(storage history.StorageEngine, dir string) tea.Cmd {
	return func() tea.Msg {
		commands, err := storage.GetCommandsByDirectory(dir)
		if err != nil {
			return errorMsg{error: err}
		}
		// Commands are already sorted by timestamp DESC from SQLite storage
		// This ensures chronological sorting with recent-first order
		return directoryHistoryMsg{commands: commands}
	}
}

// loadDirectoryTree loads the directory tree with command counts
func loadDirectoryTree(storage history.StorageEngine) tea.Cmd {
	return func() tea.Msg {
		// Try to use GetDirectoryStats if available (for SQLite storage)
		if statsStorage, ok := storage.(interface {
			GetDirectoryStats() ([]history.DirectoryIndex, error)
		}); ok {
			directories, err := statsStorage.GetDirectoryStats()
			if err == nil {
				return directoryTreeMsg{directories: directories}
			}
		}

		// Fallback to manual calculation
		directories, err := storage.GetDirectoriesWithHistory()
		if err != nil {
			return errorMsg{error: err}
		}

		// Convert to DirectoryIndex format
		var dirIndexes []history.DirectoryIndex
		for _, dir := range directories {
			// Get command count for this directory
			commands, err := storage.GetCommandsByDirectory(dir)
			if err != nil {
				continue // Skip directories with errors
			}

			// Find most recent command for last used time
			var lastUsed history.CommandRecord
			for _, cmd := range commands {
				if lastUsed.Timestamp.IsZero() || cmd.Timestamp.After(lastUsed.Timestamp) {
					lastUsed = cmd
				}
			}

			dirIndex := history.DirectoryIndex{
				Path:         dir,
				CommandCount: len(commands),
				LastUsed:     lastUsed.Timestamp,
				IsActive:     true,
			}

			dirIndexes = append(dirIndexes, dirIndex)
		}

		return directoryTreeMsg{directories: dirIndexes}
	}
}

// searchCommands searches for commands matching a pattern
func searchCommands(storage history.StorageEngine, pattern, dir string) tea.Cmd {
	return func() tea.Msg {
		commands, err := storage.SearchCommands(pattern, dir)
		if err != nil {
			return errorMsg{error: err}
		}
		return directoryHistoryMsg{commands: commands}
	}
}
