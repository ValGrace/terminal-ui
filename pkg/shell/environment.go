package shell

import (
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// EnvironmentManager handles environment variable setup for command capture
type EnvironmentManager struct{}

// NewEnvironmentManager creates a new environment manager
func NewEnvironmentManager() *EnvironmentManager {
	return &EnvironmentManager{}
}

// SetupCaptureEnvironment sets up environment variables for command capture
func (e *EnvironmentManager) SetupCaptureEnvironment() error {
	// Set the tracker executable path in environment
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	return os.Setenv("CHT_TRACKER_PATH", execPath)
}

// GetCommandFromEnvironment extracts command information from environment variables
func (e *EnvironmentManager) GetCommandFromEnvironment() (*history.CommandRecord, error) {
	command := os.Getenv("CHT_COMMAND")
	if command == "" {
		return nil, fmt.Errorf("CHT_COMMAND environment variable not set")
	}

	// Get directory with fallback to current working directory
	directory := os.Getenv("CHT_DIRECTORY")
	if directory == "" {
		if currentDir, err := os.Getwd(); err == nil {
			directory = currentDir
		} else {
			return nil, fmt.Errorf("CHT_DIRECTORY environment variable not set and cannot determine current directory")
		}
	}

	// Get shell with fallback to detection
	shellStr := os.Getenv("CHT_SHELL")
	var shell history.ShellType
	var err error
	if shellStr != "" {
		shell, err = e.parseShellType(shellStr)
		if err != nil {
			return nil, fmt.Errorf("invalid shell type: %w", err)
		}
	} else {
		// Try to detect shell if not provided
		detector := NewDetector()
		shell, err = detector.DetectShell()
		if err != nil {
			shell = history.Unknown // Default to unknown if detection fails
		}
	}

	// Parse timestamp with multiple format support
	timestamp := e.parseTimestamp(os.Getenv("CHT_TIMESTAMP"))

	// Parse exit code with validation
	exitCode := e.parseExitCode(os.Getenv("CHT_EXIT_CODE"))

	// Parse duration with validation
	duration := e.parseDuration(os.Getenv("CHT_DURATION"))

	// Generate a unique ID for the command
	id := e.GenerateCommandID(command, directory, timestamp)

	return &history.CommandRecord{
		ID:        id,
		Command:   command,
		Directory: directory,
		Timestamp: timestamp,
		Shell:     shell,
		ExitCode:  exitCode,
		Duration:  duration,
		Tags:      []string{},
	}, nil
}

// parseTimestamp parses timestamp from string with multiple format support
func (e *EnvironmentManager) parseTimestamp(timestampStr string) time.Time {
	if timestampStr == "" {
		return time.Now()
	}

	// Try multiple timestamp formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"1136239445", // Unix timestamp
	}

	for _, format := range formats {
		if format == "1136239445" {
			// Handle Unix timestamp
			if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
				return time.Unix(timestamp, 0)
			}
		} else {
			if parsed, err := time.Parse(format, timestampStr); err == nil {
				return parsed
			}
		}
	}

	// If all parsing fails, return current time
	return time.Now()
}

// parseExitCode parses exit code from string with validation
func (e *EnvironmentManager) parseExitCode(exitCodeStr string) int {
	if exitCodeStr == "" {
		return 0
	}

	if exitCode, err := strconv.Atoi(exitCodeStr); err == nil {
		// Validate exit code range (typically 0-255 on Unix systems)
		if exitCode >= 0 && exitCode <= 255 {
			return exitCode
		}
	}

	return 0 // Default to success if parsing fails
}

// parseDuration parses duration from string with multiple unit support
func (e *EnvironmentManager) parseDuration(durationStr string) time.Duration {
	if durationStr == "" {
		return 0
	}

	// Try parsing as milliseconds (default)
	if ms, err := strconv.ParseInt(durationStr, 10, 64); err == nil {
		return time.Duration(ms) * time.Millisecond
	}

	// Try parsing as Go duration string (e.g., "1.5s", "100ms")
	if duration, err := time.ParseDuration(durationStr); err == nil {
		return duration
	}

	return 0 // Default to zero if parsing fails
}

// ClearCaptureEnvironment clears command capture environment variables
func (e *EnvironmentManager) ClearCaptureEnvironment() {
	envVars := []string{
		"CHT_COMMAND",
		"CHT_DIRECTORY",
		"CHT_SHELL",
		"CHT_TIMESTAMP",
		"CHT_EXIT_CODE",
		"CHT_DURATION",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}

// SetCaptureEnvironment sets environment variables for command capture
func (e *EnvironmentManager) SetCaptureEnvironment(cmd *history.CommandRecord) error {
	envVars := map[string]string{
		"CHT_COMMAND":   cmd.Command,
		"CHT_DIRECTORY": cmd.Directory,
		"CHT_SHELL":     cmd.Shell.String(),
		"CHT_TIMESTAMP": cmd.Timestamp.Format(time.RFC3339),
		"CHT_EXIT_CODE": strconv.Itoa(cmd.ExitCode),
		"CHT_DURATION":  strconv.FormatInt(int64(cmd.Duration/time.Millisecond), 10),
	}

	for key, value := range envVars {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", key, err)
		}
	}

	return nil
}

// IsTrackerEnabled checks if command tracking is enabled via environment
func (e *EnvironmentManager) IsTrackerEnabled() bool {
	// Check if tracking is explicitly disabled
	if disabled := os.Getenv("CHT_DISABLED"); disabled == "1" || disabled == "true" {
		return false
	}

	// Check if tracker path is available
	trackerPath := os.Getenv("CHT_TRACKER_PATH")
	if trackerPath == "" {
		// If not explicitly set, try to locate the 'tracker' executable on PATH
		if p, err := exec.LookPath("tracker"); err == nil {
			// Cache discovered path for subsequent checks
			_ = os.Setenv("CHT_TRACKER_PATH", p)
			return true
		}
		return false
	}

	// Check if tracker executable exists
	if _, err := os.Stat(trackerPath); err != nil {
		return false
	}

	return true
}

// GetTrackerPath returns the path to the tracker executable
func (e *EnvironmentManager) GetTrackerPath() string {
	return os.Getenv("CHT_TRACKER_PATH")
}

// parseShellType converts string to ShellType
func (e *EnvironmentManager) parseShellType(shellStr string) (history.ShellType, error) {
	switch shellStr {
	case "unknown":
		return history.Unknown, nil
	case "powershell":
		return history.PowerShell, nil
	case "bash":
		return history.Bash, nil
	case "zsh":
		return history.Zsh, nil
	case "cmd":
		return history.Cmd, nil
	default:
		return history.Unknown, fmt.Errorf("unknown shell type: %s", shellStr)
	}
}

// GenerateCommandID creates a unique identifier for a command
func (e *EnvironmentManager) GenerateCommandID(command, directory string, timestamp time.Time) string {
	// Create a simple hash-like ID based on command content and timestamp
	hash := 0
	for _, char := range command + directory + timestamp.Format(time.RFC3339Nano) {
		hash = hash*31 + int(char)
	}

	// Convert to positive number and format as hex
	if hash < 0 {
		hash = -hash
	}

	return fmt.Sprintf("cmd_%x_%d", hash, timestamp.Unix())
}

// ValidateEnvironment checks if all required environment variables are set
func (e *EnvironmentManager) ValidateEnvironment() error {
	requiredVars := []string{
		"CHT_COMMAND",
		"CHT_DIRECTORY",
		"CHT_SHELL",
	}

	for _, envVar := range requiredVars {
		if value := os.Getenv(envVar); value == "" {
			return fmt.Errorf("required environment variable %s is not set", envVar)
		}
	}

	return nil
}

// GetEnvironmentInfo returns current environment information for debugging
func (e *EnvironmentManager) GetEnvironmentInfo() map[string]string {
	envVars := []string{
		"CHT_COMMAND",
		"CHT_DIRECTORY",
		"CHT_SHELL",
		"CHT_TIMESTAMP",
		"CHT_EXIT_CODE",
		"CHT_DURATION",
		"CHT_TRACKER_PATH",
		"CHT_DISABLED",
	}

	info := make(map[string]string)
	for _, envVar := range envVars {
		info[envVar] = os.Getenv(envVar)
	}

	return info
}
