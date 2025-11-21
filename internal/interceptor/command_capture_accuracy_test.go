package interceptor

import (
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestCommandCaptureAccuracy tests the accuracy of command capture functionality
func TestCommandCaptureAccuracy(t *testing.T) {
	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Create command capture instance
	capture := NewCommandCapture(storage, cfg)

	testCases := []struct {
		name         string
		command      string
		directory    string
		shell        history.ShellType
		exitCode     int
		duration     time.Duration
		expectedTags []string
	}{
		{
			name:         "Simple command with success",
			command:      "echo hello world",
			directory:    getCurrentDir(t),
			shell:        history.Bash,
			exitCode:     0,
			duration:     100 * time.Millisecond,
			expectedTags: []string{"success", "cmd-echo"},
		},
		{
			name:         "Git command with metadata",
			command:      "git status --porcelain",
			directory:    getCurrentDir(t),
			shell:        history.Bash,
			exitCode:     0,
			duration:     250 * time.Millisecond,
			expectedTags: []string{"git", "success", "has-flags"},
		},
		{
			name:         "Failed command",
			command:      "nonexistent-command",
			directory:    getCurrentDir(t),
			shell:        history.Bash,
			exitCode:     127,
			duration:     50 * time.Millisecond,
			expectedTags: []string{"failed"},
		},
		{
			name:         "Long running command",
			command:      "sleep 15",
			directory:    getCurrentDir(t),
			shell:        history.Bash,
			exitCode:     0,
			duration:     15 * time.Second,
			expectedTags: []string{"success", "long-running"},
		},
		{
			name:         "Complex command with pipes and redirection",
			command:      "ps aux | grep bash > /tmp/processes.txt",
			directory:    getCurrentDir(t),
			shell:        history.Bash,
			exitCode:     0,
			duration:     200 * time.Millisecond,
			expectedTags: []string{"success", "has-pipes", "has-redirection", "complex-command"},
		},
		{
			name:         "PowerShell cmdlet",
			command:      "Get-Process -Name powershell | Select-Object Name,Id",
			directory:    getCurrentDir(t),
			shell:        history.PowerShell,
			exitCode:     0,
			duration:     300 * time.Millisecond,
			expectedTags: []string{"success", "has-pipes"},
		},
		{
			name:         "Package manager command",
			command:      "npm install --save-dev typescript",
			directory:    getCurrentDir(t),
			shell:        history.Bash,
			exitCode:     0,
			duration:     5 * time.Second,
			expectedTags: []string{"success", "package-manager", "has-flags"},
		},
		{
			name:         "Build command",
			command:      "go build -o myapp ./cmd/main.go",
			directory:    getCurrentDir(t),
			shell:        history.Bash,
			exitCode:     0,
			duration:     2 * time.Second,
			expectedTags: []string{"success", "build", "has-flags"},
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Add a small delay to ensure unique timestamps
			time.Sleep(time.Duration(i+1) * time.Millisecond)

			// Capture the command
			err := capture.CaptureCommandDirect(tc.command, tc.directory, tc.shell, tc.exitCode, tc.duration)
			if err != nil {
				t.Fatalf("CaptureCommandDirect failed: %v", err)
			}

			// Retrieve and verify the captured command
			commands, err := storage.GetCommandsByDirectory(tc.directory)
			if err != nil {
				t.Fatalf("Failed to retrieve commands: %v", err)
			}

			// Find the command we just captured
			var capturedCmd *history.CommandRecord
			for j := len(commands) - 1; j >= 0; j-- {
				if commands[j].Command == tc.command && commands[j].Shell == tc.shell {
					capturedCmd = &commands[j]
					break
				}
			}

			if capturedCmd == nil {
				t.Fatalf("Captured command not found in storage")
			}

			// Verify command accuracy
			if capturedCmd.Command != tc.command {
				t.Errorf("Command mismatch: expected '%s', got '%s'", tc.command, capturedCmd.Command)
			}

			if capturedCmd.Directory != tc.directory {
				t.Errorf("Directory mismatch: expected '%s', got '%s'", tc.directory, capturedCmd.Directory)
			}

			if capturedCmd.Shell != tc.shell {
				t.Errorf("Shell mismatch: expected %v, got %v", tc.shell, capturedCmd.Shell)
			}

			if capturedCmd.ExitCode != tc.exitCode {
				t.Errorf("Exit code mismatch: expected %d, got %d", tc.exitCode, capturedCmd.ExitCode)
			}

			if capturedCmd.Duration != tc.duration {
				t.Errorf("Duration mismatch: expected %v, got %v", tc.duration, capturedCmd.Duration)
			}

			// Verify timestamp is recent
			if time.Since(capturedCmd.Timestamp) > time.Minute {
				t.Errorf("Timestamp is too old: %v", capturedCmd.Timestamp)
			}

			// Verify ID is generated
			if capturedCmd.ID == "" {
				t.Error("Command ID should not be empty")
			}

			// Verify expected tags are present
			for _, expectedTag := range tc.expectedTags {
				if !capturedCmd.HasTag(expectedTag) {
					t.Errorf("Expected tag '%s' not found in tags: %v", expectedTag, capturedCmd.Tags)
				}
			}

			// Verify system tags are present
			expectedSystemTags := []string{
				"os-" + runtime.GOOS,
				"arch-" + runtime.GOARCH,
				"shell-" + tc.shell.String(),
			}

			for _, expectedTag := range expectedSystemTags {
				hasTag := false
				for _, tag := range capturedCmd.Tags {
					if strings.Contains(tag, expectedTag) {
						hasTag = true
						break
					}
				}
				if !hasTag {
					t.Errorf("Expected system tag containing '%s' not found in tags: %v", expectedTag, capturedCmd.Tags)
				}
			}

			t.Logf("Successfully captured command with %d tags: %v", len(capturedCmd.Tags), capturedCmd.Tags)
		})
	}
}

// TestCommandCaptureMetadataAccuracy tests the accuracy of metadata collection
func TestCommandCaptureMetadataAccuracy(t *testing.T) {
	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Create command capture instance
	capture := NewCommandCapture(storage, cfg)

	testDir := getCurrentDir(t)

	// Test metadata collection for different command types
	metadataTests := []struct {
		name             string
		command          string
		shell            history.ShellType
		expectedMetadata map[string]bool // tag -> should be present
	}{
		{
			name:    "Git command metadata",
			command: "git commit -m 'Initial commit'",
			shell:   history.Bash,
			expectedMetadata: map[string]bool{
				"git":       true,
				"has-flags": true,
				"success":   true,
			},
		},
		{
			name:    "Docker command metadata",
			command: "docker run -d --name myapp nginx:latest",
			shell:   history.Bash,
			expectedMetadata: map[string]bool{
				"docker":    true,
				"has-flags": true,
				"success":   true,
			},
		},
		{
			name:    "PowerShell cmdlet metadata",
			command: "Get-ChildItem -Path C:\\ -Recurse",
			shell:   history.PowerShell,
			expectedMetadata: map[string]bool{
				"success":   true,
				"has-flags": true,
			},
		},
		{
			name:    "Test command metadata",
			command: "go test -v ./...",
			shell:   history.Bash,
			expectedMetadata: map[string]bool{
				"test":      true,
				"has-flags": true,
				"success":   true,
			},
		},
		{
			name:    "Package manager metadata",
			command: "yarn add --dev jest",
			shell:   history.Bash,
			expectedMetadata: map[string]bool{
				"package-manager": true,
				"has-flags":       true,
				"success":         true,
			},
		},
	}

	for i, tc := range metadataTests {
		t.Run(tc.name, func(t *testing.T) {
			// Add delay for unique timestamps
			time.Sleep(time.Duration(i+1) * time.Millisecond)

			// Capture command
			err := capture.CaptureCommandDirect(tc.command, testDir, tc.shell, 0, 100*time.Millisecond)
			if err != nil {
				t.Fatalf("CaptureCommandDirect failed: %v", err)
			}

			// Retrieve captured command
			commands, err := storage.GetCommandsByDirectory(testDir)
			if err != nil {
				t.Fatalf("Failed to retrieve commands: %v", err)
			}

			// Find our command
			var capturedCmd *history.CommandRecord
			for j := len(commands) - 1; j >= 0; j-- {
				if commands[j].Command == tc.command {
					capturedCmd = &commands[j]
					break
				}
			}

			if capturedCmd == nil {
				t.Fatalf("Command not found in storage")
			}

			// Verify expected metadata
			for expectedTag, shouldBePresent := range tc.expectedMetadata {
				hasTag := capturedCmd.HasTag(expectedTag)
				if hasTag != shouldBePresent {
					if shouldBePresent {
						t.Errorf("Expected tag '%s' not found in tags: %v", expectedTag, capturedCmd.Tags)
					} else {
						t.Errorf("Unexpected tag '%s' found in tags: %v", expectedTag, capturedCmd.Tags)
					}
				}
			}

			t.Logf("Command '%s' captured with tags: %v", tc.command, capturedCmd.Tags)
		})
	}
}

// TestCommandCaptureEnvironmentAccuracy tests command capture from environment variables
func TestCommandCaptureEnvironmentAccuracy(t *testing.T) {
	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Create command capture instance
	capture := NewCommandCapture(storage, cfg)

	testDir := getCurrentDir(t)
	testCommand := "echo environment test"

	// Save original environment
	originalEnv := map[string]string{
		"CHT_COMMAND":      os.Getenv("CHT_COMMAND"),
		"CHT_DIRECTORY":    os.Getenv("CHT_DIRECTORY"),
		"CHT_SHELL":        os.Getenv("CHT_SHELL"),
		"CHT_EXIT_CODE":    os.Getenv("CHT_EXIT_CODE"),
		"CHT_DURATION":     os.Getenv("CHT_DURATION"),
		"CHT_TIMESTAMP":    os.Getenv("CHT_TIMESTAMP"),
		"CHT_TRACKER_PATH": os.Getenv("CHT_TRACKER_PATH"),
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

	// Set test environment variables
	testTimestamp := time.Now()
	os.Setenv("CHT_COMMAND", testCommand)
	os.Setenv("CHT_DIRECTORY", testDir)
	os.Setenv("CHT_SHELL", "bash")
	os.Setenv("CHT_EXIT_CODE", "0")
	os.Setenv("CHT_DURATION", "150")
	os.Setenv("CHT_TIMESTAMP", testTimestamp.Format(time.RFC3339))

	// Enable tracking
	execPath, _ := os.Executable()
	os.Setenv("CHT_TRACKER_PATH", execPath)

	// Capture command from environment
	err := capture.CaptureCommand()
	if err != nil {
		t.Fatalf("CaptureCommand failed: %v", err)
	}

	// Verify captured command
	commands, err := storage.GetCommandsByDirectory(testDir)
	if err != nil {
		t.Fatalf("Failed to retrieve commands: %v", err)
	}

	if len(commands) == 0 {
		t.Fatal("No commands were captured")
	}

	capturedCmd := commands[len(commands)-1] // Get the last command

	// Verify accuracy of environment-based capture
	if capturedCmd.Command != testCommand {
		t.Errorf("Command mismatch: expected '%s', got '%s'", testCommand, capturedCmd.Command)
	}

	if capturedCmd.Directory != testDir {
		t.Errorf("Directory mismatch: expected '%s', got '%s'", testDir, capturedCmd.Directory)
	}

	if capturedCmd.Shell != history.Bash {
		t.Errorf("Shell mismatch: expected Bash, got %v", capturedCmd.Shell)
	}

	if capturedCmd.ExitCode != 0 {
		t.Errorf("Exit code mismatch: expected 0, got %d", capturedCmd.ExitCode)
	}

	if capturedCmd.Duration != 150*time.Millisecond {
		t.Errorf("Duration mismatch: expected 150ms, got %v", capturedCmd.Duration)
	}

	// Verify timestamp is close to what we set
	timeDiff := capturedCmd.Timestamp.Sub(testTimestamp)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > time.Second {
		t.Errorf("Timestamp difference too large: %v", timeDiff)
	}

	// Verify ID is generated
	if capturedCmd.ID == "" {
		t.Error("Command ID should not be empty")
	}

	// Verify basic tags are present
	if !capturedCmd.HasTag("success") {
		t.Errorf("Expected 'success' tag, got tags: %v", capturedCmd.Tags)
	}

	t.Logf("Environment capture successful: %+v", capturedCmd)
}

// TestCommandCaptureFilteringAccuracy tests command filtering accuracy
func TestCommandCaptureFilteringAccuracy(t *testing.T) {
	// Create test storage and config with specific exclude patterns
	storage, cfg := createTestStorage(t)
	cfg.ExcludePatterns = []string{"cd", "ls", "pwd", "echo", "tracker"}
	defer storage.Close()

	// Create command capture instance
	capture := NewCommandCapture(storage, cfg)

	testDir := getCurrentDir(t)

	filterTests := []struct {
		name           string
		command        string
		shouldBeStored bool
	}{
		{
			name:           "Excluded command - cd",
			command:        "cd /tmp",
			shouldBeStored: false,
		},
		{
			name:           "Excluded command - ls",
			command:        "ls -la",
			shouldBeStored: false,
		},
		{
			name:           "Excluded command - pwd",
			command:        "pwd",
			shouldBeStored: false,
		},
		{
			name:           "Excluded command - echo",
			command:        "echo hello",
			shouldBeStored: false,
		},
		{
			name:           "Excluded command - tracker",
			command:        "tracker status",
			shouldBeStored: false,
		},
		{
			name:           "Allowed command - git",
			command:        "git status",
			shouldBeStored: true,
		},
		{
			name:           "Allowed command - npm",
			command:        "npm install",
			shouldBeStored: true,
		},
		{
			name:           "Allowed command - go",
			command:        "go build",
			shouldBeStored: true,
		},
	}

	initialCommandCount := 0
	if commands, err := storage.GetCommandsByDirectory(testDir); err == nil {
		initialCommandCount = len(commands)
	}

	for i, tc := range filterTests {
		t.Run(tc.name, func(t *testing.T) {
			// Add delay for unique timestamps
			time.Sleep(time.Duration(i+1) * time.Millisecond)

			// Capture command
			err := capture.CaptureCommandDirect(tc.command, testDir, history.Bash, 0, 100*time.Millisecond)
			if err != nil {
				t.Fatalf("CaptureCommandDirect failed: %v", err)
			}

			// Check if command was stored
			commands, err := storage.GetCommandsByDirectory(testDir)
			if err != nil {
				t.Fatalf("Failed to retrieve commands: %v", err)
			}

			// Look for our specific command
			found := false
			for _, cmd := range commands {
				if cmd.Command == tc.command {
					found = true
					break
				}
			}

			if found != tc.shouldBeStored {
				if tc.shouldBeStored {
					t.Errorf("Command '%s' should have been stored but was not found", tc.command)
				} else {
					t.Errorf("Command '%s' should have been filtered but was found in storage", tc.command)
				}
			}

			t.Logf("Command '%s' filtering result: stored=%v (expected=%v)", tc.command, found, tc.shouldBeStored)
		})
	}

	// Verify final command count matches expectations
	finalCommands, err := storage.GetCommandsByDirectory(testDir)
	if err != nil {
		t.Fatalf("Failed to get final command count: %v", err)
	}

	expectedStoredCommands := 0
	for _, tc := range filterTests {
		if tc.shouldBeStored {
			expectedStoredCommands++
		}
	}

	actualNewCommands := len(finalCommands) - initialCommandCount
	if actualNewCommands != expectedStoredCommands {
		t.Errorf("Expected %d new commands, got %d", expectedStoredCommands, actualNewCommands)
	}
}

// TestCommandCaptureErrorHandling tests error handling in command capture
func TestCommandCaptureErrorHandling(t *testing.T) {
	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Create command capture instance
	capture := NewCommandCapture(storage, cfg)

	// Test with invalid directory
	err := capture.CaptureCommandDirect("echo test", "/nonexistent/directory", history.Bash, 0, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("CaptureCommandDirect should handle invalid directory gracefully: %v", err)
	}

	// Verify command was still captured (with directory-missing tag)
	commands, err := storage.GetCommandsByDirectory("/nonexistent/directory")
	if err != nil {
		t.Fatalf("Failed to retrieve commands: %v", err)
	}

	if len(commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(commands))
	}

	cmd := commands[0]
	if !cmd.HasTag("directory-missing") {
		t.Errorf("Expected 'directory-missing' tag for invalid directory, got tags: %v", cmd.Tags)
	}

	// Test with empty command
	err = capture.CaptureCommandDirect("", getCurrentDir(t), history.Bash, 0, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("CaptureCommandDirect should handle empty command gracefully: %v", err)
	}

	// Empty command should be filtered out, so no new commands should be stored
	commands, err = storage.GetCommandsByDirectory(getCurrentDir(t))
	if err != nil {
		t.Fatalf("Failed to retrieve commands: %v", err)
	}

	// Should not find the empty command
	for _, cmd := range commands {
		if cmd.Command == "" {
			t.Error("Empty command should have been filtered out")
		}
	}
}
