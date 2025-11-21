package interceptor

import (
	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"github.com/ValGrace/command-history-tracker/pkg/shell"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// CommandProcessor handles the processing of intercepted commands
type CommandProcessor struct {
	capture    *CommandCapture
	storage    history.StorageEngine
	config     *config.Config
	envManager *shell.EnvironmentManager
	detector   shell.ShellDetector
}

// NewCommandProcessor creates a new command processor
func NewCommandProcessor(storage history.StorageEngine, cfg *config.Config) *CommandProcessor {
	capture := NewCommandCapture(storage, cfg)

	return &CommandProcessor{
		capture:    capture,
		storage:    storage,
		config:     cfg,
		envManager: shell.NewEnvironmentManager(),
		detector:   shell.NewDetector(),
	}
}

// ProcessCommand processes a command from the current environment
func (p *CommandProcessor) ProcessCommand() error {
	return p.capture.CaptureCommand()
}

// ProcessCommandFromArgs processes a command from command line arguments
func (p *CommandProcessor) ProcessCommandFromArgs(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("insufficient arguments for command processing")
	}

	// Parse command line arguments
	cmdInfo, err := p.parseCommandArgs(args)
	if err != nil {
		return fmt.Errorf("failed to parse command arguments: %w", err)
	}

	// Process the command
	return p.capture.CaptureCommandDirect(
		cmdInfo.Command,
		cmdInfo.Directory,
		cmdInfo.Shell,
		cmdInfo.ExitCode,
		cmdInfo.Duration,
	)
}

// ProcessCommandFromEnvironment processes a command using environment variables
func (p *CommandProcessor) ProcessCommandFromEnvironment() error {
	// Check if we have the required environment variables
	if err := p.envManager.ValidateEnvironment(); err != nil {
		return fmt.Errorf("invalid environment for command processing: %w", err)
	}

	// Enhance environment with additional context if missing
	if err := p.enhanceEnvironmentContext(); err != nil {
		// Non-fatal error, log but continue
		// In a real implementation, this would use a proper logger
	}

	return p.capture.CaptureCommand()
}

// enhanceEnvironmentContext adds missing context to environment variables
func (p *CommandProcessor) enhanceEnvironmentContext() error {
	// Ensure directory is set
	if os.Getenv("CHT_DIRECTORY") == "" {
		if currentDir, err := os.Getwd(); err == nil {
			os.Setenv("CHT_DIRECTORY", currentDir)
		}
	}

	// Ensure timestamp is set
	if os.Getenv("CHT_TIMESTAMP") == "" {
		os.Setenv("CHT_TIMESTAMP", time.Now().Format(time.RFC3339))
	}

	// Ensure shell is detected if not set
	if os.Getenv("CHT_SHELL") == "" {
		if shell, err := p.detector.DetectShell(); err == nil {
			os.Setenv("CHT_SHELL", shell.String())
		}
	}

	return nil
}

// ProcessCommandInteractive processes a command in interactive mode
func (p *CommandProcessor) ProcessCommandInteractive(command string) error {
	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Detect current shell
	shell, err := p.detector.DetectShell()
	if err != nil {
		shell = history.Unknown // Default to unknown if detection fails
	}

	// Execute the command and capture its exit code
	exitCode, duration := p.executeAndMeasure(command)

	// Process the command
	return p.capture.CaptureCommandDirect(
		command,
		currentDir,
		shell,
		exitCode,
		duration,
	)
}

// SetupCommandHooks sets up hooks for automatic command capture
func (p *CommandProcessor) SetupCommandHooks(shellType history.ShellType) error {
	integrator := shell.NewIntegrator()

	// Setup shell integration
	if err := integrator.SetupIntegration(shellType); err != nil {
		return fmt.Errorf("failed to setup shell integration: %w", err)
	}

	// Setup environment for capture
	if err := p.envManager.SetupCaptureEnvironment(); err != nil {
		return fmt.Errorf("failed to setup capture environment: %w", err)
	}

	return nil
}

// RemoveCommandHooks removes automatic command capture hooks
func (p *CommandProcessor) RemoveCommandHooks(shellType history.ShellType) error {
	integrator := shell.NewIntegrator()

	// Remove shell integration
	if err := integrator.RemoveIntegration(shellType); err != nil {
		return fmt.Errorf("failed to remove shell integration: %w", err)
	}

	// Clear capture environment
	p.envManager.ClearCaptureEnvironment()

	return nil
}

// GetProcessorStatus returns the current status of the processor
func (p *CommandProcessor) GetProcessorStatus() (*ProcessorStatus, error) {
	// Check if tracking is enabled
	trackingEnabled := p.envManager.IsTrackerEnabled()

	// Get capture statistics
	stats, err := p.capture.GetCaptureStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get capture stats: %w", err)
	}

	// Detect current shell
	currentShell, err := p.detector.DetectShell()
	if err != nil {
		currentShell = history.Unknown
	}

	// Check integration status
	integrator := shell.NewIntegrator()
	integrationActive := false
	if currentShell != history.Unknown {
		integrationActive, _ = integrator.IsIntegrationActive(currentShell)
	}

	return &ProcessorStatus{
		TrackingEnabled:   trackingEnabled,
		CurrentShell:      currentShell,
		IntegrationActive: integrationActive,
		TotalDirectories:  stats.TotalDirectories,
		TotalCommands:     stats.TotalCommands,
		TrackerPath:       p.envManager.GetTrackerPath(),
	}, nil
}

// CommandInfo represents parsed command information
type CommandInfo struct {
	Command   string
	Directory string
	Shell     history.ShellType
	ExitCode  int
	Duration  time.Duration
	Timestamp time.Time
}

// ProcessorStatus represents the current status of the command processor
type ProcessorStatus struct {
	TrackingEnabled   bool              `json:"tracking_enabled"`
	CurrentShell      history.ShellType `json:"current_shell"`
	IntegrationActive bool              `json:"integration_active"`
	TotalDirectories  int               `json:"total_directories"`
	TotalCommands     int               `json:"total_commands"`
	TrackerPath       string            `json:"tracker_path"`
}

// parseCommandArgs parses command line arguments into CommandInfo
func (p *CommandProcessor) parseCommandArgs(args []string) (*CommandInfo, error) {
	cmdInfo := &CommandInfo{
		Timestamp: time.Now(),
		ExitCode:  0,
		Duration:  0,
		Shell:     history.Unknown,
	}

	// Parse arguments in key=value format
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]

		switch key {
		case "command":
			cmdInfo.Command = value
		case "directory", "dir":
			cmdInfo.Directory = value
		case "shell":
			shell, err := p.parseShellType(value)
			if err != nil {
				return nil, fmt.Errorf("invalid shell type: %w", err)
			}
			cmdInfo.Shell = shell
		case "exit_code", "exitcode":
			exitCode, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid exit code: %w", err)
			}
			cmdInfo.ExitCode = exitCode
		case "duration":
			duration, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid duration: %w", err)
			}
			cmdInfo.Duration = time.Duration(duration) * time.Millisecond
		case "timestamp":
			timestamp, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return nil, fmt.Errorf("invalid timestamp: %w", err)
			}
			cmdInfo.Timestamp = timestamp
		}
	}

	// Validate required fields
	if cmdInfo.Command == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Set default directory if not provided
	if cmdInfo.Directory == "" {
		if currentDir, err := os.Getwd(); err == nil {
			cmdInfo.Directory = currentDir
		} else {
			return nil, fmt.Errorf("directory is required and current directory cannot be determined")
		}
	}

	// Detect shell if not provided
	if cmdInfo.Shell == history.Unknown {
		if detectedShell, err := p.detector.DetectShell(); err == nil {
			cmdInfo.Shell = detectedShell
		}
	}

	return cmdInfo, nil
}

// parseShellType converts string to ShellType
func (p *CommandProcessor) parseShellType(shellStr string) (history.ShellType, error) {
	switch strings.ToLower(shellStr) {
	case "unknown":
		return history.Unknown, nil
	case "powershell", "pwsh":
		return history.PowerShell, nil
	case "bash":
		return history.Bash, nil
	case "zsh":
		return history.Zsh, nil
	case "cmd", "command":
		return history.Cmd, nil
	default:
		return history.Unknown, fmt.Errorf("unknown shell type: %s", shellStr)
	}
}

// executeAndMeasure executes a command and measures its execution time and exit code
func (p *CommandProcessor) executeAndMeasure(command string) (int, time.Duration) {
	// This is a placeholder implementation
	// In a real implementation, this would execute the command and capture metrics
	// For now, we'll return default values
	return 0, 0
}

// ValidateCommand validates that a command should be processed
func (p *CommandProcessor) ValidateCommand(command string) error {
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Check against exclude patterns
	if p.config != nil {
		for _, pattern := range p.config.ExcludePatterns {
			if matched, _ := filepath.Match(pattern, command); matched {
				return fmt.Errorf("command matches exclude pattern: %s", pattern)
			}
		}
	}

	return nil
}

// GetRecentCommands returns recently captured commands
func (p *CommandProcessor) GetRecentCommands(limit int) ([]history.CommandRecord, error) {
	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Normalize directory path to match how commands are stored
	normalizedDir, err := p.normalizeDirectory(currentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize directory: %w", err)
	}

	// Get commands for current directory
	commands, err := p.storage.GetCommandsByDirectory(normalizedDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get commands: %w", err)
	}

	// Limit results if requested
	if limit > 0 && len(commands) > limit {
		commands = commands[:limit]
	}

	return commands, nil
}

// normalizeDirectory normalizes the directory path for consistent storage
func (p *CommandProcessor) normalizeDirectory(dir string) (string, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return dir, err // Return original if conversion fails
	}

	// Clean the path (remove redundant elements)
	cleanPath := filepath.Clean(absPath)

	// Convert to forward slashes for consistency across platforms
	normalizedPath := filepath.ToSlash(cleanPath)

	return normalizedPath, nil
}

// CleanupOldCommands removes old commands based on retention policy
func (p *CommandProcessor) CleanupOldCommands() error {
	retentionDays := 30 // Default retention
	if p.config != nil && p.config.RetentionDays > 0 {
		retentionDays = p.config.RetentionDays
	}

	return p.storage.CleanupOldCommands(retentionDays)
}
