package browser

import (
	"fmt"
	"strings"
	"time"

	"github.com/ValGrace/command-history-tracker/pkg/history"
	"github.com/charmbracelet/lipgloss"
)

// renderDirectoryHistoryView renders the command history for current directory
func (m UIModel) renderDirectoryHistoryView() string {
	var b strings.Builder

	// Enhanced breadcrumb navigation with current directory emphasis
	b.WriteString(m.renderBreadcrumbs())
	b.WriteString("\n")

	// Enhanced header with command count, filter status, and directory context
	cmdCount := len(m.filteredCmds)
	totalCount := len(m.commands)

	// Show current directory name in header
	dirName := m.currentDir
	if dirName == "" || dirName == "." {
		dirName = "current directory"
	} else {
		// Show just the directory name, not full path
		if strings.Contains(dirName, "/") {
			parts := strings.Split(dirName, "/")
			dirName = parts[len(parts)-1]
		}
		if strings.Contains(dirName, "\\") {
			parts := strings.Split(dirName, "\\")
			dirName = parts[len(parts)-1]
		}
		if dirName == "" {
			dirName = "root"
		}
	}

	var header string
	if cmdCount != totalCount {
		header = fmt.Sprintf("📂 Current Directory History - %s (%d of %d commands)", dirName, cmdCount, totalCount)
	} else {
		header = fmt.Sprintf("📂 Current Directory History - %s (%d commands)", dirName, cmdCount)
	}

	if m.searchMode {
		header += fmt.Sprintf(" 🔍 Search: %s", m.searchQuery)
	}

	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")

	// Add directory context information
	if totalCount > 0 {
		// Show recent activity summary
		var recentActivity string
		if len(m.commands) > 0 {
			mostRecent := m.commands[0] // Commands are sorted by timestamp DESC
			timeSince := time.Since(mostRecent.Timestamp)
			if timeSince < time.Hour {
				recentActivity = fmt.Sprintf("Last command: %s ago", formatDuration(timeSince))
			} else if timeSince < 24*time.Hour {
				recentActivity = fmt.Sprintf("Last command: %s", mostRecent.Timestamp.Format("15:04 today"))
			} else {
				recentActivity = fmt.Sprintf("Last command: %s", mostRecent.Timestamp.Format("Jan 02 15:04"))
			}
		}

		if recentActivity != "" {
			b.WriteString(dimStyle.Render(fmt.Sprintf("📅 %s", recentActivity)))
			b.WriteString("\n")
		}
	}

	// Filter panel
	if m.showFilters {
		b.WriteString(m.renderFilterPanel())
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Add column headers for directory-based command display
	if len(m.filteredCmds) > 0 {
		headerLine := fmt.Sprintf("%3s │ %-*s │ %-16s │ %-10s │ %s",
			"#", 30, "Command", "Time", "Shell", "Status")
		b.WriteString(dimStyle.Render(headerLine))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(strings.Repeat("─", len(headerLine))))
		b.WriteString("\n")
	}

	// Enhanced commands list with better empty states and directory-specific messaging
	if len(m.filteredCmds) == 0 {
		if m.searchQuery != "" {
			b.WriteString(dimStyle.Render("🔍 No commands found matching search criteria in this directory."))
			b.WriteString("\n")
			b.WriteString(dimStyle.Render("   Try a different search term or clear filters with 'c'"))
		} else if totalCount == 0 {
			b.WriteString(dimStyle.Render("📭 No commands recorded in this directory yet."))
			b.WriteString("\n")
			b.WriteString(dimStyle.Render("   Commands will appear here as you execute them in this directory."))
			b.WriteString("\n")
			b.WriteString(dimStyle.Render("   Use 't' to browse other directories with command history."))
		} else {
			b.WriteString(dimStyle.Render("🚫 All commands are filtered out."))
			b.WriteString("\n")
			b.WriteString(dimStyle.Render("   Clear filters with 'c' to see all commands."))
		}
		b.WriteString("\n")
	} else {
		// Calculate available space for command list
		availableHeight := m.height - 8 // Account for header, breadcrumbs, footer, padding
		if m.showFilters {
			availableHeight -= 3 // Account for filter panel
		}
		if m.showPreview && m.selectedIndex < len(m.filteredCmds) {
			availableHeight = availableHeight / 2 // Split screen for preview
		}

		visibleLines := availableHeight
		if visibleLines < 1 {
			visibleLines = 1
		}

		start := m.scrollOffset
		end := start + visibleLines
		if end > len(m.filteredCmds) {
			end = len(m.filteredCmds)
		}

		// Directory-based command list rendering with chronological order (recent-first)
		// Commands are already sorted by timestamp DESC from storage
		for i := start; i < end; i++ {
			cmd := m.filteredCmds[i]
			isSelected := i == m.selectedIndex

			line := m.formatDirectoryCommandLine(cmd, isSelected, i)
			b.WriteString(line)
			b.WriteString("\n")
		}

		// Enhanced scroll indicator with directory context
		if len(m.filteredCmds) > visibleLines {
			scrollInfo := fmt.Sprintf("📄 Showing %d-%d of %d commands in %s",
				start+1, end, len(m.filteredCmds), m.currentDir)

			// Add navigation hints
			var navHints []string
			if start > 0 {
				navHints = append(navHints, "↑ more above")
			}
			if end < len(m.filteredCmds) {
				navHints = append(navHints, "↓ more below")
			}

			if len(navHints) > 0 {
				scrollInfo += fmt.Sprintf(" (%s)", strings.Join(navHints, ", "))
			}

			b.WriteString(dimStyle.Render(scrollInfo))
			b.WriteString("\n")
		}

		// Preview pane for selected command
		if m.showPreview && m.selectedIndex < len(m.filteredCmds) {
			b.WriteString("\n")
			b.WriteString(m.renderCommandPreview(m.filteredCmds[m.selectedIndex]))
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

// renderDirectoryTreeView renders the directory tree with command counts
func (m UIModel) renderDirectoryTreeView() string {
	var b strings.Builder

	// Enhanced breadcrumb navigation with tree context
	b.WriteString(m.renderBreadcrumbs())
	b.WriteString("\n")

	// Enhanced header with directory count and navigation hints
	totalDirs := len(m.directories)
	visibleDirs := len(m.directoryTree)
	header := fmt.Sprintf("🌳 Directory Tree - Navigate with ↑↓, expand/collapse with →, select with Enter (%d directories)", totalDirs)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")

	// Add tree navigation help
	if visibleDirs > 0 {
		helpText := "💡 Use → to expand/collapse folders, Enter to browse directory history"
		b.WriteString(dimStyle.Render(helpText))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Enhanced directory tree with hierarchical display
	if len(m.directoryTree) == 0 {
		if totalDirs == 0 {
			b.WriteString(dimStyle.Render("📭 No directories with command history found."))
			b.WriteString("\n")
			b.WriteString(dimStyle.Render("   Execute some commands to start building your history!"))
		} else {
			b.WriteString(dimStyle.Render("🔄 Building directory tree..."))
		}
		b.WriteString("\n")
	} else {
		// Calculate available space for tree display
		availableHeight := m.height - 8 // Account for header, breadcrumbs, footer, help text
		visibleLines := availableHeight
		if visibleLines < 1 {
			visibleLines = 1
		}

		start := m.scrollOffset
		end := start + visibleLines
		if end > len(m.directoryTree) {
			end = len(m.directoryTree)
		}

		// Render hierarchical directory tree
		for i := start; i < end; i++ {
			item := m.directoryTree[i]
			line := m.formatDirectoryTreeLine(item, i == m.selectedIndex)
			b.WriteString(line)
			b.WriteString("\n")
		}

		// Enhanced scroll indicator with tree context
		if len(m.directoryTree) > visibleLines {
			scrollInfo := fmt.Sprintf("📄 Showing %d-%d of %d directories",
				start+1, end, len(m.directoryTree))

			// Add navigation hints
			var navHints []string
			if start > 0 {
				navHints = append(navHints, "↑ more above")
			}
			if end < len(m.directoryTree) {
				navHints = append(navHints, "↓ more below")
			}

			if len(navHints) > 0 {
				scrollInfo += fmt.Sprintf(" (%s)", strings.Join(navHints, ", "))
			}

			b.WriteString(dimStyle.Render(scrollInfo))
			b.WriteString("\n")
		}

		// Show current directory context in tree
		if m.currentDir != "" {
			currentDirInfo := fmt.Sprintf("📍 Current: %s", m.currentDir)
			b.WriteString(breadcrumbStyle.Render(currentDirInfo))
			b.WriteString("\n")
		}
	}

	// Enhanced footer with tree-specific navigation
	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

// renderSearchView renders the search interface
func (m UIModel) renderSearchView() string {
	var b strings.Builder

	// Header
	header := "Search Commands"
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n\n")

	// Search input
	searchPrompt := fmt.Sprintf("Search: %s", m.searchQuery)
	if m.searchMode {
		searchPrompt += "_" // Cursor indicator
	}
	b.WriteString(searchStyle.Render(searchPrompt))
	b.WriteString("\n\n")

	// Results
	if len(m.filteredCmds) == 0 && m.searchQuery != "" {
		b.WriteString(dimStyle.Render("No commands found matching search criteria."))
		b.WriteString("\n")
	} else if len(m.filteredCmds) > 0 {
		visibleLines := m.height - 8
		start := m.scrollOffset
		end := start + visibleLines
		if end > len(m.filteredCmds) {
			end = len(m.filteredCmds)
		}

		for i := start; i < end; i++ {
			cmd := m.filteredCmds[i]
			line := m.formatCommandLine(cmd, i == m.selectedIndex)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

// formatDirectoryCommandLine formats a command record for directory-based browsing with enhanced selection
func (m UIModel) formatDirectoryCommandLine(cmd history.CommandRecord, selected bool, index int) string {
	// Calculate available width for command text
	maxCmdWidth := m.width - 50 // Leave space for timestamp, shell, exit code, index, and indicators
	if maxCmdWidth < 20 {
		maxCmdWidth = 20 // Minimum width
	}

	// Truncate command if too long
	command := cmd.Command
	if len(command) > maxCmdWidth {
		command = command[:maxCmdWidth-3] + "..."
	}

	// Format timestamp with enhanced relative time display for directory context
	var timestamp string
	now := time.Now()
	timeSince := now.Sub(cmd.Timestamp)

	if cmd.Timestamp.Format("2006-01-02") == now.Format("2006-01-02") {
		// Today - show time and relative indicator
		if timeSince < time.Hour {
			timestamp = fmt.Sprintf("%s (%s ago)", cmd.Timestamp.Format("15:04"), formatDuration(timeSince))
		} else {
			timestamp = cmd.Timestamp.Format("15:04:05")
		}
	} else if timeSince < 24*time.Hour {
		// Yesterday
		timestamp = cmd.Timestamp.Format("yesterday 15:04")
	} else if timeSince < 7*24*time.Hour {
		// This week
		timestamp = cmd.Timestamp.Format("Mon 15:04")
	} else {
		timestamp = cmd.Timestamp.Format("01-02 15:04")
	}

	// Format shell indicator with color coding
	shell := fmt.Sprintf("[%s]", cmd.Shell.String())

	// Enhanced exit code indicator with visual feedback
	var exitIndicator string
	if cmd.ExitCode == 0 {
		exitIndicator = "✅"
	} else {
		exitIndicator = fmt.Sprintf("❌%d", cmd.ExitCode)
	}

	// Add execution time indicator for long-running commands
	var durationIndicator string
	if cmd.Duration > time.Second {
		durationIndicator = fmt.Sprintf("⏱️%s", formatDuration(cmd.Duration))
	}

	// Add command index for easy reference
	indexStr := fmt.Sprintf("%3d", index+1)

	// Create line with enhanced columns for directory browsing
	var line string
	if durationIndicator != "" {
		line = fmt.Sprintf("%s │ %-*s │ %s │ %s │ %s │ %s",
			indexStr, maxCmdWidth, command, timestamp, shell, exitIndicator, durationIndicator)
	} else {
		line = fmt.Sprintf("%s │ %-*s │ %s │ %s │ %s",
			indexStr, maxCmdWidth, command, timestamp, shell, exitIndicator)
	}

	// Apply enhanced styling based on selection and command context
	if selected {
		// Enhanced selection highlighting with directory context
		return selectedStyle.Render("▶ " + line)
	} else if cmd.ExitCode != 0 {
		// Highlight failed commands with error styling
		return errorStyle.Render("  " + line)
	} else if timeSince < 5*time.Minute {
		// Highlight very recent commands in current directory
		return lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("  " + line) // Bright green
	} else if timeSince < time.Hour {
		// Highlight recent commands
		return lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("  " + line) // Light green
	}

	return normalStyle.Render("  " + line)
}

// formatCommandLine formats a command record for display with enhanced current directory context
func (m UIModel) formatCommandLine(cmd history.CommandRecord, selected bool) string {
	// Calculate available width for command text
	maxCmdWidth := m.width - 40 // Leave space for timestamp, shell, exit code, and indicators
	if maxCmdWidth < 20 {
		maxCmdWidth = 20 // Minimum width
	}

	// Truncate command if too long
	command := cmd.Command
	if len(command) > maxCmdWidth {
		command = command[:maxCmdWidth-3] + "..."
	}

	// Format timestamp with enhanced relative time display
	var timestamp string
	now := time.Now()
	timeSince := now.Sub(cmd.Timestamp)

	if cmd.Timestamp.Format("2006-01-02") == now.Format("2006-01-02") {
		// Today - show time and relative indicator
		if timeSince < time.Hour {
			timestamp = fmt.Sprintf("%s (%s ago)", cmd.Timestamp.Format("15:04"), formatDuration(timeSince))
		} else {
			timestamp = cmd.Timestamp.Format("15:04:05")
		}
	} else if timeSince < 24*time.Hour {
		// Yesterday
		timestamp = cmd.Timestamp.Format("yesterday 15:04")
	} else {
		timestamp = cmd.Timestamp.Format("01-02 15:04")
	}

	// Format shell indicator with color coding
	shell := fmt.Sprintf("[%s]", cmd.Shell.String())

	// Enhanced exit code indicator with visual feedback
	var exitIndicator string
	if cmd.ExitCode == 0 {
		exitIndicator = "✅"
	} else {
		exitIndicator = fmt.Sprintf("❌%d", cmd.ExitCode)
	}

	// Add execution time indicator for long-running commands
	var durationIndicator string
	if cmd.Duration > time.Second {
		durationIndicator = fmt.Sprintf("⏱️%s", formatDuration(cmd.Duration))
	}

	// Create line with enhanced columns
	var line string
	if durationIndicator != "" {
		line = fmt.Sprintf("%-*s %s %s %s %s", maxCmdWidth, command, timestamp, shell, exitIndicator, durationIndicator)
	} else {
		line = fmt.Sprintf("%-*s %s %s %s", maxCmdWidth, command, timestamp, shell, exitIndicator)
	}

	// Apply enhanced styling based on context
	if selected {
		// Enhanced selection highlighting
		return selectedStyle.Render("▶ " + line)
	} else if cmd.ExitCode != 0 {
		// Highlight failed commands with error styling
		return errorStyle.Render("  " + line)
	} else if timeSince < 5*time.Minute {
		// Highlight very recent commands
		return lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("  " + line) // Bright green
	}

	return normalStyle.Render("  " + line)
}

// formatDirectoryLine formats a directory entry for display
func (m UIModel) formatDirectoryLine(dir history.DirectoryIndex, selected bool) string {
	// Format command count with visual indicator
	var countStr string
	if dir.CommandCount > 100 {
		countStr = fmt.Sprintf("(%d+ commands)", dir.CommandCount)
	} else {
		countStr = fmt.Sprintf("(%d commands)", dir.CommandCount)
	}

	// Format last used time with relative indicators
	lastUsed := "never"
	if !dir.LastUsed.IsZero() {
		timeSince := time.Since(dir.LastUsed)
		if timeSince < time.Hour {
			lastUsed = "< 1h ago"
		} else if timeSince < 24*time.Hour {
			lastUsed = dir.LastUsed.Format("15:04 today")
		} else if timeSince < 7*24*time.Hour {
			lastUsed = dir.LastUsed.Format("Mon 15:04")
		} else {
			lastUsed = dir.LastUsed.Format("Jan 02")
		}
	}

	// Truncate path if too long, keeping the end visible
	maxPathWidth := m.width - 35
	if maxPathWidth < 20 {
		maxPathWidth = 20
	}

	path := dir.Path
	if len(path) > maxPathWidth {
		// Show end of path (more useful for directory navigation)
		path = "..." + path[len(path)-maxPathWidth+3:]
	}

	// Add activity indicator
	activityIndicator := ""
	if dir.IsActive {
		timeSince := time.Since(dir.LastUsed)
		if timeSince < time.Hour {
			activityIndicator = "🔥" // Very recent
		} else if timeSince < 24*time.Hour {
			activityIndicator = "⚡" // Recent
		} else {
			activityIndicator = "📁" // Normal
		}
	} else {
		activityIndicator = "💤" // Inactive
	}

	// Create line with visual indicators
	line := fmt.Sprintf("%s %-*s %s %s", activityIndicator, maxPathWidth, path, countStr, lastUsed)

	// Apply styling based on activity and selection
	if selected {
		return selectedStyle.Render(line)
	} else if !dir.IsActive {
		return dimStyle.Render(line)
	}
	return normalStyle.Render(line)
}

// renderBreadcrumbs renders the enhanced breadcrumb navigation
func (m UIModel) renderBreadcrumbs() string {
	if len(m.breadcrumbs) == 0 {
		return breadcrumbStyle.Render("📁 . (root)")
	}

	// Enhanced breadcrumb with navigation hints and directory context
	var parts []string

	// Add home/root indicator
	if len(m.breadcrumbs) > 1 {
		parts = append(parts, dimStyle.Render("🏠"))
	}

	// Process each breadcrumb level
	for i, crumb := range m.breadcrumbs {
		// Get just the directory name for display
		displayName := crumb
		if strings.Contains(displayName, "/") {
			pathParts := strings.Split(displayName, "/")
			displayName = pathParts[len(pathParts)-1]
		}
		if strings.Contains(displayName, "\\") {
			pathParts := strings.Split(displayName, "\\")
			displayName = pathParts[len(pathParts)-1]
		}
		if displayName == "" {
			displayName = "root"
		}

		if i == len(m.breadcrumbs)-1 {
			// Current directory - highlight it with enhanced styling
			currentStyle := breadcrumbStyle.Bold(true)
			parts = append(parts, currentStyle.Render(displayName))
		} else {
			// Parent directories - show as navigable with subtle styling
			parentStyle := dimStyle.Underline(true)
			parts = append(parts, parentStyle.Render(displayName))
		}
	}

	// Join with enhanced separators
	separator := dimStyle.Render(" ▶ ")
	breadcrumbPath := strings.Join(parts, separator)

	// Add navigation context and hints
	var contextInfo string
	if m.viewMode == DirectoryTreeView {
		contextInfo = dimStyle.Render(" (browsing tree)")
	} else {
		// Show command count for current directory if available
		cmdCount := len(m.commands)
		if cmdCount > 0 {
			contextInfo = dimStyle.Render(fmt.Sprintf(" (%d commands)", cmdCount))
		} else {
			contextInfo = dimStyle.Render(" (no commands)")
		}
	}

	// Add navigation hint
	navHint := ""
	if len(m.breadcrumbs) > 1 {
		navHint = dimStyle.Render(" • ← to go up")
	}

	return fmt.Sprintf("📁 %s%s%s", breadcrumbPath, contextInfo, navHint)
}

// renderFilterPanel renders the filter configuration panel
func (m UIModel) renderFilterPanel() string {
	var b strings.Builder

	b.WriteString(dimStyle.Render("Filters: "))

	// Text filter status
	if m.searchQuery != "" {
		b.WriteString(fmt.Sprintf("Text: %s ", searchStyle.Render(m.searchQuery)))
	}

	// Date filter status
	if m.dateFilter.Enabled {
		presetName := m.getDatePresetName(m.dateFilter.Preset)
		b.WriteString(fmt.Sprintf("Date: %s ", selectedStyle.Render(presetName)))
	}

	// Shell filter status
	if m.shellFilter != history.Unknown {
		b.WriteString(fmt.Sprintf("Shell: %s ", selectedStyle.Render(m.shellFilter.String())))
	}

	// No filters active
	if m.searchQuery == "" && !m.dateFilter.Enabled && m.shellFilter == history.Unknown {
		b.WriteString(dimStyle.Render("None active"))
	}

	b.WriteString("\n")

	// Filter help
	help := []string{
		"d: date filter",
		"s: shell filter",
		"c: clear all",
		"f: hide filters",
	}

	if m.dateFilter.Enabled {
		help = append(help, "1-6: date presets")
	}

	b.WriteString(dimStyle.Render(strings.Join(help, " • ")))

	return b.String()
}

// getDatePresetName returns the display name for a date preset
func (m UIModel) getDatePresetName(preset DatePreset) string {
	switch preset {
	case Today:
		return "Today"
	case Yesterday:
		return "Yesterday"
	case ThisWeek:
		return "This Week"
	case LastWeek:
		return "Last Week"
	case ThisMonth:
		return "This Month"
	case LastMonth:
		return "Last Month"
	default:
		return "Custom"
	}
}

// renderCommandPreview renders detailed information about the selected command
func (m UIModel) renderCommandPreview(cmd history.CommandRecord) string {
	var b strings.Builder

	// Preview header
	b.WriteString(headerStyle.Render("Command Preview"))
	b.WriteString("\n")

	// Command details
	b.WriteString(fmt.Sprintf("Command: %s\n", normalStyle.Render(cmd.Command)))
	b.WriteString(fmt.Sprintf("Directory: %s\n", dimStyle.Render(cmd.Directory)))
	b.WriteString(fmt.Sprintf("Executed: %s\n", dimStyle.Render(cmd.Timestamp.Format("2006-01-02 15:04:05"))))
	b.WriteString(fmt.Sprintf("Shell: %s\n", dimStyle.Render(cmd.Shell.String())))

	if cmd.ExitCode != 0 {
		b.WriteString(fmt.Sprintf("Exit Code: %s\n", errorStyle.Render(fmt.Sprintf("%d", cmd.ExitCode))))
	} else {
		b.WriteString(fmt.Sprintf("Exit Code: %s\n", dimStyle.Render("0 (success)")))
	}

	if cmd.Duration > 0 {
		b.WriteString(fmt.Sprintf("Duration: %s\n", dimStyle.Render(cmd.Duration.String())))
	}

	if len(cmd.Tags) > 0 {
		b.WriteString(fmt.Sprintf("Tags: %s\n", dimStyle.Render(strings.Join(cmd.Tags, ", "))))
	}

	return b.String()
}

// renderFooter renders the help text footer
func (m UIModel) renderFooter() string {
	var help []string

	switch m.viewMode {
	case DirectoryHistoryView:
		if m.showFilters {
			help = []string{
				"↑/k: up", "↓/j: down", "enter: select", "d: date", "s: shell", "c: clear", "f: hide filters", "q: quit",
			}
		} else {
			cmdCount := len(m.filteredCmds)
			if cmdCount > 0 {
				help = []string{
					"↑/k: up", "↓/j: down", "enter: execute", "space: preview", "←: parent dir", "t: browse dirs", "/: search", "f: filters", "r: refresh", "q: quit",
				}
			} else {
				help = []string{
					"t: browse directories", "←: parent dir", "r: refresh", "q: quit",
				}
			}
		}
	case DirectoryTreeView:
		if len(m.directoryTree) > 0 {
			help = []string{
				"↑/k: up", "↓/j: down", "→: expand/enter", "←: collapse/up", "enter: browse", "h: history", "r: refresh", "q: quit",
			}
		} else {
			help = []string{
				"h: history view", "r: refresh", "q: quit",
			}
		}
	case SearchView:
		if m.searchMode {
			help = []string{
				"type to search", "enter: apply filter", "esc: cancel", "q: quit",
			}
		} else {
			help = []string{
				"↑/k: up", "↓/j: down", "enter: select", "/: new search", "q: quit",
			}
		}
	}

	return dimStyle.Render(strings.Join(help, " • "))
}

// organizeDirectoriesHierarchically organizes directories in a tree-like structure
func (m UIModel) organizeDirectoriesHierarchically() []DirectoryTreeItem {
	if len(m.directories) == 0 {
		return []DirectoryTreeItem{}
	}

	// Create a map to track directory relationships
	dirMap := make(map[string]*DirectoryTreeItem)
	var roots []*DirectoryTreeItem

	// Sort directories by path for consistent ordering
	sortedDirs := make([]history.DirectoryIndex, len(m.directories))
	copy(sortedDirs, m.directories)

	// Simple sort by path length first, then alphabetically
	for i := 0; i < len(sortedDirs); i++ {
		for j := i + 1; j < len(sortedDirs); j++ {
			if len(sortedDirs[i].Path) > len(sortedDirs[j].Path) ||
				(len(sortedDirs[i].Path) == len(sortedDirs[j].Path) && sortedDirs[i].Path > sortedDirs[j].Path) {
				sortedDirs[i], sortedDirs[j] = sortedDirs[j], sortedDirs[i]
			}
		}
	}

	// First pass: create all directory items
	for _, dir := range sortedDirs {
		// Check if this directory should be expanded
		isExpanded := m.treeExpanded[dir.Path]
		if _, exists := m.treeExpanded[dir.Path]; !exists {
			// Default expansion state - expand if it's a parent of current directory
			isExpanded = m.isParentOfCurrentDir(dir.Path)
		}

		item := &DirectoryTreeItem{
			DirectoryIndex: dir,
			Level:          0,
			IsExpanded:     isExpanded,
			Children:       []*DirectoryTreeItem{},
		}
		dirMap[dir.Path] = item
	}

	// Second pass: build hierarchy
	for _, dir := range sortedDirs {
		item := dirMap[dir.Path]
		parentPath := getParentDirectory(dir.Path)

		if parentPath == "" || parentPath == dir.Path || parentPath == "." {
			// This is a root directory
			roots = append(roots, item)
		} else if parent, exists := dirMap[parentPath]; exists {
			// Add as child to parent
			item.Level = parent.Level + 1
			parent.Children = append(parent.Children, item)
		} else {
			// Parent not in our list, treat as root
			roots = append(roots, item)
		}
	}

	// Flatten the tree for display, respecting expansion state
	var flattened []DirectoryTreeItem
	var flatten func(items []*DirectoryTreeItem)
	flatten = func(items []*DirectoryTreeItem) {
		for _, item := range items {
			flattened = append(flattened, *item)
			if item.IsExpanded && len(item.Children) > 0 {
				flatten(item.Children)
			}
		}
	}

	flatten(roots)
	return flattened
}

// isParentOfCurrentDir checks if a directory is a parent of the current directory
func (m UIModel) isParentOfCurrentDir(dirPath string) bool {
	if m.currentDir == "" || dirPath == "" {
		return false
	}

	// Normalize paths for comparison
	currentNorm := strings.ReplaceAll(m.currentDir, "\\", "/")
	dirNorm := strings.ReplaceAll(dirPath, "\\", "/")

	// Check if current directory starts with this directory path
	return strings.HasPrefix(currentNorm, dirNorm) && currentNorm != dirNorm
}

// DirectoryTreeItem represents a directory in the tree view
type DirectoryTreeItem struct {
	history.DirectoryIndex
	Level      int
	IsExpanded bool
	Children   []*DirectoryTreeItem
}

// formatDirectoryTreeLine formats a directory tree item for display with indentation
func (m UIModel) formatDirectoryTreeLine(item DirectoryTreeItem, selected bool) string {
	// Create enhanced indentation with tree connectors
	var indent string
	if item.Level > 0 {
		// Create tree-like indentation with connectors
		for i := 0; i < item.Level; i++ {
			if i == item.Level-1 {
				indent += "├─ "
			} else {
				indent += "│  "
			}
		}
	}

	// Enhanced tree structure indicators with expansion state
	var treeIndicator string
	if len(item.Children) > 0 {
		if item.IsExpanded {
			treeIndicator = "📂 " // Open folder
		} else {
			treeIndicator = "📁 " // Closed folder
		}
	} else {
		treeIndicator = "📄 " // File/leaf directory
	}

	// Enhanced command count with visual indicators
	var countStr string
	if item.CommandCount > 1000 {
		countStr = fmt.Sprintf("🔥%dk", item.CommandCount/1000)
	} else if item.CommandCount > 100 {
		countStr = fmt.Sprintf("⚡%d+", item.CommandCount)
	} else if item.CommandCount > 10 {
		countStr = fmt.Sprintf("📈%d", item.CommandCount)
	} else if item.CommandCount > 0 {
		countStr = fmt.Sprintf("(%d)", item.CommandCount)
	} else {
		countStr = "(0)"
	}

	// Enhanced last used time with relative indicators
	var lastUsed string
	if !item.LastUsed.IsZero() {
		timeSince := time.Since(item.LastUsed)
		if timeSince < time.Hour {
			lastUsed = "🕐 now"
		} else if timeSince < 6*time.Hour {
			lastUsed = "🕕 recent"
		} else if timeSince < 24*time.Hour {
			lastUsed = "📅 today"
		} else if timeSince < 7*24*time.Hour {
			lastUsed = fmt.Sprintf("📆 %s", item.LastUsed.Format("Mon"))
		} else if timeSince < 30*24*time.Hour {
			lastUsed = fmt.Sprintf("📊 %s", item.LastUsed.Format("Jan 02"))
		} else {
			lastUsed = "💤 old"
		}
	} else {
		lastUsed = "❓ unknown"
	}

	// Get directory name with enhanced path handling
	dirName := item.Path
	if strings.Contains(dirName, "/") {
		parts := strings.Split(dirName, "/")
		dirName = parts[len(parts)-1]
	}
	if strings.Contains(dirName, "\\") {
		parts := strings.Split(dirName, "\\")
		dirName = parts[len(parts)-1]
	}
	if dirName == "" {
		dirName = "/"
	}

	// Add current directory indicator
	var currentIndicator string
	if item.Path == m.currentDir {
		currentIndicator = "👉 "
		dirName = dirName + " (current)"
	}

	// Calculate available width accounting for all components
	usedWidth := len(indent) + len(treeIndicator) + len(currentIndicator) + len(countStr) + len(lastUsed) + 6 // spaces and separators
	maxNameWidth := m.width - usedWidth
	if maxNameWidth < 10 {
		maxNameWidth = 10
	}

	// Truncate name if needed
	if len(dirName) > maxNameWidth {
		dirName = dirName[:maxNameWidth-3] + "..."
	}

	// Create enhanced line with better spacing
	line := fmt.Sprintf("%s%s%s%-*s %s %s",
		indent, treeIndicator, currentIndicator, maxNameWidth, dirName, countStr, lastUsed)

	// Apply enhanced styling based on context
	if selected {
		// Enhanced selection highlighting with tree context
		return selectedStyle.Render("▶ " + line)
	} else if item.Path == m.currentDir {
		// Highlight current directory
		return breadcrumbStyle.Render("  " + line)
	} else if !item.IsActive {
		// Dim inactive directories
		return dimStyle.Render("  " + line)
	} else if item.CommandCount > 50 {
		// Highlight very active directories
		return lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("  " + line) // Bright green
	} else if item.CommandCount > 10 {
		// Highlight moderately active directories
		return lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("  " + line) // Light green
	}

	return normalStyle.Render("  " + line)
}
