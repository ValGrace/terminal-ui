package browser

import (
	"strings"
	"testing"

	"github.com/ValGrace/command-history-tracker/pkg/history"
	tea "github.com/charmbracelet/bubbletea"
)

// Integration tests for browser functionality

func TestBrowserIntegration(t *testing.T) {
	storage := NewMockStorage()

	// Add test data
	storage.commands = []history.CommandRecord{
		createTestCommand("1", "ls -la", "/home/user", history.Bash, 0),
		createTestCommand("2", "git status", "/home/user/project", history.Bash, 0),
		createTestCommand("3", "npm install", "/home/user/project", history.Bash, 0),
	}

	storage.directories = []string{"/home/user", "/home/user/project"}

	browser := NewBrowser(storage)

	// Test setting current directory
	err := browser.SetCurrentDirectory("/home/user")
	if err != nil {
		t.Fatalf("Failed to set current directory: %v", err)
	}

	// Note: We can't easily test the interactive UI methods without a full terminal,
	// but we can test that they don't panic and return reasonable errors
}

func TestUIModelMessageHandling(t *testing.T) {
	model, storage := setupTestModel()

	// Test window size message
	windowMsg := tea.WindowSizeMsg{Width: 100, Height: 30}
	updatedModel, _ := model.Update(windowMsg)

	uiModel := updatedModel.(UIModel)
	if uiModel.width != 100 || uiModel.height != 30 {
		t.Errorf("Expected dimensions 100x30, got %dx%d", uiModel.width, uiModel.height)
	}

	// Test directory history message
	commands := []history.CommandRecord{
		createTestCommand("1", "test-command", "/test", history.Bash, 0),
	}
	historyMsg := directoryHistoryMsg{commands: commands}
	updatedModel, _ = model.Update(historyMsg)

	uiModel = updatedModel.(UIModel)
	if len(uiModel.commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(uiModel.commands))
	}

	// Test directory tree message
	directories := []history.DirectoryIndex{
		{Path: "/test", CommandCount: 5},
	}
	treeMsg := directoryTreeMsg{directories: directories}
	updatedModel, _ = model.Update(treeMsg)

	uiModel = updatedModel.(UIModel)
	if len(uiModel.directories) != 1 {
		t.Errorf("Expected 1 directory, got %d", len(uiModel.directories))
	}

	// Test error message
	testErr := &history.ValidationError{Field: "test", Message: "test error"}
	errorMsg := errorMsg{error: testErr}
	updatedModel, _ = model.Update(errorMsg)

	uiModel = updatedModel.(UIModel)
	if uiModel.error == nil {
		t.Error("Expected error to be set")
	}

	_ = storage // Avoid unused variable warning
}

func TestKeyboardNavigation(t *testing.T) {
	model, _ := setupTestModel()
	model.filteredCmds = []history.CommandRecord{
		createTestCommand("1", "cmd1", "/test", history.Bash, 0),
		createTestCommand("2", "cmd2", "/test", history.Bash, 0),
		createTestCommand("3", "cmd3", "/test", history.Bash, 0),
	}

	// Test up/down navigation
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updatedModel, _ := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)

	if uiModel.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex to be 1 after 'j', got %d", uiModel.selectedIndex)
	}

	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updatedModel, _ = uiModel.Update(keyMsg)
	uiModel = updatedModel.(UIModel)

	if uiModel.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be 0 after 'k', got %d", uiModel.selectedIndex)
	}

	// Test arrow keys
	keyMsg = tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = uiModel.Update(keyMsg)
	uiModel = updatedModel.(UIModel)

	if uiModel.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex to be 1 after down arrow, got %d", uiModel.selectedIndex)
	}

	keyMsg = tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ = uiModel.Update(keyMsg)
	uiModel = updatedModel.(UIModel)

	if uiModel.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be 0 after up arrow, got %d", uiModel.selectedIndex)
	}
}

func TestSearchFunctionality(t *testing.T) {
	model, _ := setupTestModel()
	model.commands = []history.CommandRecord{
		createTestCommand("1", "git status", "/test", history.Bash, 0),
		createTestCommand("2", "git commit", "/test", history.Bash, 0),
		createTestCommand("3", "ls -la", "/test", history.Bash, 0),
	}
	model.filteredCmds = model.commands

	// Enter search mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updatedModel, _ := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)

	if !uiModel.searchMode {
		t.Error("Expected search mode to be enabled")
	}

	// Type search query
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	updatedModel, _ = uiModel.Update(keyMsg)
	uiModel = updatedModel.(UIModel)

	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	updatedModel, _ = uiModel.Update(keyMsg)
	uiModel = updatedModel.(UIModel)

	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	updatedModel, _ = uiModel.Update(keyMsg)
	uiModel = updatedModel.(UIModel)

	if uiModel.searchQuery != "git" {
		t.Errorf("Expected search query to be 'git', got '%s'", uiModel.searchQuery)
	}

	// Apply search
	keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = uiModel.Update(keyMsg)
	uiModel = updatedModel.(UIModel)

	if len(uiModel.filteredCmds) != 2 {
		t.Errorf("Expected 2 filtered commands, got %d", len(uiModel.filteredCmds))
	}

	// Exit search mode
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = uiModel.Update(keyMsg)
	uiModel = updatedModel.(UIModel)

	if uiModel.searchMode {
		t.Error("Expected search mode to be disabled")
	}
}

func TestViewModeSwitching(t *testing.T) {
	model, _ := setupTestModel()

	// Switch to tree view
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	updatedModel, cmd := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)

	if uiModel.viewMode != DirectoryTreeView {
		t.Errorf("Expected DirectoryTreeView, got %v", uiModel.viewMode)
	}

	if cmd == nil {
		t.Error("Expected command to load directory tree")
	}

	// Switch back to history view
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	updatedModel, cmd = uiModel.Update(keyMsg)
	uiModel = updatedModel.(UIModel)

	if uiModel.viewMode != DirectoryHistoryView {
		t.Errorf("Expected DirectoryHistoryView, got %v", uiModel.viewMode)
	}

	if cmd == nil {
		t.Error("Expected command to load directory history")
	}
}

func TestFilterToggling(t *testing.T) {
	model, _ := setupTestModel()

	// Toggle filter panel
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}
	updatedModel, _ := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)

	if !uiModel.showFilters {
		t.Error("Expected filters to be shown")
	}

	// Toggle date filter
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	updatedModel, _ = uiModel.Update(keyMsg)
	uiModel = updatedModel.(UIModel)

	if !uiModel.dateFilter.Enabled {
		t.Error("Expected date filter to be enabled")
	}

	// Toggle shell filter
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	updatedModel, _ = uiModel.Update(keyMsg)
	uiModel = updatedModel.(UIModel)

	if uiModel.shellFilter == history.Unknown {
		t.Error("Expected shell filter to be set")
	}
}

func TestPreviewToggling(t *testing.T) {
	model, _ := setupTestModel()
	model.filteredCmds = []history.CommandRecord{
		createTestCommand("1", "test-command", "/test", history.Bash, 0),
	}

	// Toggle preview
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	updatedModel, _ := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)

	if !uiModel.showPreview {
		t.Error("Expected preview to be shown")
	}

	// Toggle preview off
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	updatedModel, _ = uiModel.Update(keyMsg)
	uiModel = updatedModel.(UIModel)

	if uiModel.showPreview {
		t.Error("Expected preview to be hidden")
	}
}

func TestDirectoryNavigation(t *testing.T) {
	model, _ := setupTestModel()
	model.currentDir = "/home/user/project"
	model.breadcrumbs = buildBreadcrumbs(model.currentDir)

	// Navigate to parent directory
	keyMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, cmd := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)

	if uiModel.currentDir != "/home/user" {
		t.Errorf("Expected current directory to be '/home/user', got '%s'", uiModel.currentDir)
	}

	if cmd == nil {
		t.Error("Expected command to load parent directory history")
	}

	// Test left arrow key (same as backspace)
	model.currentDir = "/home/user/project"
	keyMsg = tea.KeyMsg{Type: tea.KeyLeft}
	updatedModel, cmd = model.Update(keyMsg)
	uiModel = updatedModel.(UIModel)

	if uiModel.currentDir != "/home/user" {
		t.Errorf("Expected current directory to be '/home/user', got '%s'", uiModel.currentDir)
	}
}

func TestQuitFunctionality(t *testing.T) {
	model, _ := setupTestModel()

	// Test 'q' key
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updatedModel, cmd := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)

	if !uiModel.quitting {
		t.Error("Expected quitting to be true")
	}

	if cmd == nil {
		t.Error("Expected quit command")
	}

	// Test Ctrl+C
	model.quitting = false
	keyMsg = tea.KeyMsg{Type: tea.KeyCtrlC}
	updatedModel, cmd = model.Update(keyMsg)
	uiModel = updatedModel.(UIModel)

	if !uiModel.quitting {
		t.Error("Expected quitting to be true for Ctrl+C")
	}
}

// Test error handling

func TestErrorHandling(t *testing.T) {
	model, _ := setupTestModel()

	// Simulate an error
	testError := &history.ValidationError{Field: "test", Message: "test error"}
	errorMsg := errorMsg{error: testError}

	updatedModel, _ := model.Update(errorMsg)
	uiModel := updatedModel.(UIModel)

	if uiModel.error == nil {
		t.Error("Expected error to be set")
	}

	// Test view rendering with error
	view := uiModel.View()
	if !strings.Contains(view, "Error:") {
		t.Error("Expected error view to contain 'Error:'")
	}

	if !strings.Contains(view, "test error") {
		t.Error("Expected error view to contain error message")
	}
}
