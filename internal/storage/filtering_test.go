package storage

import (
	"os"
	"testing"
	"time"

	"github.com/ValGrace/command-history-tracker/pkg/history"
)

// TestFilterCommands tests the advanced filtering functionality
func TestFilterCommands(t *testing.T) {
	// Create temporary database
	dbPath := "test_filter.db"
	defer os.Remove(dbPath)

	storage := NewSQLiteStorage(dbPath)
	if err := storage.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()

	// Insert test data
	now := time.Now()
	testCommands := []history.CommandRecord{
		{
			ID:        "1",
			Command:   "git status",
			Directory: "/project",
			Timestamp: now,
			Shell:     history.Bash,
			ExitCode:  0,
			Duration:  100 * time.Millisecond,
		},
		{
			ID:        "2",
			Command:   "npm install",
			Directory: "/project",
			Timestamp: now.Add(-1 * time.Hour),
			Shell:     history.PowerShell,
			ExitCode:  0,
			Duration:  5 * time.Second,
		},
		{
			ID:        "3",
			Command:   "git commit -m 'test'",
			Directory: "/project",
			Timestamp: now.Add(-2 * time.Hour),
			Shell:     history.Bash,
			ExitCode:  0,
			Duration:  200 * time.Millisecond,
		},
		{
			ID:        "4",
			Command:   "docker ps",
			Directory: "/other",
			Timestamp: now.Add(-25 * time.Hour),
			Shell:     history.Bash,
			ExitCode:  0,
			Duration:  150 * time.Millisecond,
		},
		{
			ID:        "5",
			Command:   "npm test",
			Directory: "/project",
			Timestamp: now.Add(-3 * time.Hour),
			Shell:     history.PowerShell,
			ExitCode:  1,
			Duration:  10 * time.Second,
		},
	}

	for _, cmd := range testCommands {
		if err := storage.SaveCommand(cmd); err != nil {
			t.Fatalf("Failed to save command: %v", err)
		}
	}

	tests := []struct {
		name          string
		filters       CommandFilters
		expectedCount int
		expectedFirst string
	}{
		{
			name: "Filter by directory",
			filters: CommandFilters{
				Directory: "/project",
			},
			expectedCount: 4,
			expectedFirst: "git status",
		},
		{
			name: "Filter by pattern",
			filters: CommandFilters{
				Pattern: "git",
			},
			expectedCount: 2,
			expectedFirst: "git status",
		},
		{
			name: "Filter by shell type",
			filters: CommandFilters{
				ShellType: history.Bash,
			},
			expectedCount: 3,
			expectedFirst: "git status",
		},
		{
			name: "Filter by date range - last 24 hours",
			filters: CommandFilters{
				StartTime: now.Add(-24 * time.Hour),
				EndTime:   now.Add(1 * time.Hour),
			},
			expectedCount: 4,
			expectedFirst: "git status",
		},
		{
			name: "Filter by exit code",
			filters: CommandFilters{
				ExitCode: intPtr(1),
			},
			expectedCount: 1,
			expectedFirst: "npm test",
		},
		{
			name: "Combined filters - directory + pattern",
			filters: CommandFilters{
				Directory: "/project",
				Pattern:   "npm",
			},
			expectedCount: 2,
			expectedFirst: "npm install",
		},
		{
			name: "Combined filters - directory + shell",
			filters: CommandFilters{
				Directory: "/project",
				ShellType: history.PowerShell,
			},
			expectedCount: 2,
			expectedFirst: "npm install",
		},
		{
			name: "Combined filters - all criteria",
			filters: CommandFilters{
				Directory: "/project",
				Pattern:   "git",
				ShellType: history.Bash,
				StartTime: now.Add(-3 * time.Hour),
				EndTime:   now.Add(1 * time.Hour),
			},
			expectedCount: 2,
			expectedFirst: "git status",
		},
		{
			name: "Filter with limit",
			filters: CommandFilters{
				Directory: "/project",
				Limit:     2,
			},
			expectedCount: 2,
			expectedFirst: "git status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := storage.FilterCommands(tt.filters)
			if err != nil {
				t.Fatalf("FilterCommands failed: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			if tt.expectedCount > 0 && len(results) > 0 {
				if results[0].Command != tt.expectedFirst {
					t.Errorf("Expected first command to be '%s', got '%s'", tt.expectedFirst, results[0].Command)
				}
			}
		})
	}
}

// TestGetCommandsByShell tests shell-based filtering
func TestGetCommandsByShell(t *testing.T) {
	// Create temporary database
	dbPath := "test_shell_filter.db"
	defer os.Remove(dbPath)

	storage := NewSQLiteStorage(dbPath)
	if err := storage.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()

	// Insert test data
	now := time.Now()
	testCommands := []history.CommandRecord{
		{
			ID:        "1",
			Command:   "ls -la",
			Directory: "/test",
			Timestamp: now,
			Shell:     history.Bash,
			ExitCode:  0,
		},
		{
			ID:        "2",
			Command:   "Get-ChildItem",
			Directory: "/test",
			Timestamp: now.Add(-1 * time.Hour),
			Shell:     history.PowerShell,
			ExitCode:  0,
		},
		{
			ID:        "3",
			Command:   "pwd",
			Directory: "/test",
			Timestamp: now.Add(-2 * time.Hour),
			Shell:     history.Zsh,
			ExitCode:  0,
		},
	}

	for _, cmd := range testCommands {
		if err := storage.SaveCommand(cmd); err != nil {
			t.Fatalf("Failed to save command: %v", err)
		}
	}

	tests := []struct {
		name          string
		shellType     history.ShellType
		directory     string
		expectedCount int
	}{
		{
			name:          "Filter Bash commands",
			shellType:     history.Bash,
			directory:     "/test",
			expectedCount: 1,
		},
		{
			name:          "Filter PowerShell commands",
			shellType:     history.PowerShell,
			directory:     "/test",
			expectedCount: 1,
		},
		{
			name:          "Filter Zsh commands",
			shellType:     history.Zsh,
			directory:     "/test",
			expectedCount: 1,
		},
		{
			name:          "Filter all directories",
			shellType:     history.Bash,
			directory:     "",
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := storage.GetCommandsByShell(tt.shellType, tt.directory)
			if err != nil {
				t.Fatalf("GetCommandsByShell failed: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}
		})
	}
}

// TestGetCommandsByTimeRange tests time-based filtering
func TestGetCommandsByTimeRange(t *testing.T) {
	// Create temporary database
	dbPath := "test_time_filter.db"
	defer os.Remove(dbPath)

	storage := NewSQLiteStorage(dbPath)
	if err := storage.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}
	defer storage.Close()

	// Insert test data
	now := time.Now()
	testCommands := []history.CommandRecord{
		{
			ID:        "1",
			Command:   "recent",
			Directory: "/test",
			Timestamp: now,
			Shell:     history.Bash,
			ExitCode:  0,
		},
		{
			ID:        "2",
			Command:   "hour_ago",
			Directory: "/test",
			Timestamp: now.Add(-1 * time.Hour),
			Shell:     history.Bash,
			ExitCode:  0,
		},
		{
			ID:        "3",
			Command:   "day_ago",
			Directory: "/test",
			Timestamp: now.Add(-25 * time.Hour),
			Shell:     history.Bash,
			ExitCode:  0,
		},
		{
			ID:        "4",
			Command:   "week_ago",
			Directory: "/test",
			Timestamp: now.Add(-8 * 24 * time.Hour),
			Shell:     history.Bash,
			ExitCode:  0,
		},
	}

	for _, cmd := range testCommands {
		if err := storage.SaveCommand(cmd); err != nil {
			t.Fatalf("Failed to save command: %v", err)
		}
	}

	tests := []struct {
		name          string
		startTime     time.Time
		endTime       time.Time
		directory     string
		expectedCount int
	}{
		{
			name:          "Last hour",
			startTime:     now.Add(-1 * time.Hour),
			endTime:       now.Add(1 * time.Minute),
			directory:     "/test",
			expectedCount: 2,
		},
		{
			name:          "Last 24 hours",
			startTime:     now.Add(-24 * time.Hour),
			endTime:       now.Add(1 * time.Minute),
			directory:     "/test",
			expectedCount: 2,
		},
		{
			name:          "Last week",
			startTime:     now.Add(-7 * 24 * time.Hour),
			endTime:       now.Add(1 * time.Minute),
			directory:     "/test",
			expectedCount: 3,
		},
		{
			name:          "All time",
			startTime:     now.Add(-30 * 24 * time.Hour),
			endTime:       now.Add(1 * time.Minute),
			directory:     "/test",
			expectedCount: 4,
		},
		{
			name:          "All directories",
			startTime:     now.Add(-24 * time.Hour),
			endTime:       now.Add(1 * time.Minute),
			directory:     "",
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := storage.GetCommandsByTimeRange(tt.startTime, tt.endTime, tt.directory)
			if err != nil {
				t.Fatalf("GetCommandsByTimeRange failed: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			// Verify all results are within time range
			for _, cmd := range results {
				if cmd.Timestamp.Before(tt.startTime) || cmd.Timestamp.After(tt.endTime) {
					t.Errorf("Command timestamp %v is outside range [%v, %v]", cmd.Timestamp, tt.startTime, tt.endTime)
				}
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
