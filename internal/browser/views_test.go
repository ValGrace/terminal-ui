package browser

import (
	"strings"
	"testing"
	"time"

	"github.com/ValGrace/command-history-tracker/pkg/history"
)

// Test View Rendering Functions

func TestRenderBreadcrumbs(t *testing.T) {
	model, _ := setupTestModel()

	// Test with simple path
	model.breadcrumbs = []string{"/", "/home", "/home/user"}
	model.currentDir = "/home/user"
	result := model.renderBreadcrumbs()

	if !strings.Contains(result, "📁") {
		t.Error("Expected breadcrumbs to contain folder icon")
	}

	// The breadcrumb shows directory names, not full paths
	if !strings.Contains(result, "user") {
		t.Error("Expected breadcrumbs to contain current directory name")
	}

	// Test with empty breadcrumbs
	model.breadcrumbs = []string{}
	result = model.renderBreadcrumbs()

	if !strings.Contains(result, ".") {
		t.Error("Expected empty breadcrumbs to show current directory")
	}
}

func TestFormatCommandLine(t *testing.T) {
	model, _ := setupTestModel()
	model.width = 80

	// Test successful command
	cmd := createTestCommand("1", "ls -la", "/home/user", history.Bash, 0)
	result := model.formatCommandLine(cmd, false)

	if !strings.Contains(result, "ls -la") {
		t.Error("Expected formatted line to contain command text")
	}

	if !strings.Contains(result, "[bash]") {
		t.Error("Expected formatted line to contain shell indicator")
	}

	if !strings.Contains(result, "✅") {
		t.Error("Expected formatted line to contain success indicator")
	}

	// Test failed command
	failedCmd := createTestCommand("2", "failed-command", "/home/user", history.Bash, 1)
	result = model.formatCommandLine(failedCmd, false)

	if !strings.Contains(result, "❌1") {
		t.Error("Expected formatted line to contain failure indicator with exit code")
	}

	// Test selected command
	result = model.formatCommandLine(cmd, true)
	// The result should be styled differently for selected commands
	// We can't easily test the styling, but we can verify the content is there
	if !strings.Contains(result, "ls -la") {
		t.Error("Expected selected command line to contain command text")
	}
}

func TestFormatDirectoryLine(t *testing.T) {
	model, _ := setupTestModel()
	model.width = 80

	now := time.Now()
	dir := history.DirectoryIndex{
		Path:         "/home/user/project",
		CommandCount: 25,
		LastUsed:     now.Add(-30 * time.Minute),
		IsActive:     true,
	}

	result := model.formatDirectoryLine(dir, false)

	if !strings.Contains(result, "(25 commands)") {
		t.Error("Expected directory line to contain command count")
	}

	if !strings.Contains(result, "project") {
		t.Error("Expected directory line to contain directory name")
	}

	// Test inactive directory
	dir.IsActive = false
	result = model.formatDirectoryLine(dir, false)

	if !strings.Contains(result, "💤") {
		t.Error("Expected inactive directory to show sleep indicator")
	}

	// Test high command count
	dir.CommandCount = 150
	result = model.formatDirectoryLine(dir, false)

	if !strings.Contains(result, "(150+ commands)") {
		t.Error("Expected high command count to show '+' indicator")
	}
}

func TestRenderFilterPanel(t *testing.T) {
	model, _ := setupTestModel()

	// Test with no filters
	result := model.renderFilterPanel()

	if !strings.Contains(result, "None active") {
		t.Error("Expected filter panel to show 'None active' when no filters are set")
	}

	// Test with text filter
	model.searchQuery = "git"
	result = model.renderFilterPanel()

	if !strings.Contains(result, "Text: git") {
		t.Error("Expected filter panel to show text filter")
	}

	// Test with date filter
	model.dateFilter.Enabled = true
	model.dateFilter.Preset = Today
	result = model.renderFilterPanel()

	if !strings.Contains(result, "Date: Today") {
		t.Error("Expected filter panel to show date filter")
	}

	// Test with shell filter
	model.shellFilter = history.PowerShell
	result = model.renderFilterPanel()

	if !strings.Contains(result, "Shell: powershell") {
		t.Error("Expected filter panel to show shell filter")
	}
}

func TestGetDatePresetName(t *testing.T) {
	model, _ := setupTestModel()

	tests := []struct {
		preset   DatePreset
		expected string
	}{
		{Today, "Today"},
		{Yesterday, "Yesterday"},
		{ThisWeek, "This Week"},
		{LastWeek, "Last Week"},
		{ThisMonth, "This Month"},
		{LastMonth, "Last Month"},
	}

	for _, test := range tests {
		result := model.getDatePresetName(test.preset)
		if result != test.expected {
			t.Errorf("Expected preset name '%s', got '%s'", test.expected, result)
		}
	}
}

func TestRenderDirectoryHistoryView(t *testing.T) {
	model, _ := setupTestModel()
	model.width = 80
	model.height = 24
	model.currentDir = "/home/user"
	model.commands = []history.CommandRecord{
		createTestCommand("1", "ls -la", "/home/user", history.Bash, 0),
		createTestCommand("2", "git status", "/home/user", history.Bash, 0),
	}
	model.filteredCmds = model.commands

	result := model.renderDirectoryHistoryView()

	if !strings.Contains(result, "Current Directory History") {
		t.Error("Expected view to contain 'Current Directory History' header")
	}

	if !strings.Contains(result, "ls -la") {
		t.Error("Expected view to contain command text")
	}

	if !strings.Contains(result, "git status") {
		t.Error("Expected view to contain second command")
	}

	// Test with search mode
	model.searchMode = true
	model.searchQuery = "git"
	result = model.renderDirectoryHistoryView()

	if !strings.Contains(result, "Search: git") {
		t.Error("Expected view to show search query in header")
	}

	// Test with no commands
	model.filteredCmds = []history.CommandRecord{}
	result = model.renderDirectoryHistoryView()

	if !strings.Contains(result, "No commands found") {
		t.Error("Expected view to show 'No commands found' message")
	}
}

func TestRenderDirectoryTreeView(t *testing.T) {
	model, _ := setupTestModel()
	model.width = 80
	model.height = 24
	model.directories = []history.DirectoryIndex{
		{
			Path:         "/home/user",
			CommandCount: 10,
			LastUsed:     time.Now(),
			IsActive:     true,
		},
		{
			Path:         "/home/user/project",
			CommandCount: 5,
			LastUsed:     time.Now().Add(-time.Hour),
			IsActive:     true,
		},
	}
	// Need to organize directories into tree structure
	model.directoryTree = model.organizeDirectoriesHierarchically()

	result := model.renderDirectoryTreeView()

	if !strings.Contains(result, "Directory Tree") {
		t.Error("Expected view to contain 'Directory Tree' header")
	}

	// The view shows directory names, not full paths
	if !strings.Contains(result, "user") {
		t.Error("Expected view to contain directory name")
	}

	// Command count format varies (e.g., "📈10" for counts > 10)
	if !strings.Contains(result, "10") {
		t.Error("Expected view to contain command count")
	}

	// Test with no directories
	model.directories = []history.DirectoryIndex{}
	model.directoryTree = model.organizeDirectoriesHierarchically()
	result = model.renderDirectoryTreeView()

	if !strings.Contains(result, "No directories with command history found") {
		t.Error("Expected view to show 'No directories' message")
	}
}

func TestFormatDirectoryTreeLine(t *testing.T) {
	model, _ := setupTestModel()
	model.width = 80

	item := DirectoryTreeItem{
		DirectoryIndex: history.DirectoryIndex{
			Path:         "/home/user/project",
			CommandCount: 15,
			LastUsed:     time.Now().Add(-time.Hour),
			IsActive:     true,
		},
		Level:      1,
		IsExpanded: true,
		Children:   []*DirectoryTreeItem{},
	}

	result := model.formatDirectoryTreeLine(item, false)

	// Should contain indentation for level 1
	if !strings.HasPrefix(result, "  ") {
		t.Error("Expected tree line to have indentation for level 1")
	}

	if !strings.Contains(result, "project") {
		t.Error("Expected tree line to contain directory name")
	}

	// Command count format varies based on count (e.g., "📈15" for counts > 10)
	if !strings.Contains(result, "15") {
		t.Error("Expected tree line to contain command count")
	}

	// Test with children (should show folder icon)
	// Children is a slice of pointers to DirectoryTreeItem, assign accordingly
	item.Children = []*DirectoryTreeItem{&DirectoryTreeItem{}}
	result = model.formatDirectoryTreeLine(item, false)

	if !strings.Contains(result, "📂") && !strings.Contains(result, "📁") {
		t.Error("Expected tree line with children to show folder icon")
	}
}

func TestOrganizeDirectoriesHierarchically(t *testing.T) {
	model, _ := setupTestModel()
	model.directories = []history.DirectoryIndex{
		{Path: "/home", CommandCount: 5},
		{Path: "/home/user", CommandCount: 10},
		{Path: "/home/user/project", CommandCount: 3},
		{Path: "/var/log", CommandCount: 2},
	}

	organized := model.organizeDirectoriesHierarchically()

	if len(organized) == 0 {
		t.Fatal("Expected organized directories, got empty list")
	}

	// Should have some hierarchical structure
	// The exact structure depends on the implementation, but we can verify basic properties
	foundRoot := false
	for _, item := range organized {
		if item.Level == 0 {
			foundRoot = true
		}
	}

	if !foundRoot {
		t.Error("Expected to find at least one root-level directory")
	}
}

// Test Edge Cases

func TestViewRenderingWithLongCommands(t *testing.T) {
	model, _ := setupTestModel()
	model.width = 40 // Narrow width to test truncation

	longCommand := strings.Repeat("very-long-command-name-", 10)
	cmd := createTestCommand("1", longCommand, "/test", history.Bash, 0)

	result := model.formatCommandLine(cmd, false)

	// Should be truncated to fit width
	if len(result) > model.width+20 { // Allow some margin for styling
		t.Error("Expected long command to be truncated")
	}

	if !strings.Contains(result, "...") {
		t.Error("Expected truncated command to contain ellipsis")
	}
}

func TestViewRenderingWithLongPaths(t *testing.T) {
	model, _ := setupTestModel()
	model.width = 50 // Narrow width

	longPath := "/very/long/path/that/should/be/truncated/in/the/display"
	dir := history.DirectoryIndex{
		Path:         longPath,
		CommandCount: 5,
		LastUsed:     time.Now(),
		IsActive:     true,
	}

	result := model.formatDirectoryLine(dir, false)

	if !strings.Contains(result, "...") {
		t.Error("Expected long path to be truncated with ellipsis")
	}

	// Should show the end of the path (more useful for navigation)
	if !strings.Contains(result, "display") {
		t.Error("Expected truncated path to show the end part")
	}
}
