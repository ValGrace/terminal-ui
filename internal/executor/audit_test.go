package executor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ValGrace/command-history-tracker/pkg/history"
)

func TestNewAuditLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer logger.Close()

	if logger == nil {
		t.Fatal("logger should not be nil")
	}

	if logger.logPath != logPath {
		t.Errorf("logPath = %s, want %s", logger.logPath, logPath)
	}

	// Verify file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("audit log file was not created")
	}
}

func TestDefaultAuditLogger(t *testing.T) {
	logger, err := DefaultAuditLogger()
	if err != nil {
		t.Fatalf("DefaultAuditLogger failed: %v", err)
	}
	defer logger.Close()

	if logger == nil {
		t.Fatal("logger should not be nil")
	}
}

func TestAuditLogger_LogExecution(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer logger.Close()

	entry := AuditEntry{
		Command:   "echo test",
		Directory: "/tmp",
		Shell:     history.Bash,
		ExitCode:  0,
		Duration:  100 * time.Millisecond,
		Status:    "success",
		Validated: true,
		Confirmed: true,
	}

	err = logger.LogExecution(entry)
	if err != nil {
		t.Errorf("LogExecution failed: %v", err)
	}

	// Verify log file has content
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if len(data) == 0 {
		t.Error("log file should not be empty")
	}
}

func TestAuditLogger_LogSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer logger.Close()

	cmd := &history.CommandRecord{
		Command:   "echo test",
		Directory: "/tmp",
		Shell:     history.Bash,
		ExitCode:  0,
	}

	err = logger.LogSuccess(cmd, 100*time.Millisecond)
	if err != nil {
		t.Errorf("LogSuccess failed: %v", err)
	}
}

func TestAuditLogger_LogFailure(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer logger.Close()

	cmd := &history.CommandRecord{
		Command:   "false",
		Directory: "/tmp",
		Shell:     history.Bash,
		ExitCode:  1,
	}

	err = logger.LogFailure(cmd, 1, 50*time.Millisecond, "command failed")
	if err != nil {
		t.Errorf("LogFailure failed: %v", err)
	}
}

func TestAuditLogger_LogBlocked(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer logger.Close()

	err = logger.LogBlocked("rm -rf /", "/tmp", history.Bash, "dangerous_pattern")
	if err != nil {
		t.Errorf("LogBlocked failed: %v", err)
	}
}

func TestAuditLogger_LogCancelled(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer logger.Close()

	cmd := &history.CommandRecord{
		Command:   "rm -rf /tmp/test",
		Directory: "/tmp",
		Shell:     history.Bash,
	}

	err = logger.LogCancelled(cmd, "user_declined")
	if err != nil {
		t.Errorf("LogCancelled failed: %v", err)
	}
}

func TestAuditLogger_GetRecentEntries(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer logger.Close()

	// Log multiple entries
	for i := 0; i < 5; i++ {
		entry := AuditEntry{
			Command:   "echo test",
			Directory: "/tmp",
			Shell:     history.Bash,
			Status:    "success",
		}

		if err := logger.LogExecution(entry); err != nil {
			t.Fatalf("LogExecution failed: %v", err)
		}
	}

	// Get recent entries
	entries, err := logger.GetRecentEntries(3)
	if err != nil {
		t.Fatalf("GetRecentEntries failed: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestAuditLogger_GetRecentEntries_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer logger.Close()

	entries, err := logger.GetRecentEntries(10)
	if err != nil {
		t.Fatalf("GetRecentEntries failed: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestAuditLogger_Close(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}

	err = logger.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify we can't write after close
	entry := AuditEntry{
		Command: "echo test",
		Status:  "success",
	}

	err = logger.LogExecution(entry)
	if err == nil {
		t.Error("expected error when writing to closed logger")
	}
}

func TestAuditLogger_RotateLog(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer logger.Close()

	// Write some entries
	entry := AuditEntry{
		Command: "echo test",
		Status:  "success",
	}

	if err := logger.LogExecution(entry); err != nil {
		t.Fatalf("LogExecution failed: %v", err)
	}

	// Rotate log
	err = logger.RotateLog()
	if err != nil {
		t.Errorf("RotateLog failed: %v", err)
	}

	// Verify new log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("new log file was not created after rotation")
	}

	// Verify we can still write
	if err := logger.LogExecution(entry); err != nil {
		t.Errorf("LogExecution after rotation failed: %v", err)
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 1},
		{"line1\n", 2},
		{"line1\nline2\n", 3},
		{"line1\nline2\nline3", 3},
		{"single line", 1},
	}

	for _, tt := range tests {
		lines := splitLines(tt.input)
		if len(lines) != tt.expected {
			t.Errorf("splitLines(%q) returned %d lines, want %d", tt.input, len(lines), tt.expected)
		}
	}
}

func TestAuditEntry_Timestamp(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer logger.Close()

	// Log entry without timestamp
	entry := AuditEntry{
		Command: "echo test",
		Status:  "success",
	}

	if err := logger.LogExecution(entry); err != nil {
		t.Fatalf("LogExecution failed: %v", err)
	}

	// Retrieve and verify timestamp was set
	entries, err := logger.GetRecentEntries(1)
	if err != nil {
		t.Fatalf("GetRecentEntries failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatal("expected 1 entry")
	}

	if entries[0].Timestamp.IsZero() {
		t.Error("timestamp should be set automatically")
	}
}
