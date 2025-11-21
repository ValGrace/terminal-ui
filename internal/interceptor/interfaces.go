package interceptor

import (
	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"github.com/ValGrace/command-history-tracker/pkg/shell"
	"fmt"
)

// Interceptor implements the CommandInterceptor interface
type Interceptor struct {
	recording  bool
	shell      history.ShellType
	detector   shell.ShellDetector
	integrator shell.ShellIntegrator
	envManager *shell.EnvironmentManager
	processor  *CommandProcessor
}

// NewInterceptor creates a new command interceptor
func NewInterceptor(storage history.StorageEngine, cfg *config.Config) *Interceptor {
	processor := NewCommandProcessor(storage, cfg)

	return &Interceptor{
		recording:  false,
		detector:   shell.NewDetector(),
		integrator: shell.NewIntegrator(),
		envManager: shell.NewEnvironmentManager(),
		processor:  processor,
	}
}

// StartRecording begins command capture
func (i *Interceptor) StartRecording() error {
	if i.recording {
		return nil // Already recording
	}

	// Detect current shell if not already set
	if i.shell == history.Unknown {
		detectedShell, err := i.detector.DetectShell()
		if err != nil {
			return fmt.Errorf("failed to detect shell: %w", err)
		}
		i.shell = detectedShell
	}

	// Setup shell integration
	if err := i.SetupShellIntegration(i.shell); err != nil {
		return fmt.Errorf("failed to setup shell integration: %w", err)
	}

	// Setup capture environment
	if err := i.envManager.SetupCaptureEnvironment(); err != nil {
		return fmt.Errorf("failed to setup capture environment: %w", err)
	}

	i.recording = true
	return nil
}

// StopRecording stops command capture
func (i *Interceptor) StopRecording() error {
	if !i.recording {
		return nil // Not recording
	}

	// Remove shell integration
	if err := i.integrator.RemoveIntegration(i.shell); err != nil {
		return fmt.Errorf("failed to remove shell integration: %w", err)
	}

	// Clear capture environment
	i.envManager.ClearCaptureEnvironment()

	i.recording = false
	return nil
}

// SetupShellIntegration configures shell-specific hooks
func (i *Interceptor) SetupShellIntegration(shell history.ShellType) error {
	if !i.detector.IsShellSupported(shell) {
		return fmt.Errorf("shell %s is not supported on this platform", shell.String())
	}

	// Setup integration for the specified shell
	if err := i.integrator.SetupIntegration(shell); err != nil {
		return fmt.Errorf("failed to setup integration for %s: %w", shell.String(), err)
	}

	i.shell = shell
	return nil
}

// IsRecording returns true if currently recording commands
func (i *Interceptor) IsRecording() bool {
	return i.recording
}

// GetCurrentShell returns the currently configured shell type
func (i *Interceptor) GetCurrentShell() history.ShellType {
	return i.shell
}

// DetectShell detects and returns the current shell type
func (i *Interceptor) DetectShell() (history.ShellType, error) {
	return i.detector.DetectShell()
}

// IsIntegrationActive checks if shell integration is currently active
func (i *Interceptor) IsIntegrationActive() (bool, error) {
	if i.shell == history.Unknown {
		return false, fmt.Errorf("no shell configured")
	}
	return i.integrator.IsIntegrationActive(i.shell)
}

// GetCommandFromEnvironment extracts command information from environment variables
func (i *Interceptor) GetCommandFromEnvironment() (*history.CommandRecord, error) {
	return i.envManager.GetCommandFromEnvironment()
}

// ProcessCommand processes a command from the current environment
func (i *Interceptor) ProcessCommand() error {
	if !i.recording {
		return fmt.Errorf("interceptor is not recording")
	}
	return i.processor.ProcessCommand()
}

// ProcessCommandFromArgs processes a command from command line arguments
func (i *Interceptor) ProcessCommandFromArgs(args []string) error {
	return i.processor.ProcessCommandFromArgs(args)
}

// ProcessCommandInteractive processes a command in interactive mode
func (i *Interceptor) ProcessCommandInteractive(command string) error {
	return i.processor.ProcessCommandInteractive(command)
}

// GetStatus returns the current status of the interceptor
func (i *Interceptor) GetStatus() (*ProcessorStatus, error) {
	return i.processor.GetProcessorStatus()
}

// ValidateCommand validates that a command should be processed
func (i *Interceptor) ValidateCommand(command string) error {
	return i.processor.ValidateCommand(command)
}

// GetRecentCommands returns recently captured commands
func (i *Interceptor) GetRecentCommands(limit int) ([]history.CommandRecord, error) {
	return i.processor.GetRecentCommands(limit)
}

// CleanupOldCommands removes old commands based on retention policy
func (i *Interceptor) CleanupOldCommands() error {
	return i.processor.CleanupOldCommands()
}
