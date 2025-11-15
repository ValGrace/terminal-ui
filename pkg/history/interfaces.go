package history

import (
	"time"
)

// ShellType represents different shell environments
type ShellType int

const (
	Unknown ShellType = iota
	PowerShell
	Bash
	Zsh
	Cmd
)

// String returns the string representation of ShellType
func (s ShellType) String() string {
	switch s {
	case Unknown:
		return "unknown"
	case PowerShell:
		return "powershell"
	case Bash:
		return "bash"
	case Zsh:
		return "zsh"
	case Cmd:
		return "cmd"
	default:
		return "unknown"
	}
}

// MarshalJSON implements json.Marshaler
func (s ShellType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler
func (s *ShellType) UnmarshalJSON(data []byte) error {
	str := string(data)
	// Remove quotes
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	switch str {
	case "unknown":
		*s = Unknown
	case "powershell":
		*s = PowerShell
	case "bash":
		*s = Bash
	case "zsh":
		*s = Zsh
	case "cmd":
		*s = Cmd
	default:
		*s = Unknown
	}

	return nil
}

// CommandRecord represents a stored command with metadata
type CommandRecord struct {
	ID        string        `json:"id" db:"id"`
	Command   string        `json:"command" db:"command"`
	Directory string        `json:"directory" db:"directory"`
	Timestamp time.Time     `json:"timestamp" db:"timestamp"`
	Shell     ShellType     `json:"shell" db:"shell"`
	ExitCode  int           `json:"exit_code" db:"exit_code"`
	Duration  time.Duration `json:"duration" db:"duration"`
	Tags      []string      `json:"tags" db:"tags"`
}

// CommandInterceptor handles capturing commands from shell environments
type CommandInterceptor interface {
	// StartRecording begins command capture for the specified shell
	StartRecording() error

	// StopRecording stops command capture
	StopRecording() error

	// SetupShellIntegration configures shell-specific hooks
	SetupShellIntegration(shell ShellType) error

	// IsRecording returns true if currently recording commands
	IsRecording() bool
}

// StorageEngine handles persistence and retrieval of command history
type StorageEngine interface {
	// SaveCommand stores a command record
	SaveCommand(cmd CommandRecord) error

	// GetCommandsByDirectory retrieves commands for a specific directory
	GetCommandsByDirectory(dir string) ([]CommandRecord, error)

	// GetDirectoriesWithHistory returns all directories that have command history
	GetDirectoriesWithHistory() ([]string, error)

	// CleanupOldCommands removes commands older than specified retention period
	CleanupOldCommands(retentionDays int) error

	// SearchCommands finds commands matching a pattern
	SearchCommands(pattern string, dir string) ([]CommandRecord, error)

	// Close closes the storage connection
	Close() error
}

// HistoryBrowser provides interactive interface for command navigation
type HistoryBrowser interface {
	// ShowDirectoryHistory displays commands for a specific directory
	ShowDirectoryHistory(dir string) error

	// ShowDirectoryTree displays directory tree with command counts
	ShowDirectoryTree() error

	// SelectCommand allows user to select a command interactively
	SelectCommand() (*CommandRecord, error)

	// FilterCommands applies search filter to displayed commands
	FilterCommands(pattern string) error

	// SetCurrentDirectory changes the current directory context
	SetCurrentDirectory(dir string) error
}

// CommandExecutor handles safe execution of selected commands
type CommandExecutor interface {
	// ExecuteCommand runs a command in the specified directory context
	ExecuteCommand(cmd *CommandRecord, currentDir string) error

	// ValidateCommand checks if a command is safe to execute
	ValidateCommand(cmd *CommandRecord) error

	// PreviewCommand returns a preview of what will be executed
	PreviewCommand(cmd *CommandRecord) string

	// ConfirmExecution prompts user for confirmation before execution
	ConfirmExecution(cmd *CommandRecord) (bool, error)
}

// DirectoryIndex represents directory metadata for command history
type DirectoryIndex struct {
	Path         string    `json:"path"`
	CommandCount int       `json:"command_count"`
	LastUsed     time.Time `json:"last_used"`
	IsActive     bool      `json:"is_active"`
}

// Validate checks if the CommandRecord has valid data
func (c *CommandRecord) Validate() error {
	if c.ID == "" {
		return &ValidationError{Field: "ID", Message: "ID cannot be empty"}
	}
	if c.Command == "" {
		return &ValidationError{Field: "Command", Message: "Command cannot be empty"}
	}
	if c.Directory == "" {
		return &ValidationError{Field: "Directory", Message: "Directory cannot be empty"}
	}
	if c.Timestamp.IsZero() {
		return &ValidationError{Field: "Timestamp", Message: "Timestamp cannot be zero"}
	}
	if c.Shell <= Unknown || c.Shell > Cmd {
		return &ValidationError{Field: "Shell", Message: "Invalid shell type"}
	}
	if c.Duration < 0 {
		return &ValidationError{Field: "Duration", Message: "Duration cannot be negative"}
	}
	return nil
}

// IsEmpty checks if the CommandRecord is empty/uninitialized
func (c *CommandRecord) IsEmpty() bool {
	return c.ID == "" && c.Command == "" && c.Directory == ""
}

// HasTag checks if the CommandRecord has a specific tag
func (c *CommandRecord) HasTag(tag string) bool {
	for _, t := range c.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// AddTag adds a tag to the CommandRecord if it doesn't already exist
func (c *CommandRecord) AddTag(tag string) {
	if !c.HasTag(tag) {
		c.Tags = append(c.Tags, tag)
	}
}

// RemoveTag removes a tag from the CommandRecord
func (c *CommandRecord) RemoveTag(tag string) {
	for i, t := range c.Tags {
		if t == tag {
			c.Tags = append(c.Tags[:i], c.Tags[i+1:]...)
			break
		}
	}
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
