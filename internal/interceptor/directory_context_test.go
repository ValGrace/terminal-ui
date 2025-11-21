package interceptor

import (
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDirectoryContextResolution tests directory context detection and normalization
func TestDirectoryContextResolution(t *testing.T) {
	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Create command capture instance
	capture := NewCommandCapture(storage, cfg)

	// Get current directory for testing
	currentDir := getCurrentDir(t)

	// Create temporary test directories
	tempDir := t.TempDir()

	directoryTests := []struct {
		name              string
		inputDirectory    string
		expectedDirectory string
	}{
		{
			name:              "Current directory",
			inputDirectory:    currentDir,
			expectedDirectory: currentDir,
		},
		{
			name:              "Temporary directory",
			inputDirectory:    tempDir,
			expectedDirectory: filepath.ToSlash(tempDir),
		},
	}

	for i, tc := range directoryTests {
		t.Run(tc.name, func(t *testing.T) {
			// Add delay for unique timestamps
			time.Sleep(time.Duration(i+1) * time.Millisecond)

			testCommand := "echo directory test " + tc.name

			// Capture command with test directory
			err := capture.CaptureCommandDirect(testCommand, tc.inputDirectory, history.Bash, 0, 100*time.Millisecond)
			if err != nil {
				t.Fatalf("CaptureCommandDirect failed: %v", err)
			}

			// Retrieve captured command
			commands, err := storage.GetCommandsByDirectory(tc.expectedDirectory)
			if err != nil {
				t.Fatalf("Failed to retrieve commands from expected directory: %v", err)
			}

			// Find our command
			var capturedCmd *history.CommandRecord
			for j := len(commands) - 1; j >= 0; j-- {
				if commands[j].Command == testCommand {
					capturedCmd = &commands[j]
					break
				}
			}

			if capturedCmd == nil {
				// Debug: list all directories with commands
				allDirs, _ := storage.GetDirectoriesWithHistory()
				t.Logf("All directories with history: %v", allDirs)
				t.Fatalf("Command not found in expected directory '%s'", tc.expectedDirectory)
			}

			// Verify directory normalization
			if capturedCmd.Directory != tc.expectedDirectory {
				t.Errorf("Directory normalization failed: expected '%s', got '%s'",
					tc.expectedDirectory, capturedCmd.Directory)
			}

			t.Logf("Directory resolution successful: '%s' -> '%s'", tc.inputDirectory, capturedCmd.Directory)
		})
	}
}

// TestDirectoryContextDetection tests automatic directory context detection
func TestDirectoryContextDetection(t *testing.T) {
	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Create command capture instance
	capture := NewCommandCapture(storage, cfg)

	// Test automatic directory detection from environment
	testDir := getCurrentDir(t)
	testCommand := "echo directory detection test"

	// Save original environment
	originalEnv := map[string]string{
		"CHT_COMMAND":   os.Getenv("CHT_COMMAND"),
		"CHT_DIRECTORY": os.Getenv("CHT_DIRECTORY"),
	}
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Set test environment
	os.Setenv("CHT_COMMAND", testCommand)
	os.Setenv("CHT_DIRECTORY", testDir)

	// Capture command without specifying directory
	err := capture.CaptureCommand()
	if err != nil {
		t.Fatalf("CaptureCommand failed: %v", err)
	}

	// Verify command was captured with correct directory
	commands, err := storage.GetCommandsByDirectory(testDir)
	if err != nil {
		t.Fatalf("Failed to retrieve commands: %v", err)
	}

	if len(commands) == 0 {
		t.Fatal("No commands were captured")
	}

	// Find the command
	var capturedCmd *history.CommandRecord
	for _, cmd := range commands {
		if cmd.Command == testCommand && cmd.Directory == testDir {
			capturedCmd = &cmd
			break
		}
	}

	if capturedCmd == nil {
		t.Errorf("Command not found with expected directory '%s'", testDir)
	}
}
