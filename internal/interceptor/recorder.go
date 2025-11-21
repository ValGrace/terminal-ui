package interceptor

import (
	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/ValGrace/command-history-tracker/internal/storage"
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"fmt"
)

// CommandRecorder provides a simple interface for recording commands from shell hooks
type CommandRecorder struct {
	processor *CommandProcessor
	storage   history.StorageEngine
}

// NewCommandRecorder creates a new command recorder
func NewCommandRecorder() (*CommandRecorder, error) {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		// Use default config if loading fails
		cfg = config.DefaultConfig()
	}

	// Initialize storage
	storageEngine := storage.NewSQLiteStorage(cfg.StoragePath)
	if err := storageEngine.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Create processor
	processor := NewCommandProcessor(storageEngine, cfg)

	return &CommandRecorder{
		processor: processor,
		storage:   storageEngine,
	}, nil
}

// RecordFromEnvironment records a command using environment variables
func (r *CommandRecorder) RecordFromEnvironment() error {
	return r.processor.ProcessCommandFromEnvironment()
}

// RecordFromArgs records a command using command line arguments
func (r *CommandRecorder) RecordFromArgs(args []string) error {
	return r.processor.ProcessCommandFromArgs(args)
}

// RecordInteractive records a command interactively
func (r *CommandRecorder) RecordInteractive(command string) error {
	return r.processor.ProcessCommandInteractive(command)
}

// Close closes the recorder and releases resources
func (r *CommandRecorder) Close() error {
	if r.storage != nil {
		return r.storage.Close()
	}
	return nil
}

// GetStatus returns the current status of command recording
func (r *CommandRecorder) GetStatus() (*ProcessorStatus, error) {
	return r.processor.GetProcessorStatus()
}

// ValidateEnvironment checks if the environment is properly set up for recording
func (r *CommandRecorder) ValidateEnvironment() error {
	return r.processor.envManager.ValidateEnvironment()
}

// IsEnabled checks if command recording is enabled
func (r *CommandRecorder) IsEnabled() bool {
	return r.processor.envManager.IsTrackerEnabled()
}

// RecordCommand is a convenience function for recording a single command
func RecordCommand() error {
	recorder, err := NewCommandRecorder()
	if err != nil {
		return fmt.Errorf("failed to create recorder: %w", err)
	}
	defer recorder.Close()

	// Check if recording is enabled
	if !recorder.IsEnabled() {
		return nil // Silently skip if disabled
	}

	// Record from environment
	return recorder.RecordFromEnvironment()
}

// RecordCommandWithArgs is a convenience function for recording a command with arguments
func RecordCommandWithArgs(args []string) error {
	recorder, err := NewCommandRecorder()
	if err != nil {
		return fmt.Errorf("failed to create recorder: %w", err)
	}
	defer recorder.Close()

	return recorder.RecordFromArgs(args)
}

// GetRecorderStatus is a convenience function for getting recorder status
func GetRecorderStatus() (*ProcessorStatus, error) {
	recorder, err := NewCommandRecorder()
	if err != nil {
		return nil, fmt.Errorf("failed to create recorder: %w", err)
	}
	defer recorder.Close()

	return recorder.GetStatus()
}

// SetupRecording sets up command recording for the current shell
func SetupRecording() error {
	recorder, err := NewCommandRecorder()
	if err != nil {
		return fmt.Errorf("failed to create recorder: %w", err)
	}
	defer recorder.Close()

	// Get current shell
	status, err := recorder.GetStatus()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Setup hooks for the current shell
	return recorder.processor.SetupCommandHooks(status.CurrentShell)
}

// RemoveRecording removes command recording hooks
func RemoveRecording() error {
	recorder, err := NewCommandRecorder()
	if err != nil {
		return fmt.Errorf("failed to create recorder: %w", err)
	}
	defer recorder.Close()

	// Get current shell
	status, err := recorder.GetStatus()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Remove hooks for the current shell
	return recorder.processor.RemoveCommandHooks(status.CurrentShell)
}

// CleanupRecording performs cleanup of old recorded commands
func CleanupRecording() error {
	recorder, err := NewCommandRecorder()
	if err != nil {
		return fmt.Errorf("failed to create recorder: %w", err)
	}
	defer recorder.Close()

	return recorder.processor.CleanupOldCommands()
}

// TestRecording tests if command recording is working properly
func TestRecording() error {
	recorder, err := NewCommandRecorder()
	if err != nil {
		return fmt.Errorf("failed to create recorder: %w", err)
	}
	defer recorder.Close()

	// Validate environment
	if err := recorder.ValidateEnvironment(); err != nil {
		return fmt.Errorf("environment validation failed: %w", err)
	}

	// Check if enabled
	if !recorder.IsEnabled() {
		return fmt.Errorf("command recording is not enabled")
	}

	// Try to record a test command
	testCommand := "echo 'test command'"
	if err := recorder.RecordInteractive(testCommand); err != nil {
		return fmt.Errorf("failed to record test command: %w", err)
	}

	fmt.Println("Command recording test successful")
	return nil
}

// PrintStatus prints the current recording status to stdout
func PrintStatus() error {
	status, err := GetRecorderStatus()
	if err != nil {
		return err
	}

	fmt.Printf("Command History Tracker Status:\n")
	fmt.Printf("  Tracking Enabled: %v\n", status.TrackingEnabled)
	fmt.Printf("  Current Shell: %s\n", status.CurrentShell.String())
	fmt.Printf("  Integration Active: %v\n", status.IntegrationActive)
	fmt.Printf("  Total Directories: %d\n", status.TotalDirectories)
	fmt.Printf("  Total Commands: %d\n", status.TotalCommands)
	fmt.Printf("  Tracker Path: %s\n", status.TrackerPath)

	return nil
}
