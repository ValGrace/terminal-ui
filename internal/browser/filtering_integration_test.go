package browser

import (
	"os"
	"testing"
	"time"

	"github.com/ValGrace/command-history-tracker/internal/storage"
	"github.com/ValGrace/command-history-tracker/pkg/history"
)

// TestFilteringIntegration tests the complete filtering workflow
func TestFilteringIntegration(t *testing.T) {
	// Create temporary database
	dbPath := "test_filtering_integration.db"
	defer os.Remove(dbPath)

	// Initialize storage
	store := storage.NewSQLiteStorage(dbPath)
	if err := store.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Create test data with various attributes
	now := time.Now()
	testCommands := []history.CommandRecord{
		{
			ID:        "1",
			Command:   "git status",
			Directory: "/project",
			Timestamp: now,
			Shell:     history.Bash,
			ExitCode:  0,
			Duration:  time.Second,
		},
		{
			ID:        "2",
			Command:   "npm install",
			Directory: "/project",
			Timestamp: now.Add(-1 * time.Hour),
			Shell:     history.PowerShell,
			ExitCode:  0,
			Duration:  30 * time.Second,
		},
		{
			ID:        "3",
			Command:   "git commit -m 'test'",
			Directory: "/project",
			Timestamp: now.Add(-2 * time.Hour),
			Shell:     history.Bash,
			ExitCode:  0,
			Duration:  2 * time.Second,
		},
		{
			ID:        "4",
			Command:   "docker ps",
			Directory: "/project",
			Timestamp: now.Add(-25 * time.Hour),
			Shell:     history.Bash,
			ExitCode:  0,
			Duration:  time.Second,
		},
		{
			ID:        "5",
			Command:   "go test",
			Directory: "/project",
			Timestamp: now.Add(-3 * time.Hour),
			Shell:     history.Bash,
			ExitCode:  1,
			Duration:  5 * time.Second,
		},
	}

	// Save test commands
	for _, cmd := range testCommands {
		if err := store.SaveCommand(cmd); err != nil {
			t.Fatalf("Failed to save command: %v", err)
		}
	}

	// Test 1: Text filtering
	t.Run("Text filtering", func(t *testing.T) {
		model := *NewUIModel(store, "/project")
		model.commands = testCommands
		model.searchQuery = "git"
		model = model.applyFilters()

		if len(model.filteredCmds) != 2 {
			t.Errorf("Expected 2 git commands, got %d", len(model.filteredCmds))
		}

		for _, cmd := range model.filteredCmds {
			if cmd.Command != "git status" && cmd.Command != "git commit -m 'test'" {
				t.Errorf("Unexpected command in filtered results: %s", cmd.Command)
			}
		}
	})

	// Test 2: Shell type filtering
	t.Run("Shell type filtering", func(t *testing.T) {
		model := *NewUIModel(store, "/project")
		model.commands = testCommands
		model.shellFilter = history.PowerShell
		model = model.applyFilters()

		if len(model.filteredCmds) != 1 {
			t.Errorf("Expected 1 PowerShell command, got %d", len(model.filteredCmds))
		}

		if len(model.filteredCmds) > 0 && model.filteredCmds[0].Shell != history.PowerShell {
			t.Errorf("Expected PowerShell command, got %v", model.filteredCmds[0].Shell)
		}
	})

	// Test 3: Date range filtering (Today)
	t.Run("Date range filtering - Today", func(t *testing.T) {
		model := *NewUIModel(store, "/project")
		model.commands = testCommands
		model.dateFilter.Enabled = true
		model.dateFilter.Preset = Today
		model = model.applyDateFilter()

		// Should get commands from today (within last 24 hours)
		if len(model.filteredCmds) != 4 {
			t.Errorf("Expected 4 commands from today, got %d", len(model.filteredCmds))
		}

		for _, cmd := range model.filteredCmds {
			if time.Since(cmd.Timestamp) > 24*time.Hour {
				t.Errorf("Command %s is older than 24 hours", cmd.Command)
			}
		}
	})

	// Test 4: Combined filtering (text + shell)
	t.Run("Combined filtering - text + shell", func(t *testing.T) {
		model := *NewUIModel(store, "/project")
		model.commands = testCommands
		model.searchQuery = "git"
		model.shellFilter = history.Bash
		model = model.applyFilters()

		if len(model.filteredCmds) != 2 {
			t.Errorf("Expected 2 git+bash commands, got %d", len(model.filteredCmds))
		}

		for _, cmd := range model.filteredCmds {
			if cmd.Shell != history.Bash {
				t.Errorf("Expected Bash shell, got %v", cmd.Shell)
			}
			if cmd.Command != "git status" && cmd.Command != "git commit -m 'test'" {
				t.Errorf("Unexpected command: %s", cmd.Command)
			}
		}
	})

	// Test 5: Combined filtering (text + date + shell)
	t.Run("Combined filtering - text + date + shell", func(t *testing.T) {
		model := *NewUIModel(store, "/project")
		model.commands = testCommands
		model.searchQuery = "git"
		model.shellFilter = history.Bash
		model.dateFilter.Enabled = true
		model.dateFilter.Preset = Today
		model = model.applyDateFilter()

		// Should get git commands from bash shell within today
		if len(model.filteredCmds) != 2 {
			t.Errorf("Expected 2 filtered commands, got %d", len(model.filteredCmds))
		}

		for _, cmd := range model.filteredCmds {
			if cmd.Shell != history.Bash {
				t.Errorf("Expected Bash shell, got %v", cmd.Shell)
			}
			if time.Since(cmd.Timestamp) > 24*time.Hour {
				t.Errorf("Command is older than 24 hours")
			}
		}
	})

	// Test 6: Clear filters
	t.Run("Clear filters", func(t *testing.T) {
		model := *NewUIModel(store, "/project")
		model.commands = testCommands
		model.searchQuery = "git"
		model.shellFilter = history.Bash
		model.dateFilter.Enabled = true
		model = model.clearFilters()

		if model.searchQuery != "" {
			t.Error("Search query should be cleared")
		}
		if model.shellFilter != history.Unknown {
			t.Error("Shell filter should be cleared")
		}
		if model.dateFilter.Enabled {
			t.Error("Date filter should be disabled")
		}
		if len(model.filteredCmds) != len(testCommands) {
			t.Errorf("Expected all commands after clearing filters, got %d", len(model.filteredCmds))
		}
	})

	// Test 7: Filter cycling
	t.Run("Shell filter cycling", func(t *testing.T) {
		model := *NewUIModel(store, "/project")

		expectedCycle := []history.ShellType{
			history.PowerShell,
			history.Bash,
			history.Zsh,
			history.Cmd,
			history.Unknown,
		}

		for i, expected := range expectedCycle {
			model.shellFilter = model.getNextShellFilter()
			if model.shellFilter != expected {
				t.Errorf("Cycle step %d: expected %v, got %v", i, expected, model.shellFilter)
			}
		}
	})

	// Test 8: Date preset cycling
	t.Run("Date preset cycling", func(t *testing.T) {
		model := *NewUIModel(store, "/project")
		model.dateFilter.Enabled = true
		model.dateFilter.Preset = Today

		presets := []DatePreset{Yesterday, ThisWeek, LastWeek, ThisMonth, LastMonth, Today}

		for _, expected := range presets {
			model = model.cycleDatePreset()
			if model.dateFilter.Preset != expected {
				t.Errorf("Expected preset %v, got %v", expected, model.dateFilter.Preset)
			}
		}
	})
}

// TestFilteringWithStorageBackend tests filtering using actual storage queries
func TestFilteringWithStorageBackend(t *testing.T) {
	// Create temporary database
	dbPath := "test_filtering_storage.db"
	defer os.Remove(dbPath)

	// Initialize storage
	store := storage.NewSQLiteStorage(dbPath)
	if err := store.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Create test data
	now := time.Now()
	testCommands := []history.CommandRecord{
		{
			ID:        "1",
			Command:   "git status",
			Directory: "/project",
			Timestamp: now,
			Shell:     history.Bash,
			ExitCode:  0,
		},
		{
			ID:        "2",
			Command:   "npm install",
			Directory: "/project",
			Timestamp: now.Add(-1 * time.Hour),
			Shell:     history.PowerShell,
			ExitCode:  0,
		},
		{
			ID:        "3",
			Command:   "git commit",
			Directory: "/project",
			Timestamp: now.Add(-2 * time.Hour),
			Shell:     history.Bash,
			ExitCode:  0,
		},
	}

	// Save test commands
	for _, cmd := range testCommands {
		if err := store.SaveCommand(cmd); err != nil {
			t.Fatalf("Failed to save command: %v", err)
		}
	}

	// Test storage-level text search
	t.Run("Storage text search", func(t *testing.T) {
		results, err := store.SearchCommands("git", "/project")
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
	})

	// Test storage-level shell filtering
	t.Run("Storage shell filtering", func(t *testing.T) {
		results, err := store.GetCommandsByShell(history.Bash, "/project")
		if err != nil {
			t.Fatalf("Shell filter failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 Bash commands, got %d", len(results))
		}
	})

	// Test storage-level date range filtering
	t.Run("Storage date range filtering", func(t *testing.T) {
		startTime := now.Add(-3 * time.Hour)
		endTime := now.Add(1 * time.Hour)

		results, err := store.GetCommandsByTimeRange(startTime, endTime, "/project")
		if err != nil {
			t.Fatalf("Date range filter failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 commands in range, got %d", len(results))
		}
	})

	// Test storage-level combined filtering
	t.Run("Storage combined filtering", func(t *testing.T) {
		filters := storage.CommandFilters{
			Directory: "/project",
			Pattern:   "git",
			ShellType: history.Bash,
			StartTime: now.Add(-3 * time.Hour),
			EndTime:   now.Add(1 * time.Hour),
		}

		results, err := store.FilterCommands(filters)
		if err != nil {
			t.Fatalf("Combined filter failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 filtered commands, got %d", len(results))
		}

		for _, cmd := range results {
			if cmd.Shell != history.Bash {
				t.Errorf("Expected Bash shell, got %v", cmd.Shell)
			}
		}
	})
}
