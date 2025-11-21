package interceptor

import (
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCommandCaptureIntegration tests the complete command capture workflow
func TestCommandCaptureIntegration(t *testing.T) {
	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Create command capture instance
	capture := NewCommandCapture(storage, cfg)

	// Test direct command capture
	testDir := getCurrentDir(t)
	testCommand := "git status"
	shell := history.Bash
	exitCode := 0
	duration := 150 * time.Millisecond

	err := capture.CaptureCommandDirect(testCommand, testDir, shell, exitCode, duration)
	if err != nil {
		t.Fatalf("CaptureCommandDirect failed: %v", err)
	}

	// Verify command was stored with enhanced metadata
	commands, err := storage.GetCommandsByDirectory(testDir)
	if err != nil {
		t.Fatalf("Failed to retrieve commands: %v", err)
	}

	if len(commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(commands))
	}

	cmd := commands[0]

	// Verify basic command data
	if cmd.Command != testCommand {
		t.Errorf("Expected command '%s', got '%s'", testCommand, cmd.Command)
	}
	if cmd.Directory != testDir {
		t.Errorf("Expected directory '%s', got '%s'", testDir, cmd.Directory)
	}
	if cmd.Shell != shell {
		t.Errorf("Expected shell %v, got %v", shell, cmd.Shell)
	}
	if cmd.ExitCode != exitCode {
		t.Errorf("Expected exit code %d, got %d", exitCode, cmd.ExitCode)
	}
	if cmd.Duration != duration {
		t.Errorf("Expected duration %v, got %v", duration, cmd.Duration)
	}

	// Verify enhanced metadata was collected
	if len(cmd.Tags) == 0 {
		t.Error("Expected command to have tags from metadata collection")
	}

	// Debug: print all tags
	t.Logf("Command tags: %v", cmd.Tags)

	// Check for expected tags
	expectedTags := []string{"git", "success"}
	for _, expectedTag := range expectedTags {
		if !cmd.HasTag(expectedTag) {
			t.Errorf("Expected command to have tag '%s', tags: %v", expectedTag, cmd.Tags)
		}
	}

	// Check for shell tag (might be different format)
	hasShellTag := false
	for _, tag := range cmd.Tags {
		if strings.Contains(tag, "shell") {
			hasShellTag = true
			break
		}
	}
	if !hasShellTag {
		t.Errorf("Expected command to have a shell tag, tags: %v", cmd.Tags)
	}

	// Verify timestamp was set
	if cmd.Timestamp.IsZero() {
		t.Error("Expected command to have a timestamp")
	}

	// Verify ID was generated
	if cmd.ID == "" {
		t.Error("Expected command to have an ID")
	}
}

// TestCommandCaptureWithEnvironment tests command capture using environment variables
func TestCommandCaptureWithEnvironment(t *testing.T) {
	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Create command capture instance
	capture := NewCommandCapture(storage, cfg)

	// Set up environment variables
	testDir := getCurrentDir(t)
	testCommand := "npm install"

	originalEnv := map[string]string{
		"CHT_COMMAND":   os.Getenv("CHT_COMMAND"),
		"CHT_DIRECTORY": os.Getenv("CHT_DIRECTORY"),
		"CHT_SHELL":     os.Getenv("CHT_SHELL"),
		"CHT_EXIT_CODE": os.Getenv("CHT_EXIT_CODE"),
		"CHT_DURATION":  os.Getenv("CHT_DURATION"),
		"CHT_TIMESTAMP": os.Getenv("CHT_TIMESTAMP"),
	}

	// Clean up environment after test
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Set test environment variables
	os.Setenv("CHT_COMMAND", testCommand)
	os.Setenv("CHT_DIRECTORY", testDir)
	os.Setenv("CHT_SHELL", "bash")
	os.Setenv("CHT_EXIT_CODE", "0")
	os.Setenv("CHT_DURATION", "2500")
	os.Setenv("CHT_TIMESTAMP", time.Now().Format(time.RFC3339))

	// Enable tracking - use current executable path
	execPath, _ := os.Executable()
	os.Setenv("CHT_TRACKER_PATH", execPath)

	// Capture command from environment
	err := capture.CaptureCommand()
	if err != nil {
		t.Fatalf("CaptureCommand failed: %v", err)
	}

	// Verify command was stored
	commands, err := storage.GetCommandsByDirectory(testDir)
	if err != nil {
		t.Fatalf("Failed to retrieve commands: %v", err)
	}

	if len(commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(commands))
	}

	cmd := commands[0]

	// Verify command data
	if cmd.Command != testCommand {
		t.Errorf("Expected command '%s', got '%s'", testCommand, cmd.Command)
	}
	if cmd.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", cmd.ExitCode)
	}
	if cmd.Duration != 2500*time.Millisecond {
		t.Errorf("Expected duration 2500ms, got %v", cmd.Duration)
	}

	// Check for package manager tag
	if !cmd.HasTag("package-manager") {
		t.Errorf("Expected command to have 'package-manager' tag, tags: %v", cmd.Tags)
	}
}

// TestCommandCaptureMetadataCollection tests the metadata collection functionality
func TestCommandCaptureMetadataCollection(t *testing.T) {
	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Create command capture instance
	capture := NewCommandCapture(storage, cfg)

	// Test different types of commands to verify metadata collection
	testCases := []struct {
		command     string
		expectedTag string
	}{
		{"git commit -m 'test'", "git"},
		{"docker run nginx", "docker"},
		{"go build ./...", "build"},
		{"go test ./...", "test"},
		{"npm run build", "package-manager"},
		{"make clean", "build"},
	}

	testDir := getCurrentDir(t)

	for i, tc := range testCases {
		err := capture.CaptureCommandDirect(tc.command, testDir, history.Bash, 0, 100*time.Millisecond)
		if err != nil {
			t.Fatalf("Test case %d: CaptureCommandDirect failed: %v", i, err)
		}
	}

	// Verify all commands were stored with correct metadata
	commands, err := storage.GetCommandsByDirectory(testDir)
	if err != nil {
		t.Fatalf("Failed to retrieve commands: %v", err)
	}

	if len(commands) != len(testCases) {
		t.Fatalf("Expected %d commands, got %d", len(testCases), len(commands))
	}

	// Verify each command has the expected tag
	for i, cmd := range commands {
		tc := testCases[len(testCases)-1-i] // Commands are returned in reverse order
		if !cmd.HasTag(tc.expectedTag) {
			t.Errorf("Command '%s' expected to have tag '%s', tags: %v",
				tc.command, tc.expectedTag, cmd.Tags)
		}

		// All commands should have success tag (exit code 0)
		if !cmd.HasTag("success") {
			t.Errorf("Command '%s' should have 'success' tag, tags: %v",
				tc.command, cmd.Tags)
		}

		// All commands should have a shell tag
		hasShellTag := false
		for _, tag := range cmd.Tags {
			if strings.Contains(tag, "shell") {
				hasShellTag = true
				break
			}
		}
		if !hasShellTag {
			t.Errorf("Command '%s' should have a shell tag, tags: %v",
				tc.command, cmd.Tags)
		}
	}
}

// TestCommandCaptureFiltering tests command filtering functionality
func TestCommandCaptureFiltering(t *testing.T) {
	// Create test storage and config with exclude patterns
	storage, cfg := createTestStorage(t)
	cfg.ExcludePatterns = []string{"cd", "ls", "pwd"}
	defer storage.Close()

	// Create command capture instance
	capture := NewCommandCapture(storage, cfg)

	testDir := getCurrentDir(t)

	// Test commands that should be filtered out
	filteredCommands := []string{"cd /tmp", "ls -la", "pwd"}
	for _, cmd := range filteredCommands {
		err := capture.CaptureCommandDirect(cmd, testDir, history.Bash, 0, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("CaptureCommandDirect failed for filtered command '%s': %v", cmd, err)
		}
	}

	// Test command that should not be filtered
	validCommand := "echo hello"
	err := capture.CaptureCommandDirect(validCommand, testDir, history.Bash, 0, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("CaptureCommandDirect failed for valid command: %v", err)
	}

	// Verify only the valid command was stored
	commands, err := storage.GetCommandsByDirectory(testDir)
	if err != nil {
		t.Fatalf("Failed to retrieve commands: %v", err)
	}

	if len(commands) != 1 {
		t.Fatalf("Expected 1 command (filtered), got %d", len(commands))
	}

	if commands[0].Command != validCommand {
		t.Errorf("Expected stored command '%s', got '%s'", validCommand, commands[0].Command)
	}
}

// TestCommandCaptureStats tests the capture statistics functionality
func TestCommandCaptureStats(t *testing.T) {
	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Create command capture instance
	capture := NewCommandCapture(storage, cfg)

	// Add commands to multiple directories
	dirs := []string{getCurrentDir(t), filepath.Join(getCurrentDir(t), "subdir")}

	for _, dir := range dirs {
		for i := 0; i < 3; i++ {
			cmd := fmt.Sprintf("echo test%d", i)
			err := capture.CaptureCommandDirect(cmd, dir, history.Bash, 0, 100*time.Millisecond)
			if err != nil {
				t.Fatalf("CaptureCommandDirect failed: %v", err)
			}
		}
	}

	// Get capture statistics
	stats, err := capture.GetCaptureStats()
	if err != nil {
		t.Fatalf("GetCaptureStats failed: %v", err)
	}

	// Verify statistics
	if stats.TotalDirectories < 1 {
		t.Errorf("Expected at least 1 directory, got %d", stats.TotalDirectories)
	}
	if stats.TotalCommands < 3 {
		t.Errorf("Expected at least 3 commands, got %d", stats.TotalCommands)
	}
}
