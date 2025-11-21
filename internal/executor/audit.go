package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ValGrace/command-history-tracker/pkg/history"
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Command   string            `json:"command"`
	Directory string            `json:"directory"`
	Shell     history.ShellType `json:"shell"`
	ExitCode  int               `json:"exit_code"`
	Duration  time.Duration     `json:"duration"`
	User      string            `json:"user"`
	Status    string            `json:"status"` // success, failed, blocked, cancelled
	Reason    string            `json:"reason,omitempty"`
	Validated bool              `json:"validated"`
	Confirmed bool              `json:"confirmed"`
}

// AuditLogger handles logging of command executions for security auditing
type AuditLogger struct {
	logPath string
	mu      sync.Mutex
	file    *os.File
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logPath string) (*AuditLogger, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	// Open log file in append mode
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	return &AuditLogger{
		logPath: logPath,
		file:    file,
	}, nil
}

// DefaultAuditLogger creates an audit logger with default path
func DefaultAuditLogger() (*AuditLogger, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	logPath := filepath.Join(homeDir, ".command-history-tracker", "audit.log")
	return NewAuditLogger(logPath)
}

// LogExecution logs a command execution
func (a *AuditLogger) LogExecution(entry AuditEntry) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Set timestamp if not set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Get current user
	if entry.User == "" {
		if user := os.Getenv("USER"); user != "" {
			entry.User = user
		} else if user := os.Getenv("USERNAME"); user != "" {
			entry.User = user
		} else {
			entry.User = "unknown"
		}
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	// Write to log file
	if _, err := a.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write audit entry: %w", err)
	}

	// Sync to disk
	if err := a.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync audit log: %w", err)
	}

	return nil
}

// LogSuccess logs a successful command execution
func (a *AuditLogger) LogSuccess(cmd *history.CommandRecord, duration time.Duration) error {
	return a.LogExecution(AuditEntry{
		Command:   cmd.Command,
		Directory: cmd.Directory,
		Shell:     cmd.Shell,
		ExitCode:  cmd.ExitCode,
		Duration:  duration,
		Status:    "success",
		Validated: true,
		Confirmed: true,
	})
}

// LogFailure logs a failed command execution
func (a *AuditLogger) LogFailure(cmd *history.CommandRecord, exitCode int, duration time.Duration, reason string) error {
	return a.LogExecution(AuditEntry{
		Command:   cmd.Command,
		Directory: cmd.Directory,
		Shell:     cmd.Shell,
		ExitCode:  exitCode,
		Duration:  duration,
		Status:    "failed",
		Reason:    reason,
		Validated: true,
		Confirmed: true,
	})
}

// LogBlocked logs a blocked command execution
func (a *AuditLogger) LogBlocked(command string, directory string, shell history.ShellType, reason string) error {
	return a.LogExecution(AuditEntry{
		Command:   command,
		Directory: directory,
		Shell:     shell,
		Status:    "blocked",
		Reason:    reason,
		Validated: false,
		Confirmed: false,
	})
}

// LogCancelled logs a cancelled command execution
func (a *AuditLogger) LogCancelled(cmd *history.CommandRecord, reason string) error {
	return a.LogExecution(AuditEntry{
		Command:   cmd.Command,
		Directory: cmd.Directory,
		Shell:     cmd.Shell,
		Status:    "cancelled",
		Reason:    reason,
		Validated: true,
		Confirmed: false,
	})
}

// GetRecentEntries retrieves recent audit log entries
func (a *AuditLogger) GetRecentEntries(limit int) ([]AuditEntry, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Read entire log file
	data, err := os.ReadFile(a.logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []AuditEntry{}, nil
		}
		return nil, fmt.Errorf("failed to read audit log: %w", err)
	}

	// Parse entries
	lines := splitLines(string(data))
	entries := make([]AuditEntry, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		var entry AuditEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip invalid entries
			continue
		}

		entries = append(entries, entry)
	}

	// Return most recent entries
	if limit > 0 && len(entries) > limit {
		return entries[len(entries)-limit:], nil
	}

	return entries, nil
}

// Close closes the audit logger
func (a *AuditLogger) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.file != nil {
		return a.file.Close()
	}
	return nil
}

// splitLines splits a string into lines
func splitLines(s string) []string {
	lines := make([]string, 0)
	start := 0

	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}

	if start < len(s) {
		lines = append(lines, s[start:])
	}

	return lines
}

// RotateLog rotates the audit log file
func (a *AuditLogger) RotateLog() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Close current file
	if a.file != nil {
		if err := a.file.Close(); err != nil {
			return fmt.Errorf("failed to close audit log: %w", err)
		}
	}

	// Rename current log file with timestamp
	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := fmt.Sprintf("%s.%s", a.logPath, timestamp)

	if err := os.Rename(a.logPath, rotatedPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to rotate audit log: %w", err)
		}
	}

	// Open new log file
	file, err := os.OpenFile(a.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open new audit log: %w", err)
	}

	a.file = file
	return nil
}
