package browser

import (
	"testing"
	"time"

	"github.com/ValGrace/command-history-tracker/pkg/history"
)

// MockStorage implements the StorageEngine interface for testing
type MockStorage struct {
	commands    []history.CommandRecord
	directories []string
	dirStats    []history.DirectoryIndex
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		commands:    []history.CommandRecord{},
		directories: []string{},
		dirStats:    []history.DirectoryIndex{},
	}
}

func (m *MockStorage) Initialize() error {
	return nil
}

func (m *MockStorage) SaveCommand(cmd history.CommandRecord) error {
	m.commands = append(m.commands, cmd)
	return nil
}

func (m *MockStorage) GetCommandsByDirectory(dir string) ([]history.CommandRecord, error) {
	var result []history.CommandRecord
	for _, cmd := range m.commands {
		if cmd.Directory == dir {
			result = append(result, cmd)
		}
	}
	return result, nil
}

func (m *MockStorage) GetDirectoriesWithHistory() ([]string, error) {
	return m.directories, nil
}

func (m *MockStorage) CleanupOldCommands(retentionDays int) error {
	return nil
}

func (m *MockStorage) SearchCommands(pattern string, dir string) ([]history.CommandRecord, error) {
	var result []history.CommandRecord
	for _, cmd := range m.commands {
		if (dir == "" || cmd.Directory == dir) &&
			(pattern == "" || cmd.Command == pattern) {
			result = append(result, cmd)
		}
	}
	return result, nil
}

func (m *MockStorage) Close() error {
	return nil
}

func (m *MockStorage) GetDirectoryStats() ([]history.DirectoryIndex, error) {
	return m.dirStats, nil
}

// Test helper functions

func createTestCommand(id, command, directory string, shell history.ShellType, exitCode int) history.CommandRecord {
	return history.CommandRecord{
		ID:        id,
		Command:   command,
		Directory: directory,
		Timestamp: time.Now(),
		Shell:     shell,
		ExitCode:  exitCode,
		Duration:  time.Second,
		Tags:      []string{},
	}
}

func setupTestModel() (*UIModel, *MockStorage) {
	storage := NewMockStorage()

	// Add test commands
	storage.commands = []history.CommandRecord{
		createTestCommand("1", "ls -la", "/home/user", history.Bash, 0),
		createTestCommand("2", "git status", "/home/user/project", history.Bash, 0),
		createTestCommand("3", "npm install", "/home/user/project", history.Bash, 0),
		createTestCommand("4", "Get-ChildItem", "/home/user", history.PowerShell, 0),
		createTestCommand("5", "failed-command", "/home/user", history.Bash, 1),
	}

	// Add test directories
	storage.directories = []string{"/home/user", "/home/user/project"}

	// Add directory stats
	storage.dirStats = []history.DirectoryIndex{
		{
			Path:         "/home/user",
			CommandCount: 3,
			LastUsed:     time.Now().Add(-time.Hour),
			IsActive:     true,
		},
		{
			Path:         "/home/user/project",
			CommandCount: 2,
			LastUsed:     time.Now().Add(-time.Minute),
			IsActive:     true,
		},
	}

	model := NewUIModel(storage, "/home/user")
	return model, storage
}

// Test UI Model Creation

func TestNewUIModel(t *testing.T) {
	storage := NewMockStorage()
	model := NewUIModel(storage, "/home/user")

	if model.currentDir != "/home/user" {
		t.Errorf("Expected currentDir to be '/home/user', got '%s'", model.currentDir)
	}

	if model.viewMode != DirectoryHistoryView {
		t.Errorf("Expected viewMode to be DirectoryHistoryView, got %v", model.viewMode)
	}

	if model.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be 0, got %d", model.selectedIndex)
	}

	if model.searchMode {
		t.Error("Expected searchMode to be false")
	}

	if model.showFilters {
		t.Error("Expected showFilters to be false")
	}
}

// Test Navigation Functions

func TestMoveUp(t *testing.T) {
	model, _ := setupTestModel()
	model.filteredCmds = []history.CommandRecord{
		createTestCommand("1", "cmd1", "/test", history.Bash, 0),
		createTestCommand("2", "cmd2", "/test", history.Bash, 0),
	}
	model.selectedIndex = 1

	updatedModel := model.moveUp()

	if updatedModel.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be 0, got %d", updatedModel.selectedIndex)
	}

	// Test boundary condition
	updatedModel = updatedModel.moveUp()
	if updatedModel.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to stay at 0, got %d", updatedModel.selectedIndex)
	}
}

func TestMoveDown(t *testing.T) {
	model, _ := setupTestModel()
	model.filteredCmds = []history.CommandRecord{
		createTestCommand("1", "cmd1", "/test", history.Bash, 0),
		createTestCommand("2", "cmd2", "/test", history.Bash, 0),
	}
	model.selectedIndex = 0

	updatedModel := model.moveDown()

	if updatedModel.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex to be 1, got %d", updatedModel.selectedIndex)
	}

	// Test boundary condition
	updatedModel = updatedModel.moveDown()
	if updatedModel.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex to stay at 1, got %d", updatedModel.selectedIndex)
	}
}

func TestGetMaxIndex(t *testing.T) {
	model, _ := setupTestModel()

	// Test with empty commands
	model.filteredCmds = []history.CommandRecord{}
	maxIndex := model.getMaxIndex()
	if maxIndex != 0 {
		t.Errorf("Expected maxIndex to be 0 for empty commands, got %d", maxIndex)
	}

	// Test with commands
	model.filteredCmds = []history.CommandRecord{
		createTestCommand("1", "cmd1", "/test", history.Bash, 0),
		createTestCommand("2", "cmd2", "/test", history.Bash, 0),
	}
	maxIndex = model.getMaxIndex()
	if maxIndex != 1 {
		t.Errorf("Expected maxIndex to be 1, got %d", maxIndex)
	}

	// Test directory tree view
	model.viewMode = DirectoryTreeView
	model.directories = []history.DirectoryIndex{
		{Path: "/dir1", CommandCount: 5},
		{Path: "/dir2", CommandCount: 3},
	}
	// Need to organize directories into tree structure
	model.directoryTree = model.organizeDirectoriesHierarchically()
	maxIndex = model.getMaxIndex()
	if maxIndex != 1 {
		t.Errorf("Expected maxIndex to be 1 for directory view, got %d", maxIndex)
	}
}

// Test Filtering Functions

func TestApplyFilters_UI(t *testing.T) {
	model, _ := setupTestModel()
	model.commands = []history.CommandRecord{
		createTestCommand("1", "ls -la", "/home/user", history.Bash, 0),
		createTestCommand("2", "git status", "/home/user", history.Bash, 0),
		createTestCommand("3", "Get-ChildItem", "/home/user", history.PowerShell, 0),
	}

	// Test text filter
	model.searchQuery = "git"
	updatedModel := model.applyFilters()

	if len(updatedModel.filteredCmds) != 1 {
		t.Errorf("Expected 1 filtered command, got %d", len(updatedModel.filteredCmds))
	}

	if updatedModel.filteredCmds[0].Command != "git status" {
		t.Errorf("Expected filtered command to be 'git status', got '%s'", updatedModel.filteredCmds[0].Command)
	}
}

func TestShellFilter(t *testing.T) {
	model, _ := setupTestModel()
	model.commands = []history.CommandRecord{
		createTestCommand("1", "ls -la", "/home/user", history.Bash, 0),
		createTestCommand("2", "Get-ChildItem", "/home/user", history.PowerShell, 0),
		createTestCommand("3", "dir", "/home/user", history.Cmd, 0),
	}

	// Test PowerShell filter
	model.shellFilter = history.PowerShell
	updatedModel := model.applyFilters()

	if len(updatedModel.filteredCmds) != 1 {
		t.Errorf("Expected 1 PowerShell command, got %d", len(updatedModel.filteredCmds))
	}

	if updatedModel.filteredCmds[0].Shell != history.PowerShell {
		t.Errorf("Expected PowerShell command, got %v", updatedModel.filteredCmds[0].Shell)
	}
}

func TestDateFilter(t *testing.T) {
	model, _ := setupTestModel()

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	model.commands = []history.CommandRecord{
		{
			ID:        "1",
			Command:   "recent-command",
			Directory: "/test",
			Timestamp: now,
			Shell:     history.Bash,
		},
		{
			ID:        "2",
			Command:   "old-command",
			Directory: "/test",
			Timestamp: yesterday,
			Shell:     history.Bash,
		},
	}

	// Test today filter
	model.dateFilter.Enabled = true
	model.dateFilter.Preset = Today
	updatedModel := model.applyDateFilter()

	if len(updatedModel.filteredCmds) != 1 {
		t.Errorf("Expected 1 command from today, got %d", len(updatedModel.filteredCmds))
	}

	if updatedModel.filteredCmds[0].Command != "recent-command" {
		t.Errorf("Expected 'recent-command', got '%s'", updatedModel.filteredCmds[0].Command)
	}
}

func TestClearFilters_UI(t *testing.T) {
	model, _ := setupTestModel()
	model.searchQuery = "test"
	model.dateFilter.Enabled = true
	model.shellFilter = history.PowerShell
	model.filterMode = CombinedFilter

	updatedModel := model.clearFilters()

	if updatedModel.searchQuery != "" {
		t.Errorf("Expected empty search query, got '%s'", updatedModel.searchQuery)
	}

	if updatedModel.dateFilter.Enabled {
		t.Error("Expected date filter to be disabled")
	}

	if updatedModel.shellFilter != history.Unknown {
		t.Errorf("Expected shell filter to be Unknown, got %v", updatedModel.shellFilter)
	}

	if updatedModel.filterMode != NoFilter {
		t.Errorf("Expected filter mode to be NoFilter, got %v", updatedModel.filterMode)
	}
}

// Test Command Selection

func TestGetSelectedCommand(t *testing.T) {
	model, _ := setupTestModel()
	model.viewMode = DirectoryHistoryView
	model.filteredCmds = []history.CommandRecord{
		createTestCommand("1", "first-command", "/test", history.Bash, 0),
		createTestCommand("2", "second-command", "/test", history.Bash, 0),
	}
	model.selectedIndex = 1

	selectedCmd := model.GetSelectedCommand()

	if selectedCmd == nil {
		t.Fatal("Expected selected command, got nil")
	}

	if selectedCmd.Command != "second-command" {
		t.Errorf("Expected 'second-command', got '%s'", selectedCmd.Command)
	}

	// Test with no commands
	model.filteredCmds = []history.CommandRecord{}
	selectedCmd = model.GetSelectedCommand()

	if selectedCmd != nil {
		t.Error("Expected nil for empty command list")
	}

	// Test with tree view
	model.viewMode = DirectoryTreeView
	selectedCmd = model.GetSelectedCommand()

	if selectedCmd != nil {
		t.Error("Expected nil for tree view mode")
	}
}

// Test Breadcrumb Functions

func TestBuildBreadcrumbs(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{"."}},
		{".", []string{"."}},
		{"/", []string{"/"}},
		{"/home", []string{"/", "/home"}},
		{"/home/user", []string{"/", "/home", "/home/user"}},
		{"C:\\Users\\test", []string{"C:", "C:/Users", "C:/Users/test"}},
	}

	for _, test := range tests {
		result := buildBreadcrumbs(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("For input '%s', expected %d breadcrumbs, got %d", test.input, len(test.expected), len(result))
			continue
		}

		for i, expected := range test.expected {
			if result[i] != expected {
				t.Errorf("For input '%s', expected breadcrumb[%d] to be '%s', got '%s'", test.input, i, expected, result[i])
			}
		}
	}
}

func TestGetParentDirectory(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{".", ""},
		{"/", ""},
		{"/home", "/"},
		{"/home/user", "/home"},
		{"/home/user/project", "/home/user"},
		{"C:\\Users\\test", "C:/Users"},
	}

	for _, test := range tests {
		result := getParentDirectory(test.input)
		if result != test.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'", test.input, test.expected, result)
		}
	}
}

// Test Shell Filter Cycling

func TestGetNextShellFilter(t *testing.T) {
	model, _ := setupTestModel()

	// Test cycling through shell filters
	model.shellFilter = history.Unknown
	next := model.getNextShellFilter()
	if next != history.PowerShell {
		t.Errorf("Expected PowerShell after Unknown, got %v", next)
	}

	model.shellFilter = history.PowerShell
	next = model.getNextShellFilter()
	if next != history.Bash {
		t.Errorf("Expected Bash after PowerShell, got %v", next)
	}

	model.shellFilter = history.Cmd
	next = model.getNextShellFilter()
	if next != history.Unknown {
		t.Errorf("Expected Unknown after Cmd, got %v", next)
	}
}
