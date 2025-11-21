package browser

import (
	"fmt"
	"strings"
	"time"

	"github.com/ValGrace/command-history-tracker/pkg/history"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ViewMode represents different UI view modes
type ViewMode int

const (
	DirectoryHistoryView ViewMode = iota
	DirectoryTreeView
	SearchView
)

// FilterMode represents different filtering modes
type FilterMode int

const (
	NoFilter FilterMode = iota
	TextFilter
	DateRangeFilter
	ShellFilter
	CombinedFilter
)

// DateFilterConfig represents date range filtering options
type DateFilterConfig struct {
	Enabled   bool
	StartTime time.Time
	EndTime   time.Time
	Preset    DatePreset
}

// DatePreset represents common date range presets
type DatePreset int

const (
	NoDatePreset DatePreset = iota
	Today
	Yesterday
	ThisWeek
	LastWeek
	ThisMonth
	LastMonth
)

// UIModel represents the bubbletea model for the history browser
type UIModel struct {
	// Core state
	viewMode    ViewMode
	currentDir  string
	commands    []history.CommandRecord
	directories []history.DirectoryIndex

	// Selection and navigation
	selectedIndex int
	scrollOffset  int
	pageSize      int
	currentPage   int
	totalPages    int

	// Search and filtering
	searchQuery  string
	searchMode   bool
	filteredCmds []history.CommandRecord

	// Advanced filtering
	filterMode  FilterMode
	dateFilter  DateFilterConfig
	shellFilter history.ShellType
	showFilters bool

	// Cross-directory navigation
	breadcrumbs   []string
	parentDirs    []string
	directoryTree []DirectoryTreeItem
	treeExpanded  map[string]bool

	// UI dimensions
	width  int
	height int

	// Dependencies
	storage history.StorageEngine

	// UI state
	quitting    bool
	error       error
	selectedCmd *history.CommandRecord
	showPreview bool
}

// NewUIModel creates a new terminal UI model
func NewUIModel(storage history.StorageEngine, currentDir string) *UIModel {
	return &UIModel{
		viewMode:      DirectoryHistoryView,
		currentDir:    currentDir,
		commands:      []history.CommandRecord{},
		directories:   []history.DirectoryIndex{},
		selectedIndex: 0,
		scrollOffset:  0,
		pageSize:      50, // Show 50 commands per page for better performance
		currentPage:   0,
		totalPages:    0,
		searchQuery:   "",
		searchMode:    false,
		filteredCmds:  []history.CommandRecord{},
		filterMode:    NoFilter,
		dateFilter:    DateFilterConfig{Enabled: false},
		shellFilter:   history.Unknown,
		showFilters:   false,
		breadcrumbs:   buildBreadcrumbs(currentDir),
		parentDirs:    []string{},
		directoryTree: []DirectoryTreeItem{},
		treeExpanded:  make(map[string]bool),
		width:         80,
		height:        24,
		storage:       storage,
		quitting:      false,
		error:         nil,
		selectedCmd:   nil,
		showPreview:   false,
	}
}

// Init implements tea.Model
func (m UIModel) Init() tea.Cmd {
	return tea.Batch(
		loadDirectoryHistory(m.storage, m.currentDir),
		loadDirectoryTree(m.storage),
	)
}

// Update implements tea.Model
func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case directoryHistoryMsg:
		m.commands = msg.commands
		// Initialize filtered commands for directory-based browsing
		// Commands are already sorted chronologically (recent-first) from storage
		if len(m.commands) > 0 {
			// Apply existing filters to new commands
			m = m.applyFilters()
		} else {
			// No commands in this directory
			m.filteredCmds = []history.CommandRecord{}
		}
		// Reset selection when loading new directory
		m.selectedIndex = 0
		m.scrollOffset = 0
		return m, nil

	case directoryTreeMsg:
		m.directories = msg.directories
		// Build hierarchical directory tree for enhanced navigation
		m.directoryTree = m.organizeDirectoriesHierarchically()
		return m, nil

	case errorMsg:
		m.error = msg.error
		return m, nil
	}

	return m, nil
}

// View implements tea.Model
func (m UIModel) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if m.error != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'q' to quit.", m.error)
	}

	switch m.viewMode {
	case DirectoryHistoryView:
		return m.renderDirectoryHistoryView()
	case DirectoryTreeView:
		return m.renderDirectoryTreeView()
	case SearchView:
		return m.renderSearchView()
	default:
		return "Unknown view mode"
	}
}

// handleKeyPress processes keyboard input
func (m UIModel) handleKeyPress(msg tea.KeyMsg) (UIModel, tea.Cmd) {
	// Global key bindings
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "tab":
		return m.switchViewMode(), nil

	case "/":
		if !m.searchMode {
			m.searchMode = true
			m.searchQuery = ""
			m.filterMode = TextFilter
			return m, nil
		}

	case "f":
		// Toggle filter panel
		m.showFilters = !m.showFilters
		return m, nil

	case "d":
		// Toggle date filter
		if m.showFilters {
			m.dateFilter.Enabled = !m.dateFilter.Enabled
			if m.dateFilter.Enabled {
				m.filterMode = DateRangeFilter
				m.dateFilter.Preset = Today
				return m.applyDateFilter(), nil
			} else {
				return m.clearFilters(), nil
			}
		}

	case "s":
		// Cycle through shell filters
		if m.showFilters {
			m.shellFilter = m.getNextShellFilter()
			m.filterMode = ShellFilter
			return m.applyFilters(), nil
		}
	}

	// Search mode key bindings
	if m.searchMode {
		return m.handleSearchInput(msg)
	}

	// Filter mode key bindings
	if m.showFilters {
		switch msg.String() {
		case "1", "2", "3", "4", "5", "6":
			// Quick date preset selection
			if m.dateFilter.Enabled {
				presetNum := int(msg.String()[0] - '0')
				m.dateFilter.Preset = DatePreset(presetNum)
				return m.applyDateFilter(), nil
			}
		case "c":
			// Clear all filters
			return m.clearFilters(), nil
		}
	}

	// Navigation key bindings
	switch msg.String() {
	case "up", "k":
		return m.moveUp(), nil

	case "down", "j":
		return m.moveDown(), nil

	case "enter":
		return m.selectItem()

	case "space", " ":
		m.showPreview = !m.showPreview
		return m, nil

	case "t":
		m.viewMode = DirectoryTreeView
		return m, loadDirectoryTree(m.storage)

	case "h":
		m.viewMode = DirectoryHistoryView
		return m, loadDirectoryHistory(m.storage, m.currentDir)

	case "r":
		// Refresh current view
		switch m.viewMode {
		case DirectoryHistoryView:
			return m, loadDirectoryHistory(m.storage, m.currentDir)
		case DirectoryTreeView:
			return m, loadDirectoryTree(m.storage)
		}

	case "e":
		// Expand all directories in tree view
		if m.viewMode == DirectoryTreeView {
			for _, dir := range m.directories {
				m.treeExpanded[dir.Path] = true
			}
			m.directoryTree = m.organizeDirectoriesHierarchically()
			return m, nil
		}

	case "c":
		if m.viewMode == DirectoryTreeView && !m.showFilters {
			// Collapse all directories in tree view
			m.treeExpanded = make(map[string]bool)
			m.directoryTree = m.organizeDirectoriesHierarchically()
			return m, nil
		}

	case "backspace", "left":
		if m.viewMode == DirectoryHistoryView {
			// Navigate to parent directory
			parentDir := getParentDirectory(m.currentDir)
			if parentDir != "" {
				m.currentDir = parentDir
				m.breadcrumbs = buildBreadcrumbs(parentDir)
				return m, loadDirectoryHistory(m.storage, parentDir)
			}
		} else if m.viewMode == DirectoryTreeView {
			// Collapse current directory or navigate to parent
			if len(m.directoryTree) > 0 && m.selectedIndex < len(m.directoryTree) {
				selectedItem := m.directoryTree[m.selectedIndex]

				// If expanded, collapse it
				if selectedItem.IsExpanded && len(selectedItem.Children) > 0 {
					m.treeExpanded[selectedItem.Path] = false
					m.directoryTree = m.organizeDirectoriesHierarchically()
					return m, nil
				} else {
					// Navigate to parent directory in tree
					parentPath := getParentDirectory(selectedItem.Path)
					if parentPath != "" {
						// Find parent in tree and select it
						for i, item := range m.directoryTree {
							if item.Path == parentPath {
								m.selectedIndex = i
								break
							}
						}
					}
				}
			}
		}

	case "right":
		// Navigate into selected directory (if in tree view)
		if m.viewMode == DirectoryTreeView {
			if len(m.directoryTree) > 0 && m.selectedIndex < len(m.directoryTree) {
				selectedItem := m.directoryTree[m.selectedIndex]

				// If it has children, toggle expansion
				if len(selectedItem.Children) > 0 {
					m.treeExpanded[selectedItem.Path] = !m.treeExpanded[selectedItem.Path]
					m.directoryTree = m.organizeDirectoriesHierarchically()
					return m, nil
				} else {
					// Navigate to directory
					m.currentDir = selectedItem.Path
					m.breadcrumbs = buildBreadcrumbs(selectedItem.Path)
					m.viewMode = DirectoryHistoryView
					return m, loadDirectoryHistory(m.storage, selectedItem.Path)
				}
			}
		}
	}

	return m, nil
}

// handleSearchInput processes search mode input
func (m UIModel) handleSearchInput(msg tea.KeyMsg) (UIModel, tea.Cmd) {
	switch msg.String() {
	case "escape":
		m.searchMode = false
		m.searchQuery = ""
		m.filteredCmds = m.commands
		m.selectedIndex = 0
		return m, nil

	case "enter":
		m.searchMode = false
		return m.filterCommands(), nil

	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			return m.filterCommands(), nil
		}

	default:
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
			return m.filterCommands(), nil
		}
	}

	return m, nil
}

// switchViewMode cycles through available view modes
func (m UIModel) switchViewMode() UIModel {
	switch m.viewMode {
	case DirectoryHistoryView:
		m.viewMode = DirectoryTreeView
	case DirectoryTreeView:
		m.viewMode = DirectoryHistoryView
	}
	return m
}

// moveUp moves selection up
func (m UIModel) moveUp() UIModel {
	if m.selectedIndex > 0 {
		m.selectedIndex--

		// Adjust scroll offset if needed
		if m.selectedIndex < m.scrollOffset {
			m.scrollOffset = m.selectedIndex
		}
	}
	return m
}

// moveDown moves selection down
func (m UIModel) moveDown() UIModel {
	maxIndex := m.getMaxIndex()
	if m.selectedIndex < maxIndex {
		m.selectedIndex++

		// Adjust scroll offset if needed
		visibleLines := m.height - 5 // Account for header and footer
		if m.selectedIndex >= m.scrollOffset+visibleLines {
			m.scrollOffset = m.selectedIndex - visibleLines + 1
		}
	}
	return m
}

// getMaxIndex returns the maximum selectable index for current view
func (m UIModel) getMaxIndex() int {
	switch m.viewMode {
	case DirectoryHistoryView:
		if len(m.filteredCmds) == 0 {
			return 0
		}
		return len(m.filteredCmds) - 1
	case DirectoryTreeView:
		if len(m.directoryTree) == 0 {
			return 0
		}
		return len(m.directoryTree) - 1
	default:
		return 0
	}
}

// selectItem handles item selection with enhanced directory-based functionality
func (m UIModel) selectItem() (UIModel, tea.Cmd) {
	switch m.viewMode {
	case DirectoryHistoryView:
		if len(m.filteredCmds) > 0 && m.selectedIndex < len(m.filteredCmds) {
			// Store selected command for retrieval
			m.selectedCmd = &m.filteredCmds[m.selectedIndex]
			// Return selected command (will be handled by parent)
			return m, tea.Quit
		}
	case DirectoryTreeView:
		if len(m.directoryTree) > 0 && m.selectedIndex < len(m.directoryTree) {
			// Switch to selected directory for directory-based browsing
			selectedItem := m.directoryTree[m.selectedIndex]
			m.currentDir = selectedItem.Path
			m.breadcrumbs = buildBreadcrumbs(selectedItem.Path)
			m.viewMode = DirectoryHistoryView
			m.selectedIndex = 0
			m.scrollOffset = 0
			// Clear any existing filters when switching directories
			m.searchQuery = ""
			m.searchMode = false
			m.filteredCmds = []history.CommandRecord{}
			return m, loadDirectoryHistory(m.storage, selectedItem.Path)
		}
	}
	return m, nil
}

// filterCommands applies search filter to commands
func (m UIModel) filterCommands() UIModel {
	return m.applyFilters()
}

// GetSelectedCommand returns the currently selected command
func (m UIModel) GetSelectedCommand() *history.CommandRecord {
	// Return stored selected command if available (from selectItem)
	if m.selectedCmd != nil {
		return m.selectedCmd
	}

	// Fallback to current selection
	if m.viewMode == DirectoryHistoryView && len(m.filteredCmds) > 0 && m.selectedIndex < len(m.filteredCmds) {
		return &m.filteredCmds[m.selectedIndex]
	}
	return nil
}

// Styles for UI components
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("240"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Background(lipgloss.Color("57"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	searchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Background(lipgloss.Color("235"))

	breadcrumbStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)
)

// buildBreadcrumbs creates breadcrumb navigation from a directory path
func buildBreadcrumbs(dir string) []string {
	if dir == "" || dir == "." {
		return []string{"."}
	}

	// Split path into components
	parts := strings.Split(strings.ReplaceAll(dir, "\\", "/"), "/")
	breadcrumbs := []string{}

	// Build cumulative paths
	currentPath := ""
	for i, part := range parts {
		if part == "" && i == 0 {
			// Root directory on Unix
			currentPath = "/"
			breadcrumbs = append(breadcrumbs, "/")
		} else if part != "" {
			if currentPath == "" || currentPath == "/" {
				currentPath = currentPath + part
			} else {
				currentPath = currentPath + "/" + part
			}
			breadcrumbs = append(breadcrumbs, currentPath)
		}
	}

	if len(breadcrumbs) == 0 {
		return []string{"."}
	}

	return breadcrumbs
}

// getParentDirectory returns the parent directory of the current directory
func getParentDirectory(dir string) string {
	if dir == "" || dir == "." || dir == "/" {
		return ""
	}

	// Normalize path separators
	normalizedDir := strings.ReplaceAll(dir, "\\", "/")

	// Remove trailing slash
	normalizedDir = strings.TrimSuffix(normalizedDir, "/")

	// Find last separator
	lastSep := strings.LastIndex(normalizedDir, "/")
	if lastSep == -1 {
		return "."
	}

	if lastSep == 0 {
		return "/"
	}

	return normalizedDir[:lastSep]
}

// getNextShellFilter cycles through shell filter options
func (m UIModel) getNextShellFilter() history.ShellType {
	switch m.shellFilter {
	case history.Unknown:
		return history.PowerShell
	case history.PowerShell:
		return history.Bash
	case history.Bash:
		return history.Zsh
	case history.Zsh:
		return history.Cmd
	case history.Cmd:
		return history.Unknown // Back to no filter
	default:
		return history.Unknown
	}
}

// applyDateFilter applies date range filtering
func (m UIModel) applyDateFilter() UIModel {
	if !m.dateFilter.Enabled {
		return m.applyFilters()
	}

	now := time.Now()
	switch m.dateFilter.Preset {
	case Today:
		m.dateFilter.StartTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		m.dateFilter.EndTime = m.dateFilter.StartTime.Add(24 * time.Hour)
	case Yesterday:
		yesterday := now.AddDate(0, 0, -1)
		m.dateFilter.StartTime = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
		m.dateFilter.EndTime = m.dateFilter.StartTime.Add(24 * time.Hour)
	case ThisWeek:
		// Start of week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday = 7
		}
		daysFromMonday := weekday - 1
		monday := now.AddDate(0, 0, -daysFromMonday)
		m.dateFilter.StartTime = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
		m.dateFilter.EndTime = m.dateFilter.StartTime.Add(7 * 24 * time.Hour)
	case LastWeek:
		// Previous week
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		daysFromLastMonday := weekday + 6
		lastMonday := now.AddDate(0, 0, -daysFromLastMonday)
		m.dateFilter.StartTime = time.Date(lastMonday.Year(), lastMonday.Month(), lastMonday.Day(), 0, 0, 0, 0, lastMonday.Location())
		m.dateFilter.EndTime = m.dateFilter.StartTime.Add(7 * 24 * time.Hour)
	case ThisMonth:
		m.dateFilter.StartTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		m.dateFilter.EndTime = m.dateFilter.StartTime.AddDate(0, 1, 0)
	case LastMonth:
		lastMonth := now.AddDate(0, -1, 0)
		m.dateFilter.StartTime = time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, lastMonth.Location())
		m.dateFilter.EndTime = m.dateFilter.StartTime.AddDate(0, 1, 0)
	}

	return m.applyFilters()
}

// applyFilters applies all active filters to the command list
func (m UIModel) applyFilters() UIModel {
	m.filteredCmds = []history.CommandRecord{}

	for _, cmd := range m.commands {
		// Apply text filter
		if m.searchQuery != "" {
			if !strings.Contains(strings.ToLower(cmd.Command), strings.ToLower(m.searchQuery)) {
				continue
			}
		}

		// Apply date filter
		if m.dateFilter.Enabled {
			if cmd.Timestamp.Before(m.dateFilter.StartTime) || cmd.Timestamp.After(m.dateFilter.EndTime) {
				continue
			}
		}

		// Apply shell filter
		if m.shellFilter != history.Unknown {
			if cmd.Shell != m.shellFilter {
				continue
			}
		}

		m.filteredCmds = append(m.filteredCmds, cmd)
	}

	m.selectedIndex = 0
	m.scrollOffset = 0
	return m
}

// clearFilters removes all active filters
func (m UIModel) clearFilters() UIModel {
	m.searchQuery = ""
	m.searchMode = false
	m.dateFilter.Enabled = false
	m.shellFilter = history.Unknown
	m.filterMode = NoFilter
	m.filteredCmds = m.commands
	m.selectedIndex = 0
	m.scrollOffset = 0
	return m
}

// cycleDatePreset cycles through date filter presets
func (m UIModel) cycleDatePreset() UIModel {
	switch m.dateFilter.Preset {
	case Today:
		m.dateFilter.Preset = Yesterday
	case Yesterday:
		m.dateFilter.Preset = ThisWeek
	case ThisWeek:
		m.dateFilter.Preset = LastWeek
	case LastWeek:
		m.dateFilter.Preset = ThisMonth
	case ThisMonth:
		m.dateFilter.Preset = LastMonth
	case LastMonth:
		m.dateFilter.Preset = Today
	default:
		m.dateFilter.Preset = Today
	}

	return m.applyDateFilter()
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	} else {
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	}
}
