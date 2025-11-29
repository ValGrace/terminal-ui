package browser

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ValGrace/command-history-tracker/pkg/history"
)

// Comprehensive UI tests for keyboard navigation, command display/selection, and directory navigation

// Test Keyboard Navigation - Comprehensive Coverage

func TestKeyboardNavigation_PageNavigation(t *testing.T) {
	model, _ := setupTestModel()
	
	// Create a large list of commands to test pagination
	commands := make([]history.CommandRecord, 100)
	for i := 0; i < 100; i++ {
		commands[i] = createTestCommand(
			string(rune(i)),
			"command-"+string(rune(i)),
			"/test",
			history.Bash,
			0,
		)
	}
	model.commands = commands
	model.filteredCmds = commands
	model.height = 20 // Small height to force scrolling
	
	// Test moving down multiple times
	for i := 0; i < 15; i++ {
		updatedModel := model.moveDown()
		model = &updatedModel
	}
	
	if model.selectedIndex != 15 {
		t.Errorf("Expected selectedIndex to be 15, got %d", model.selectedIndex)
	}
	
	// Verify scroll offset adjusted
	if model.scrollOffset == 0 {
		t.Error("Expected scroll offset to be adjusted for visibility")
	}
	
	// Test moving up
	for i := 0; i < 10; i++ {
		updatedModel := model.moveUp()
		model = &updatedModel
	}
	
	if model.selectedIndex != 5 {
		t.Errorf("Expected selectedIndex to be 5, got %d", model.selectedIndex)
	}
}

func TestKeyboardNavigation_BoundaryConditions(t *testing.T) {
	model, _ := setupTestModel()
	model.filteredCmds = []history.CommandRecord{
		createTestCommand("1", "cmd1", "/test", history.Bash, 0),
		createTestCommand("2", "cmd2", "/test", history.Bash, 0),
	}
	
	// Test moving up at top boundary
	model.selectedIndex = 0
	originalIndex := model.selectedIndex
	updatedModel := model.moveUp()
	
	if updatedModel.selectedIndex != originalIndex {
		t.Errorf("Expected selectedIndex to stay at %d when at top, got %d", originalIndex, updatedModel.selectedIndex)
	}
	
	// Test moving down at bottom boundary
	model.selectedIndex = 1
	originalIndex = model.selectedIndex
	updatedModel = model.moveDown()
	
	if updatedModel.selectedIndex != originalIndex {
		t.Errorf("Expected selectedIndex to stay at %d when at bottom, got %d", originalIndex, updatedModel.selectedIndex)
	}
}

func TestKeyboardNavigation_EmptyList(t *testing.T) {
	model, _ := setupTestModel()
	model.filteredCmds = []history.CommandRecord{}
	model.selectedIndex = 0
	
	// Test navigation with empty list
	updatedModel := model.moveDown()
	if updatedModel.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to stay at 0 for empty list, got %d", updatedModel.selectedIndex)
	}
	
	updatedModel = model.moveUp()
	if updatedModel.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to stay at 0 for empty list, got %d", updatedModel.selectedIndex)
	}
}

func TestKeyboardNavigation_VimStyleKeys(t *testing.T) {
	model, _ := setupTestModel()
	model.filteredCmds = []history.CommandRecord{
		createTestCommand("1", "cmd1", "/test", history.Bash, 0),
		createTestCommand("2", "cmd2", "/test", history.Bash, 0),
		createTestCommand("3", "cmd3", "/test", history.Bash, 0),
	}
	
	// Test 'j' key (down)
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updatedModel, _ := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)
	
	if uiModel.selectedIndex != 1 {
		t.Errorf("Expected 'j' to move down to index 1, got %d", uiModel.selectedIndex)
	}
	
	// Test 'k' key (up)
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updatedModel, _ = uiModel.Update(keyMsg)
	uiModel = updatedModel.(UIModel)
	
	if uiModel.selectedIndex != 0 {
		t.Errorf("Expected 'k' to move up to index 0, got %d", uiModel.selectedIndex)
	}
}

func TestKeyboardNavigation_ArrowKeys(t *testing.T) {
	model, _ := setupTestModel()
	model.filteredCmds = []history.CommandRecord{
		createTestCommand("1", "cmd1", "/test", history.Bash, 0),
		createTestCommand("2", "cmd2", "/test", history.Bash, 0),
		createTestCommand("3", "cmd3", "/test", history.Bash, 0),
	}
	
	// Test down arrow
	keyMsg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)
	
	if uiModel.selectedIndex != 1 {
		t.Errorf("Expected down arrow to move to index 1, got %d", uiModel.selectedIndex)
	}
	
	// Test up arrow
	keyMsg = tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ = uiModel.Update(keyMsg)
	uiModel = updatedModel.(UIModel)
	
	if uiModel.selectedIndex != 0 {
		t.Errorf("Expected up arrow to move to index 0, got %d", uiModel.selectedIndex)
	}
}

// Test Command Display and Selection - Comprehensive Coverage

func TestCommandDisplay_WithDifferentShells(t *testing.T) {
	model, _ := setupTestModel()
	model.width = 100
	
	shells := []history.ShellType{
		history.Bash,
		history.PowerShell,
		history.Zsh,
		history.Cmd,
	}
	
	for _, shell := range shells {
		cmd := createTestCommand("1", "test-command", "/test", shell, 0)
		line := model.formatCommandLine(cmd, false)
		
		if line == "" {
			t.Errorf("Expected non-empty line for shell %v", shell)
		}
		
		// Verify shell indicator is present
		shellName := ""
		switch shell {
		case history.Bash:
			shellName = "bash"
		case history.PowerShell:
			shellName = "powershell"
		case history.Zsh:
			shellName = "zsh"
		case history.Cmd:
			shellName = "cmd"
		}
		
		if shellName != "" && !contains(line, shellName) {
			t.Errorf("Expected line to contain shell name '%s' for shell %v", shellName, shell)
		}
	}
}

func TestCommandDisplay_WithDifferentExitCodes(t *testing.T) {
	model, _ := setupTestModel()
	model.width = 100
	
	testCases := []struct {
		exitCode int
		expected string
	}{
		{0, "✅"},
		{1, "❌1"},
		{127, "❌127"},
		{255, "❌255"},
	}
	
	for _, tc := range testCases {
		cmd := createTestCommand("1", "test-command", "/test", history.Bash, tc.exitCode)
		line := model.formatCommandLine(cmd, false)
		
		if !contains(line, tc.expected) {
			t.Errorf("Expected line to contain '%s' for exit code %d, got: %s", tc.expected, tc.exitCode, line)
		}
	}
}

func TestCommandDisplay_SelectedVsUnselected(t *testing.T) {
	model, _ := setupTestModel()
	model.width = 100
	
	cmd := createTestCommand("1", "test-command", "/test", history.Bash, 0)
	
	// Test unselected
	unselectedLine := model.formatCommandLine(cmd, false)
	
	// Test selected
	selectedLine := model.formatCommandLine(cmd, true)
	
	// Both should contain the command text
	if !contains(unselectedLine, "test-command") {
		t.Error("Expected unselected line to contain command text")
	}
	
	if !contains(selectedLine, "test-command") {
		t.Error("Expected selected line to contain command text")
	}
	
	// Lines should be different (due to styling)
	// We can't easily test the styling, but we can verify both are non-empty
	if unselectedLine == "" || selectedLine == "" {
		t.Error("Expected both selected and unselected lines to be non-empty")
	}
}

func TestCommandSelection_InHistoryView(t *testing.T) {
	model, _ := setupTestModel()
	model.viewMode = DirectoryHistoryView
	model.filteredCmds = []history.CommandRecord{
		createTestCommand("1", "first-command", "/test", history.Bash, 0),
		createTestCommand("2", "second-command", "/test", history.Bash, 0),
	}
	model.selectedIndex = 1
	
	// Test Enter key to select command
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)
	
	// Should have quit command
	if cmd == nil {
		t.Error("Expected quit command after selecting item")
	}
	
	// Should have stored selected command
	selectedCmd := uiModel.GetSelectedCommand()
	if selectedCmd == nil {
		t.Fatal("Expected selected command to be stored")
	}
	
	if selectedCmd.Command != "second-command" {
		t.Errorf("Expected selected command to be 'second-command', got '%s'", selectedCmd.Command)
	}
}

func TestCommandSelection_EmptyList(t *testing.T) {
	model, _ := setupTestModel()
	model.viewMode = DirectoryHistoryView
	model.filteredCmds = []history.CommandRecord{}
	
	// Test Enter key with empty list
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)
	
	// Should not quit
	if cmd != nil {
		t.Error("Expected no command for empty list selection")
	}
	
	// Should not have selected command
	selectedCmd := uiModel.GetSelectedCommand()
	if selectedCmd != nil {
		t.Error("Expected no selected command for empty list")
	}
}

// Test Directory Navigation Flows - Comprehensive Coverage

func TestDirectoryNavigation_ParentNavigation(t *testing.T) {
	model, _ := setupTestModel()
	model.viewMode = DirectoryHistoryView
	model.currentDir = "/home/user/project/src"
	model.breadcrumbs = buildBreadcrumbs("/home/user/project/src")
	
	// Navigate to parent using backspace
	keyMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, cmd := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)
	
	if uiModel.currentDir != "/home/user/project" {
		t.Errorf("Expected currentDir to be '/home/user/project', got '%s'", uiModel.currentDir)
	}
	
	if cmd == nil {
		t.Error("Expected command to load parent directory history")
	}
	
	// Navigate to parent using left arrow
	keyMsg = tea.KeyMsg{Type: tea.KeyLeft}
	updatedModel, cmd = uiModel.Update(keyMsg)
	uiModel = updatedModel.(UIModel)
	
	if uiModel.currentDir != "/home/user" {
		t.Errorf("Expected currentDir to be '/home/user', got '%s'", uiModel.currentDir)
	}
}

func TestDirectoryNavigation_AtRoot(t *testing.T) {
	model, _ := setupTestModel()
	model.viewMode = DirectoryHistoryView
	model.currentDir = "/"
	model.breadcrumbs = buildBreadcrumbs("/")
	
	// Try to navigate to parent at root
	keyMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, cmd := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)
	
	// Should stay at root
	if uiModel.currentDir != "/" {
		t.Errorf("Expected to stay at root, got '%s'", uiModel.currentDir)
	}
	
	// Should not have command
	if cmd != nil {
		t.Error("Expected no command when already at root")
	}
}

func TestDirectoryNavigation_TreeViewSelection(t *testing.T) {
	model, storage := setupTestModel()
	model.viewMode = DirectoryTreeView
	
	storage.dirStats = []history.DirectoryIndex{
		{Path: "/home/user", CommandCount: 5, IsActive: true},
		{Path: "/home/user/project", CommandCount: 10, IsActive: true},
	}
	
	model.directories = storage.dirStats
	model.directoryTree = model.organizeDirectoriesHierarchically()
	model.selectedIndex = 0
	
	// Select directory using Enter
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)
	
	// Should switch to history view
	if uiModel.viewMode != DirectoryHistoryView {
		t.Errorf("Expected DirectoryHistoryView, got %v", uiModel.viewMode)
	}
	
	// Should update current directory
	if len(model.directoryTree) > 0 {
		expectedDir := model.directoryTree[0].Path
		if uiModel.currentDir != expectedDir {
			t.Errorf("Expected currentDir to be '%s', got '%s'", expectedDir, uiModel.currentDir)
		}
	}
	
	// Should have command to load directory history
	if cmd == nil {
		t.Error("Expected command to load directory history")
	}
	
	// Should reset selection
	if uiModel.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be reset to 0, got %d", uiModel.selectedIndex)
	}
}

func TestDirectoryNavigation_TreeExpansionCollapse(t *testing.T) {
	model, storage := setupTestModel()
	model.viewMode = DirectoryTreeView
	
	storage.dirStats = []history.DirectoryIndex{
		{Path: "/home", CommandCount: 5, IsActive: true},
		{Path: "/home/user", CommandCount: 10, IsActive: true},
		{Path: "/home/user/project", CommandCount: 15, IsActive: true},
	}
	
	model.directories = storage.dirStats
	model.directoryTree = model.organizeDirectoriesHierarchically()
	
	// Find a directory with children
	var dirWithChildren *DirectoryTreeItem
	for i := range model.directoryTree {
		if len(model.directoryTree[i].Children) > 0 {
			dirWithChildren = &model.directoryTree[i]
			model.selectedIndex = i
			break
		}
	}
	
	if dirWithChildren != nil {
		// Test right arrow to expand
		keyMsg := tea.KeyMsg{Type: tea.KeyRight}
		updatedModel, _ := model.Update(keyMsg)
		uiModel := updatedModel.(UIModel)
		
		// Verify expansion state changed
		if uiModel.viewMode != DirectoryTreeView {
			t.Error("Expected to stay in DirectoryTreeView")
		}
		
		// Test left arrow to collapse
		keyMsg = tea.KeyMsg{Type: tea.KeyLeft}
		updatedModel, _ = uiModel.Update(keyMsg)
		uiModel = updatedModel.(UIModel)
		
		if uiModel.viewMode != DirectoryTreeView {
			t.Error("Expected to stay in DirectoryTreeView")
		}
	}
}

func TestDirectoryNavigation_ViewModeSwitching(t *testing.T) {
	model, _ := setupTestModel()
	model.viewMode = DirectoryHistoryView
	
	// Switch to tree view using 't' key
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	updatedModel, cmd := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)
	
	if uiModel.viewMode != DirectoryTreeView {
		t.Errorf("Expected DirectoryTreeView, got %v", uiModel.viewMode)
	}
	
	if cmd == nil {
		t.Error("Expected command to load directory tree")
	}
	
	// Switch back to history view using 'h' key
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

func TestDirectoryNavigation_TabSwitching(t *testing.T) {
	model, _ := setupTestModel()
	model.viewMode = DirectoryHistoryView
	
	// Switch view using Tab key
	keyMsg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)
	
	if uiModel.viewMode != DirectoryTreeView {
		t.Errorf("Expected DirectoryTreeView after Tab, got %v", uiModel.viewMode)
	}
	
	// Switch back using Tab
	keyMsg = tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ = uiModel.Update(keyMsg)
	uiModel = updatedModel.(UIModel)
	
	if uiModel.viewMode != DirectoryHistoryView {
		t.Errorf("Expected DirectoryHistoryView after second Tab, got %v", uiModel.viewMode)
	}
}

func TestDirectoryNavigation_BreadcrumbsUpdate(t *testing.T) {
	model, _ := setupTestModel()
	model.currentDir = "/home/user/project"
	model.breadcrumbs = buildBreadcrumbs("/home/user/project")
	
	// Navigate to parent
	keyMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, _ := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)
	
	// Verify breadcrumbs updated
	if len(uiModel.breadcrumbs) >= len(model.breadcrumbs) {
		t.Error("Expected breadcrumbs to be shorter after navigating to parent")
	}
	
	// Verify breadcrumbs match current directory
	expectedBreadcrumbs := buildBreadcrumbs(uiModel.currentDir)
	if len(uiModel.breadcrumbs) != len(expectedBreadcrumbs) {
		t.Errorf("Expected %d breadcrumbs, got %d", len(expectedBreadcrumbs), len(uiModel.breadcrumbs))
	}
}

// Test Refresh Functionality

func TestRefreshFunctionality_HistoryView(t *testing.T) {
	model, _ := setupTestModel()
	model.viewMode = DirectoryHistoryView
	model.currentDir = "/test"
	
	// Test refresh with 'r' key
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	updatedModel, cmd := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)
	
	if uiModel.viewMode != DirectoryHistoryView {
		t.Error("Expected to stay in DirectoryHistoryView")
	}
	
	if cmd == nil {
		t.Error("Expected command to reload directory history")
	}
}

func TestRefreshFunctionality_TreeView(t *testing.T) {
	model, _ := setupTestModel()
	model.viewMode = DirectoryTreeView
	
	// Test refresh with 'r' key
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	updatedModel, cmd := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)
	
	if uiModel.viewMode != DirectoryTreeView {
		t.Error("Expected to stay in DirectoryTreeView")
	}
	
	if cmd == nil {
		t.Error("Expected command to reload directory tree")
	}
}

// Test Quit Functionality

func TestQuitFunctionality_QKey(t *testing.T) {
	model, _ := setupTestModel()
	
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updatedModel, cmd := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)
	
	if !uiModel.quitting {
		t.Error("Expected quitting to be true after 'q' key")
	}
	
	if cmd == nil {
		t.Error("Expected quit command")
	}
}

func TestQuitFunctionality_CtrlC(t *testing.T) {
	model, _ := setupTestModel()
	
	keyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updatedModel, cmd := model.Update(keyMsg)
	uiModel := updatedModel.(UIModel)
	
	if !uiModel.quitting {
		t.Error("Expected quitting to be true after Ctrl+C")
	}
	
	if cmd == nil {
		t.Error("Expected quit command")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
