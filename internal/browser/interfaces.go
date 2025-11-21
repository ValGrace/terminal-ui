package browser

import (
	"github.com/ValGrace/command-history-tracker/pkg/history"
	tea "github.com/charmbracelet/bubbletea"
)

// Browser implements the HistoryBrowser interface
type Browser struct {
	currentDir string
	storage    history.StorageEngine
}

// NewBrowser creates a new history browser
func NewBrowser(storage history.StorageEngine) *Browser {
	return &Browser{
		storage: storage,
	}
}

// ShowDirectoryHistory displays commands for a specific directory using terminal UI
func (b *Browser) ShowDirectoryHistory(dir string) error {
	b.currentDir = dir

	// Create and run the terminal UI
	model := NewUIModel(b.storage, dir)
	program := tea.NewProgram(model, tea.WithAltScreen())

	_, err := program.Run()
	return err
}

// ShowDirectoryTree displays directory tree with command counts using terminal UI
func (b *Browser) ShowDirectoryTree() error {
	// Create UI model in directory tree view mode
	model := NewUIModel(b.storage, b.currentDir)
	model.viewMode = DirectoryTreeView

	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err := program.Run()
	return err
}

// SelectCommand allows user to select a command interactively
func (b *Browser) SelectCommand() (*history.CommandRecord, error) {
	// Create UI model for command selection
	model := NewUIModel(b.storage, b.currentDir)

	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return nil, err
	}

	// Extract selected command from final model
	if uiModel, ok := finalModel.(UIModel); ok {
		return uiModel.GetSelectedCommand(), nil
	}

	return nil, nil
}

// FilterCommands applies search filter to displayed commands
func (b *Browser) FilterCommands(pattern string) error {
	// Create UI model in search mode
	model := NewUIModel(b.storage, b.currentDir)
	model.searchMode = true
	model.searchQuery = pattern
	filteredModel := model.filterCommands()
	model = &filteredModel

	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err := program.Run()
	return err
}

// SetCurrentDirectory changes the current directory context
func (b *Browser) SetCurrentDirectory(dir string) error {
	b.currentDir = dir
	return nil
}
