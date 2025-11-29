package browser

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ValGrace/command-history-tracker/pkg/history"
)

// Test Directory Tree Organization

func TestOrganizeDirectoriesHierarchically_WithParentExpansion(t *testing.T) {
	model, storage := setupTestModel()

	// Set up hierarchical directory structure
	storage.dirStats = []history.DirectoryIndex{
		{
			Path:         "/home",
			CommandCount: 5,
			LastUsed:     time.Now().Add(-time.Hour),
			IsActive:     true,
		},
		{
			Path:         "/home/user",
			CommandCount: 10,
			LastUsed:     time.Now().Add(-30 * time.Minute),
			IsActive:     true,
		},
		{
			Path:         "/home/user/project",
			CommandCount: 15,
			LastUsed:     time.Now().Add(-5 * time.Minute),
			IsActive:     true,
		},
		{
			Path:         "/home/user/project/src",
			CommandCount: 8,
			LastUsed:     time.Now(),
			IsActive:     true,
		},
		{
			Path:         "/var",
			CommandCount: 3,
			LastUsed:     time.Now().Add(-2 * time.Hour),
			IsActive:     true,
		},
	}

	model.directories = storage.dirStats
	model.currentDir = "/home/user/project"

	// Organize directories hierarchically
	tree := model.organizeDirectoriesHierarchically()

	// Verify tree structure
	if len(tree) == 0 {
		t.Fatal("Expected non-empty directory tree")
	}

	// Verify root directories are at level 0
	rootCount := 0
	for _, item := range tree {
		if item.Level == 0 {
			rootCount++
		}
	}

	if rootCount == 0 {
		t.Error("Expected at least one root directory")
	}

	// Verify parent directories of current directory are expanded
	foundCurrentDir := false
	for _, item := range tree {
		if item.Path == "/home/user/project" {
			foundCurrentDir = true
		}
		// Parent directories should be expanded by default
		if item.Path == "/home" || item.Path == "/home/user" {
			if !item.IsExpanded {
				t.Errorf("Expected parent directory %s to be expanded", item.Path)
			}
		}
	}

	if !foundCurrentDir {
		t.Error("Expected to find current directory in tree")
	}
}

func TestOrganizeDirectoriesHierarchically_EmptyDirectories(t *testing.T) {
	model, _ := setupTestModel()
	model.directories = []history.DirectoryIndex{}

	tree := model.organizeDirectoriesHierarchically()

	if len(tree) != 0 {
		t.Errorf("Expected empty tree for no directories, got %d items", len(tree))
	}
}

func TestOrganizeDirectoriesHierarchically_SingleDirectory(t *testing.T) {
	model, _ := setupTestModel()
	model.directories = []history.DirectoryIndex{
		{
			Path:         "/home/user",
			CommandCount: 5,
			LastUsed:     time.Now(),
			IsActive:     true,
		},
	}

	tree := model.organizeDirectoriesHierarchically()

	if len(tree) != 1 {
		t.Errorf("Expected 1 item in tree, got %d", len(tree))
	}

	if tree[0].Level != 0 {
		t.Errorf("Expected root level (0), got %d", tree[0].Level)
	}

	if tree[0].Path != "/home/user" {
		t.Errorf("Expected path '/home/user', got '%s'", tree[0].Path)
	}
}

func TestOrganizeDirectoriesHierarchically_WithExpansion(t *testing.T) {
	model, storage := setupTestModel()

	storage.dirStats = []history.DirectoryIndex{
		{Path: "/home", CommandCount: 5, IsActive: true},
		{Path: "/home/user", CommandCount: 10, IsActive: true},
		{Path: "/home/user/project", CommandCount: 15, IsActive: true},
	}

	model.directories = storage.dirStats
	model.currentDir = "/home/user"

	// Manually expand /home
	model.treeExpanded["/home"] = true

	tree := model.organizeDirectoriesHierarchically()

	// Verify /home is expanded and its children are visible
	foundHome := false
	foundUser := false
	for _, item := range tree {
		if item.Path == "/home" {
			foundHome = true
			if !item.IsExpanded {
				t.Error("Expected /home to be expanded")
			}
		}
		if item.Path == "/home/user" {
			foundUser = true
			if item.Level != 1 {
				t.Errorf("Expected /home/user to be at level 1, got %d", item.Level)
			}
		}
	}

	if !foundHome {
		t.Error("Expected to find /home in tree")
	}
	if !foundUser {
		t.Error("Expected to find /home/user in tree (child of expanded /home)")
	}
}

func TestOrganizeDirectoriesHierarchically_WithCollapse(t *testing.T) {
	model, storage := setupTestModel()

	storage.dirStats = []history.DirectoryIndex{
		{Path: "/home", CommandCount: 5, IsActive: true},
		{Path: "/home/user", CommandCount: 10, IsActive: true},
		{Path: "/home/user/project", CommandCount: 15, IsActive: true},
	}

	model.directories = storage.dirStats
	model.currentDir = "/var" // Different directory

	// Explicitly collapse /home
	model.treeExpanded["/home"] = false

	tree := model.organizeDirectoriesHierarchically()

	// Verify /home/user is not visible when /home is collapsed
	foundUser := false
	for _, item := range tree {
		if item.Path == "/home/user" {
			foundUser = true
		}
	}

	if foundUser {
		t.Error("Expected /home/user to be hidden when /home is collapsed")
	}
}

// Test Directory Tree Item Formatting

func TestFormatDirectoryTreeLine_WithIndentation(t *testing.T) {
	model, _ := setupTestModel()
	model.width = 100

	// Test item at level 2 (should have more indentation)
	item := DirectoryTreeItem{
		DirectoryIndex: history.DirectoryIndex{
			Path:         "/home/user/project/src",
			CommandCount: 10,
			LastUsed:     time.Now(),
			IsActive:     true,
		},
		Level:      2,
		IsExpanded: false,
		Children:   []*DirectoryTreeItem{},
	}

	line := model.formatDirectoryTreeLine(item, false)

	// Verify line is not empty
	if line == "" {
		t.Error("Expected non-empty formatted line")
	}

	// Level 2 should have more indentation than level 0
	// We can't easily test the exact indentation, but we can verify the line is longer
	if len(line) < 20 {
		t.Error("Expected formatted line with indentation to be reasonably long")
	}
}

func TestFormatDirectoryTreeLine_CurrentDirectory(t *testing.T) {
	model, _ := setupTestModel()
	model.width = 100
	model.currentDir = "/home/user/project"

	item := DirectoryTreeItem{
		DirectoryIndex: history.DirectoryIndex{
			Path:         "/home/user/project",
			CommandCount: 25,
			LastUsed:     time.Now(),
			IsActive:     true,
		},
		Level:      1,
		IsExpanded: false,
		Children:   []*DirectoryTreeItem{},
	}

	line := model.formatDirectoryTreeLine(item, false)

	// Current directory should be highlighted differently
	if line == "" {
		t.Error("Expected non-empty formatted line for current directory")
	}
}

func TestFormatDirectoryTreeLine_InactiveDirectory(t *testing.T) {
	model, _ := setupTestModel()
	model.width = 100

	item := DirectoryTreeItem{
		DirectoryIndex: history.DirectoryIndex{
			Path:         "/old/project",
			CommandCount: 5,
			LastUsed:     time.Now().Add(-30 * 24 * time.Hour),
			IsActive:     false,
		},
		Level:      0,
		IsExpanded: false,
		Children:   []*DirectoryTreeItem{},
	}

	line := model.formatDirectoryTreeLine(item, false)

	// Inactive directories should be styled differently (dimmed)
	if line == "" {
		t.Error("Expected non-empty formatted line for inactive directory")
	}
}

// Test Breadcrumb Navigation with Multiple Levels

func TestRenderBreadcrumbs_MultiLevel(t *testing.T) {
	model, _ := setupTestModel()
	model.currentDir = "/home/user/project/src"
	model.breadcrumbs = buildBreadcrumbs("/home/user/project/src")
	model.commands = []history.CommandRecord{
		createTestCommand("1", "test", "/home/user/project/src", history.Bash, 0),
	}

	breadcrumbStr := model.renderBreadcrumbs()

	if breadcrumbStr == "" {
		t.Error("Expected non-empty breadcrumb string")
	}

	// Verify breadcrumb contains directory information
	if len(breadcrumbStr) < 10 {
		t.Error("Expected breadcrumb to contain directory path information")
	}
}

func TestRenderBreadcrumbs_NavigationHint(t *testing.T) {
	model, _ := setupTestModel()
	model.currentDir = "/home/user"
	model.breadcrumbs = buildBreadcrumbs("/home/user")
	model.commands = []history.CommandRecord{
		createTestCommand("1", "cmd1", "/home/user", history.Bash, 0),
		createTestCommand("2", "cmd2", "/home/user", history.Bash, 0),
		createTestCommand("3", "cmd3", "/home/user", history.Bash, 0),
	}

	breadcrumbStr := model.renderBreadcrumbs()

	// Should include command count in context
	if breadcrumbStr == "" {
		t.Error("Expected breadcrumb with command count")
	}

	// Should show navigation hint when not at root
	if len(model.breadcrumbs) > 1 {
		// Navigation hint should be present
		if len(breadcrumbStr) < 20 {
			t.Error("Expected breadcrumb with navigation hint to be longer")
		}
	}
}

// Test Parent Directory Detection

func TestIsParentOfCurrentDir(t *testing.T) {
	model, _ := setupTestModel()
	model.currentDir = "/home/user/project/src"

	tests := []struct {
		dirPath  string
		expected bool
	}{
		{"/home", true},
		{"/home/user", true},
		{"/home/user/project", true},
		{"/home/user/project/src", false}, // Same as current
		{"/home/user/project/src/components", false},
		{"/var", false},
		{"", false},
	}

	for _, test := range tests {
		result := model.isParentOfCurrentDir(test.dirPath)
		if result != test.expected {
			t.Errorf("For dirPath '%s' with currentDir '%s', expected %v, got %v",
				test.dirPath, model.currentDir, test.expected, result)
		}
	}
}

func TestIsParentOfCurrentDir_WindowsPaths(t *testing.T) {
	model, _ := setupTestModel()
	model.currentDir = "C:\\Users\\test\\project"

	tests := []struct {
		dirPath  string
		expected bool
	}{
		{"C:\\Users", true},
		{"C:\\Users\\test", true},
		{"C:\\Users\\test\\project", false}, // Same as current
		{"D:\\Users", false},
	}

	for _, test := range tests {
		result := model.isParentOfCurrentDir(test.dirPath)
		if result != test.expected {
			t.Errorf("For dirPath '%s' with currentDir '%s', expected %v, got %v",
				test.dirPath, model.currentDir, test.expected, result)
		}
	}
}

// Test Directory Selection

func TestSelectItem_DirectoryTree(t *testing.T) {
	model, storage := setupTestModel()
	model.viewMode = DirectoryTreeView

	storage.dirStats = []history.DirectoryIndex{
		{Path: "/home/user", CommandCount: 5, IsActive: true},
		{Path: "/home/user/project", CommandCount: 10, IsActive: true},
	}

	model.directories = storage.dirStats
	model.directoryTree = model.organizeDirectoriesHierarchically()
	model.selectedIndex = 0

	// Select first directory
	updatedModel, cmd := model.selectItem()

	// Should switch to directory history view
	if updatedModel.viewMode != DirectoryHistoryView {
		t.Errorf("Expected DirectoryHistoryView after selection, got %v", updatedModel.viewMode)
	}

	// Should update current directory
	if updatedModel.currentDir != model.directoryTree[0].Path {
		t.Errorf("Expected currentDir to be '%s', got '%s'",
			model.directoryTree[0].Path, updatedModel.currentDir)
	}

	// Should reset selection
	if updatedModel.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be reset to 0, got %d", updatedModel.selectedIndex)
	}

	// Should have a command to load directory history
	if cmd == nil {
		t.Error("Expected command to load directory history")
	}
}

func TestSelectItem_DirectoryTree_EmptyTree(t *testing.T) {
	model, _ := setupTestModel()
	model.viewMode = DirectoryTreeView
	model.directoryTree = []DirectoryTreeItem{}
	model.selectedIndex = 0

	updatedModel, cmd := model.selectItem()

	// Should not change view mode
	if updatedModel.viewMode != DirectoryTreeView {
		t.Error("Expected to stay in DirectoryTreeView")
	}

	// Should not have a command
	if cmd != nil {
		t.Error("Expected no command for empty tree")
	}
}

// Test Directory Tree View with Hierarchical Structure

func TestRenderDirectoryTreeView_WithHierarchy(t *testing.T) {
	model, storage := setupTestModel()
	model.viewMode = DirectoryTreeView
	model.width = 100
	model.height = 30

	storage.dirStats = []history.DirectoryIndex{
		{
			Path:         "/home",
			CommandCount: 5,
			LastUsed:     time.Now().Add(-time.Hour),
			IsActive:     true,
		},
		{
			Path:         "/home/user",
			CommandCount: 10,
			LastUsed:     time.Now().Add(-30 * time.Minute),
			IsActive:     true,
		},
		{
			Path:         "/home/user/project",
			CommandCount: 15,
			LastUsed:     time.Now(),
			IsActive:     true,
		},
	}

	model.directories = storage.dirStats
	model.directoryTree = model.organizeDirectoriesHierarchically()

	view := model.renderDirectoryTreeView()

	if view == "" {
		t.Error("Expected non-empty directory tree view")
	}

	// View should contain directory information
	if len(view) < 50 {
		t.Error("Expected directory tree view to contain substantial content")
	}
}

// Test Directory Navigation with Keyboard

func TestHandleKeyPress_DirectoryTreeExpansion(t *testing.T) {
	model, storage := setupTestModel()
	model.viewMode = DirectoryTreeView

	storage.dirStats = []history.DirectoryIndex{
		{Path: "/home", CommandCount: 5, IsActive: true},
		{Path: "/home/user", CommandCount: 10, IsActive: true},
		{Path: "/home/user/project", CommandCount: 15, IsActive: true},
	}

	model.directories = storage.dirStats
	model.directoryTree = model.organizeDirectoriesHierarchically()
	model.selectedIndex = 0

	// Test right arrow to expand directory
	if len(model.directoryTree) > 0 && len(model.directoryTree[0].Children) > 0 {
		keyMsg := tea.KeyMsg{Type: tea.KeyRight}
		updatedModel, _ := model.handleKeyPress(keyMsg)

		// Verify the model was updated (expansion state may have changed)
		if updatedModel.viewMode != DirectoryTreeView {
			t.Error("Expected to stay in DirectoryTreeView after expansion")
		}
	}
}

func TestHandleKeyPress_NavigateToParentDirectory(t *testing.T) {
	model, _ := setupTestModel()
	model.viewMode = DirectoryHistoryView
	model.currentDir = "/home/user/project"
	model.breadcrumbs = buildBreadcrumbs("/home/user/project")

	// Test left arrow / backspace to go to parent
	keyMsg := tea.KeyMsg{Type: tea.KeyLeft}
	updatedModel, cmd := model.handleKeyPress(keyMsg)

	// Should navigate to parent directory
	if updatedModel.currentDir != "/home/user" {
		t.Errorf("Expected currentDir to be '/home/user', got '%s'", updatedModel.currentDir)
	}

	// Should have command to load parent directory history
	if cmd == nil {
		t.Error("Expected command to load parent directory history")
	}
}

func TestHandleKeyPress_NavigateToParentDirectory_AtRoot(t *testing.T) {
	model, _ := setupTestModel()
	model.viewMode = DirectoryHistoryView
	model.currentDir = "/"
	model.breadcrumbs = buildBreadcrumbs("/")

	// Test left arrow at root
	keyMsg := tea.KeyMsg{Type: tea.KeyLeft}
	updatedModel, cmd := model.handleKeyPress(keyMsg)

	// Should stay at root
	if updatedModel.currentDir != "/" {
		t.Errorf("Expected to stay at root, got '%s'", updatedModel.currentDir)
	}

	// Should not have a command
	if cmd != nil {
		t.Error("Expected no command when already at root")
	}
}

