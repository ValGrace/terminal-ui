package executor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ValGrace/command-history-tracker/pkg/history"
)

// MockLogger implements ExecutionLogger for testing
type MockLogger struct {
	commands []history.CommandRecord
}

func (m *MockLogger) SaveCommand(cmd history.CommandRecord) error {
	m.commands = append(m.commands, cmd)
	return nil
}

func (m *MockLogger) GetCommands() []history.CommandRecord {
	return m.commands
}

func TestNewExecutor(t *testing.T) {
	executor := NewExecutor()

	if executor == nil {
		t.Fatal("NewExecutor returned nil")
	}

	if executor.dangerousPatterns == nil {
		t.Error("dangerousPatterns not initialized")
	}

	if executor.confirmRequired == nil {
		t.Error("confirmRequired not initialized")
	}

	if executor.validator == nil {
		t.Error("validator not initialized")
	}
}

func TestNewExecutorWithLogger(t *testing.T) {
	logger := &MockLogger{}
	executor := NewExecutorWithLogger(logger)

	if executor == nil {
		t.Fatal("NewExecutorWithLogger returned nil")
	}

	if executor.storage != logger {
		t.Error("storage logger not set correctly")
	}
}

func TestValidateCommand_NilCommand(t *testing.T) {
	executor := NewExecutor()

	err := executor.ValidateCommand(nil)
	if err == nil {
		t.Error("expected error for nil command, got nil")
	}
}

func TestValidateCommand_EmptyCommand(t *testing.T) {
	executor := NewExecutor()

	cmd := &history.CommandRecord{
		Command: "",
	}

	err := executor.ValidateCommand(cmd)
	if err == nil {
		t.Error("expected error for empty command, got nil")
	}
}

func TestValidateCommand_SafeCommand(t *testing.T) {
	executor := NewExecutor()

	cmd := &history.CommandRecord{
		Command:   "echo hello",
		Directory: "/tmp",
		Shell:     history.Bash,
	}

	err := executor.ValidateCommand(cmd)
	if err != nil {
		t.Errorf("expected no error for safe command, got: %v", err)
	}
}

func TestValidateCommand_DangerousCommand(t *testing.T) {
	executor := NewExecutor()

	dangerousCommands := []string{
		"rm -rf /",
		"rm -rf /*",
		":(){ :|:& };:",
	}

	for _, cmdStr := range dangerousCommands {
		cmd := &history.CommandRecord{
			Command:   cmdStr,
			Directory: "/tmp",
			Shell:     history.Bash,
		}

		err := executor.ValidateCommand(cmd)
		if err == nil {
			t.Errorf("expected error for dangerous command '%s', got nil", cmdStr)
		}
	}
}

func TestPreviewCommand(t *testing.T) {
	executor := NewExecutor()

	cmd := &history.CommandRecord{
		ID:        "test-123",
		Command:   "echo test",
		Directory: "/tmp",
		Shell:     history.Bash,
		Timestamp: time.Now(),
		ExitCode:  0,
	}

	preview := executor.PreviewCommand(cmd)

	if preview == "" {
		t.Error("preview should not be empty")
	}

	if len(preview) < 10 {
		t.Error("preview seems too short")
	}
}

func TestPreviewCommand_NilCommand(t *testing.T) {
	executor := NewExecutor()

	preview := executor.PreviewCommand(nil)

	if preview != "" {
		t.Error("preview should be empty for nil command")
	}
}

func TestRequiresConfirmation(t *testing.T) {
	executor := NewExecutor()

	tests := []struct {
		command  string
		expected bool
	}{
		{"echo hello", false},
		{"ls -la", false},
		{"rm -rf /tmp/test", true},
		{"rm -r /tmp/test", true},
		{"git push --force", true},
		{"git reset --hard", true},
	}

	for _, tt := range tests {
		result := executor.RequiresConfirmation(tt.command)
		if result != tt.expected {
			t.Errorf("RequiresConfirmation(%q) = %v, want %v", tt.command, result, tt.expected)
		}
	}
}

func TestIsDangerous(t *testing.T) {
	executor := NewExecutor()

	tests := []struct {
		command  string
		expected bool
	}{
		{"echo hello", false},
		{"ls -la", false},
		{"rm -rf /", true},
		{"rm -rf /*", true},
		{":(){ :|:& };:", true},
	}

	for _, tt := range tests {
		result := executor.IsDangerous(tt.command)
		if result != tt.expected {
			t.Errorf("IsDangerous(%q) = %v, want %v", tt.command, result, tt.expected)
		}
	}
}

func TestExecuteInDirectory_SafeCommand(t *testing.T) {
	executor := NewExecutor()

	// Create a temporary directory
	tmpDir := t.TempDir()

	// Execute a safe command
	result, err := executor.ExecuteInDirectory("echo test", tmpDir, history.Bash)

	if err != nil {
		t.Errorf("expected no error for safe command, got: %v", err)
	}

	if result == nil {
		t.Fatal("result should not be nil")
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	if result.Duration <= 0 {
		t.Error("duration should be positive")
	}
}

func TestExecuteInDirectory_InvalidDirectory(t *testing.T) {
	executor := NewExecutor()

	// Try to execute in non-existent directory
	_, err := executor.ExecuteInDirectory("echo test", "/nonexistent/directory/path", history.Bash)

	if err == nil {
		t.Error("expected error for non-existent directory, got nil")
	}
}

func TestExecuteInDirectory_DangerousCommand(t *testing.T) {
	executor := NewExecutor()

	tmpDir := t.TempDir()

	// Try to execute a dangerous command
	_, err := executor.ExecuteInDirectory("rm -rf /", tmpDir, history.Bash)

	if err == nil {
		t.Error("expected error for dangerous command, got nil")
	}
}

func TestGenerateCommandID(t *testing.T) {
	id1 := generateCommandID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateCommandID()

	if id1 == "" {
		t.Error("generated ID should not be empty")
	}

	if id1 == id2 {
		t.Error("generated IDs should be unique")
	}
}

func TestIsWindows(t *testing.T) {
	result := isWindows()

	// Just verify it returns a boolean without error
	if result != (os.PathSeparator == '\\') {
		t.Error("isWindows() returned unexpected result")
	}
}

func TestExecutorWithValidator(t *testing.T) {
	executor := NewExecutor()
	validator := NewCommandValidator()

	executor.SetValidator(validator)

	if executor.GetValidator() != validator {
		t.Error("validator not set correctly")
	}
}

func TestExecutorWithAuditLogger(t *testing.T) {
	executor := NewExecutor()

	// Create temporary audit log
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	auditLogger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("failed to create audit logger: %v", err)
	}
	defer auditLogger.Close()

	executor.SetAuditLogger(auditLogger)

	if executor.GetAuditLogger() != auditLogger {
		t.Error("audit logger not set correctly")
	}
}

func TestExecutorClose(t *testing.T) {
	executor := NewExecutor()

	// Should not error even without audit logger
	err := executor.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// With audit logger
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	auditLogger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("failed to create audit logger: %v", err)
	}

	executor.SetAuditLogger(auditLogger)

	err = executor.Close()
	if err != nil {
		t.Errorf("Close() with audit logger returned error: %v", err)
	}
}
