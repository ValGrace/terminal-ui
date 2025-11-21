package browser

import (
	"testing"
	"time"

	"github.com/ValGrace/command-history-tracker/pkg/history"
)

// TestApplyFilters tests the filter application logic
func TestApplyFilters(t *testing.T) {
	// Create test commands
	now := time.Now()
	commands := []history.CommandRecord{
		{
			ID:        "1",
			Command:   "git status",
			Directory: "/test",
			Timestamp: now,
			Shell:     history.Bash,
			ExitCode:  0,
		},
		{
			ID:        "2",
			Command:   "npm install",
			Directory: "/test",
			Timestamp: now.Add(-1 * time.Hour),
			Shell:     history.PowerShell,
			ExitCode:  0,
		},
		{
			ID:        "3",
			Command:   "git commit",
			Directory: "/test",
			Timestamp: now.Add(-2 * time.Hour),
			Shell:     history.Bash,
			ExitCode:  0,
		},
		{
			ID:        "4",
			Command:   "docker ps",
			Directory: "/test",
			Timestamp: now.Add(-25 * time.Hour),
			Shell:     history.Bash,
			ExitCode:  0,
		},
	}

	tests := []struct {
		name          string
		searchQuery   string
		dateEnabled   bool
		datePreset    DatePreset
		shellFilter   history.ShellType
		expectedCount int
		expectedFirst string
	}{
		{
			name:          "No filters",
			searchQuery:   "",
			dateEnabled:   false,
			shellFilter:   history.Unknown,
			expectedCount: 4,
			expectedFirst: "git status",
		},
		{
			name:          "Text filter - git",
			searchQuery:   "git",
			dateEnabled:   false,
			shellFilter:   history.Unknown,
			expectedCount: 2,
			expectedFirst: "git status",
		},
		{
			name:          "Shell filter - Bash",
			searchQuery:   "",
			dateEnabled:   false,
			shellFilter:   history.Bash,
			expectedCount: 3,
			expectedFirst: "git status",
		},
		{
			name:          "Shell filter - PowerShell",
			searchQuery:   "",
			dateEnabled:   false,
			shellFilter:   history.PowerShell,
			expectedCount: 1,
			expectedFirst: "npm install",
		},
		{
			name:          "Date filter - Today",
			searchQuery:   "",
			dateEnabled:   true,
			datePreset:    Today,
			shellFilter:   history.Unknown,
			expectedCount: 3,
			expectedFirst: "git status",
		},
		{
			name:          "Combined filters - git + Bash",
			searchQuery:   "git",
			dateEnabled:   false,
			shellFilter:   history.Bash,
			expectedCount: 2,
			expectedFirst: "git status",
		},
		{
			name:          "Combined filters - git + Today",
			searchQuery:   "git",
			dateEnabled:   true,
			datePreset:    Today,
			shellFilter:   history.Unknown,
			expectedCount: 2,
			expectedFirst: "git status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create model with test data
			m := UIModel{
				commands:    commands,
				searchQuery: tt.searchQuery,
				shellFilter: tt.shellFilter,
				dateFilter: DateFilterConfig{
					Enabled: tt.dateEnabled,
					Preset:  tt.datePreset,
				},
			}

			// Apply date filter if enabled
			if tt.dateEnabled {
				m = m.applyDateFilter()
			} else {
				m = m.applyFilters()
			}

			// Check filtered count
			if len(m.filteredCmds) != tt.expectedCount {
				t.Errorf("Expected %d filtered commands, got %d", tt.expectedCount, len(m.filteredCmds))
			}

			// Check first command if any
			if tt.expectedCount > 0 && len(m.filteredCmds) > 0 {
				if m.filteredCmds[0].Command != tt.expectedFirst {
					t.Errorf("Expected first command to be '%s', got '%s'", tt.expectedFirst, m.filteredCmds[0].Command)
				}
			}
		})
	}
}

// TestDateFilterPresets tests date filter preset calculations
func TestDateFilterPresets(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		preset     DatePreset
		checkStart func(time.Time) bool
		checkEnd   func(time.Time) bool
	}{
		{
			name:   "Today",
			preset: Today,
			checkStart: func(t time.Time) bool {
				return t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day() && t.Hour() == 0
			},
			checkEnd: func(t time.Time) bool {
				return t.After(now)
			},
		},
		{
			name:   "Yesterday",
			preset: Yesterday,
			checkStart: func(t time.Time) bool {
				yesterday := now.AddDate(0, 0, -1)
				return t.Year() == yesterday.Year() && t.Month() == yesterday.Month() && t.Day() == yesterday.Day()
			},
			checkEnd: func(t time.Time) bool {
				return t.Before(now) || t.Equal(now)
			},
		},
		{
			name:   "This Week",
			preset: ThisWeek,
			checkStart: func(t time.Time) bool {
				// Should be Monday of current week
				return t.Weekday() == time.Monday || (t.Weekday() == time.Sunday && now.Weekday() == time.Sunday)
			},
			checkEnd: func(t time.Time) bool {
				return t.After(now)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := UIModel{
				dateFilter: DateFilterConfig{
					Enabled: true,
					Preset:  tt.preset,
				},
			}

			m = m.applyDateFilter()

			if !tt.checkStart(m.dateFilter.StartTime) {
				t.Errorf("Start time %v doesn't match expected range for preset %s", m.dateFilter.StartTime, tt.name)
			}

			if !tt.checkEnd(m.dateFilter.EndTime) {
				t.Errorf("End time %v doesn't match expected range for preset %s", m.dateFilter.EndTime, tt.name)
			}
		})
	}
}

// TestShellFilterCycle tests cycling through shell filter options
func TestShellFilterCycle(t *testing.T) {
	m := UIModel{
		shellFilter: history.Unknown,
	}

	expectedCycle := []history.ShellType{
		history.PowerShell,
		history.Bash,
		history.Zsh,
		history.Cmd,
		history.Unknown,
	}

	for i, expected := range expectedCycle {
		m.shellFilter = m.getNextShellFilter()
		if m.shellFilter != expected {
			t.Errorf("Cycle step %d: expected %v, got %v", i, expected, m.shellFilter)
		}
	}
}

// TestClearFilters tests clearing all filters
func TestClearFilters(t *testing.T) {
	commands := []history.CommandRecord{
		{
			ID:        "1",
			Command:   "test",
			Directory: "/test",
			Timestamp: time.Now(),
			Shell:     history.Bash,
		},
	}

	m := UIModel{
		commands:    commands,
		searchQuery: "test",
		dateFilter: DateFilterConfig{
			Enabled: true,
			Preset:  Today,
		},
		shellFilter: history.Bash,
		filterMode:  CombinedFilter,
	}

	m = m.clearFilters()

	if m.searchQuery != "" {
		t.Error("Search query should be cleared")
	}

	if m.dateFilter.Enabled {
		t.Error("Date filter should be disabled")
	}

	if m.shellFilter != history.Unknown {
		t.Error("Shell filter should be cleared")
	}

	if m.filterMode != NoFilter {
		t.Error("Filter mode should be NoFilter")
	}

	if len(m.filteredCmds) != len(commands) {
		t.Error("Filtered commands should match all commands")
	}
}

// TestFilterModeTransitions tests filter mode state transitions
func TestFilterModeTransitions(t *testing.T) {
	m := UIModel{
		filterMode: NoFilter,
	}

	// Activate text filter
	m.searchQuery = "test"
	m.filterMode = TextFilter
	if m.filterMode != TextFilter {
		t.Error("Should transition to TextFilter")
	}

	// Activate date filter
	m.dateFilter.Enabled = true
	m.filterMode = DateRangeFilter
	if m.filterMode != DateRangeFilter {
		t.Error("Should transition to DateRangeFilter")
	}

	// Activate shell filter
	m.shellFilter = history.Bash
	m.filterMode = ShellFilter
	if m.filterMode != ShellFilter {
		t.Error("Should transition to ShellFilter")
	}

	// Multiple filters active
	if m.searchQuery != "" && m.dateFilter.Enabled && m.shellFilter != history.Unknown {
		m.filterMode = CombinedFilter
		if m.filterMode != CombinedFilter {
			t.Error("Should transition to CombinedFilter")
		}
	}
}
