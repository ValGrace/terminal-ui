package shell

import "github.com/ValGrace/command-history-tracker/pkg/history"

// ShellDetector identifies the current shell environment
type ShellDetector interface {
	// DetectShell identifies the current shell type
	DetectShell() (history.ShellType, error)

	// GetShellPath returns the path to the shell executable
	GetShellPath(shell history.ShellType) (string, error)

	// IsShellSupported checks if a shell type is supported
	IsShellSupported(shell history.ShellType) bool
}

// ShellIntegrator handles shell-specific integration setup
type ShellIntegrator interface {
	// SetupIntegration configures shell hooks for command capture
	SetupIntegration(shell history.ShellType) error

	// RemoveIntegration removes shell hooks
	RemoveIntegration(shell history.ShellType) error

	// GetIntegrationScript returns the script content for shell integration
	GetIntegrationScript(shell history.ShellType) (string, error)

	// IsIntegrationActive checks if integration is currently active
	IsIntegrationActive(shell history.ShellType) (bool, error)
}
