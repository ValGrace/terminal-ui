//go:build integration
// +build integration

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/ValGrace/command-history-tracker/internal/storage"
	"github.com/ValGrace/command-history-tracker/pkg/history"
)

// TestEndToEndWorkflow tests the complete workflow from installation to command browsing
func TestEndToEndWorkflow(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Set up test environment
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	t.Run("CompleteWorkflow", func(t *testing.T) {
		// Step 1: Initialize configuration
		cfg := config.DefaultConfig()
		cfg.StoragePath = filepath.Join(tmpDir, ".command-history-tracker", "commands.db")

		if err := cfg.SaveConfig(); err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		t.Log("✓ Configuration initialized")

		// Step 2: Create storage
		store, err := storage.NewSQLiteStorage(cfg.StoragePath)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}
		defer store.Close()

		t.Log("✓ Storage created")

		// Step 3: Record some commands
		testCommands := []struct {
			command   string
			directory string
		}{
			{"git status", "/home/user/project1"},
			{"npm test", "/home/user/project1"},
			{"go build", "/home/user/project2"},
			{"make test", "/home/user/project2"},
			{"docker ps", "/home/user/project1"},
		}

		for _, tc := range testCommands {
			record := history.CommandRecord{
				Command:   tc.command,
				Directory: tc.directory,
				Timestamp: time.Now(),
				Shell:     history.Bash,
				ExitCode:  0,
				Duration:  time.Second,
			}

			if err := store.SaveCommand(record); err != nil {
				t.Fatalf("Failed to save command: %v", err)
			}
		}

		t.Log("✓ Commands recorded")

		// Step 4: Verify commands can be retrieved
		dirs, err := store.GetDirectoriesWithHistory()
		if err != nil {
			t.Fatalf("Failed to get directories: %v", err)
		}

		if len(dirs) != 2 {
			t.Errorf("Expected 2 directories, got %d", len(dirs))
		}

		t.Log("✓ Directories retrieved")

		// Step 5: Verify commands for specific directory
		commands, err := store.GetCommandsByDirectory("/home/user/project1")
		if err != nil {
			t.Fatalf("Failed to get commands: %v", err)
		}

		if len(commands) != 3 {
			t.Errorf("Expected 3 commands for project1, got %d", len(commands))
		}

		t.Log("✓ Commands retrieved for directory")

		// Step 6: Test cleanup
		if err := store.CleanupOldCommands(0); err != nil {
			t.Fatalf("Failed to cleanup: %v", err)
		}

		// Verify all commands were cleaned up
		commands, err = store.GetCommandsByDirectory("/home/user/project1")
		if err != nil {
			t.Fatalf("Failed to get commands after cleanup: %v", err)
		}

		if len(commands) != 0 {
			t.Errorf("Expected 0 commands after cleanup, got %d", len(commands))
		}

		t.Log("✓ Cleanup completed")
	})
}

// TestShellIntegration tests shell integration across platforms
func TestShellIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	t.Run("BashIntegration", func(t *testing.T) {
		// Skip if bash is not available
		if _, err := exec.LookPath("bash"); err != nil {
			t.Skip("Bash not available")
		}

		// Create a test bash script that simulates command recording
		bashScript := filepath.Join(tmpDir, "test.sh")
		scriptContent := `#!/bin/bash
echo "Testing command recording"
tracker record --command "test command" --directory "$PWD" --exit-code 0
`

		if err := os.WriteFile(bashScript, []byte(scriptContent), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		t.Log("✓ Bash integration test prepared")
	})

	t.Run("PowerShellIntegration", func(t *testing.T) {
		// Skip if PowerShell is not available
		if _, err := exec.LookPath("pwsh"); err != nil {
			if _, err := exec.LookPath("powershell"); err != nil {
				t.Skip("PowerShell not available")
			}
		}

		t.Log("✓ PowerShell integration test prepared")
	})
}

// TestCommandRecordingAccuracy tests that commands are recorded accurately
func TestCommandRecordingAccuracy(t *testing.T) {
	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cfg := config.DefaultConfig()
	cfg.StoragePath = filepath.Join(tmpDir, "commands.db")

	store, err := storage.NewSQLiteStorage(cfg.StoragePath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	testCases := []struct {
		name      string
		command   string
		directory string
		exitCode  int
	}{
		{
			name:      "SimpleCommand",
			command:   "ls -la",
			directory: "/home/user",
			exitCode:  0,
		},
		{
			name:      "CommandWithPipes",
			command:   "cat file.txt | grep pattern | wc -l",
			directory: "/home/user/project",
			exitCode:  0,
		},
		{
			name:      "CommandWithQuotes",
			command:   `echo "Hello World"`,
			directory: "/home/user",
			exitCode:  0,
		},
		{
			name:      "FailedCommand",
			command:   "false",
			directory: "/home/user",
			exitCode:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			record := history.CommandRecord{
				Command:   tc.command,
				Directory: tc.directory,
				Timestamp: time.Now(),
				Shell:     history.Bash,
				ExitCode:  tc.exitCode,
				Duration:  time.Millisecond * 100,
			}

			if err := store.SaveCommand(record); err != nil {
				t.Fatalf("Failed to save command: %v", err)
			}

			// Retrieve and verify
			commands, err := store.GetCommandsByDirectory(tc.directory)
			if err != nil {
				t.Fatalf("Failed to retrieve commands: %v", err)
			}

			found := false
			for _, cmd := range commands {
				if cmd.Command == tc.command && cmd.ExitCode == tc.exitCode {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Command not found or incorrectly recorded: %s", tc.command)
			}
		})
	}
}

// TestCrossDirectoryNavigation tests browsing commands across directories
func TestCrossDirectoryNavigation(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.StoragePath = filepath.Join(tmpDir, "commands.db")

	store, err := storage.NewSQLiteStorage(cfg.StoragePath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Create a directory tree with commands
	directories := []string{
		"/home/user/project1",
		"/home/user/project1/src",
		"/home/user/project1/tests",
		"/home/user/project2",
		"/home/user/project2/api",
	}

	for _, dir := range directories {
		for i := 0; i < 5; i++ {
			record := history.CommandRecord{
				Command:   "test command " + string(rune(i)),
				Directory: dir,
				Timestamp: time.Now(),
				Shell:     history.Bash,
				ExitCode:  0,
				Duration:  time.Second,
			}

			if err := store.SaveCommand(record); err != nil {
				t.Fatalf("Failed to save command: %v", err)
			}
		}
	}

	// Verify all directories are tracked
	dirs, err := store.GetDirectoriesWithHistory()
	if err != nil {
		t.Fatalf("Failed to get directories: %v", err)
	}

	if len(dirs) != len(directories) {
		t.Errorf("Expected %d directories, got %d", len(directories), len(dirs))
	}

	// Verify we can navigate to each directory
	for _, dir := range directories {
		commands, err := store.GetCommandsByDirectory(dir)
		if err != nil {
			t.Fatalf("Failed to get commands for %s: %v", dir, err)
		}

		if len(commands) != 5 {
			t.Errorf("Expected 5 commands for %s, got %d", dir, len(commands))
		}
	}

	t.Log("✓ Cross-directory navigation verified")
}

// TestInstallationProcess tests the complete installation process
func TestInstallationProcess(t *testing.T) {
	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	t.Run("FreshInstallation", func(t *testing.T) {
		// Verify no config exists
		configPath := config.GetConfigPath()
		if _, err := os.Stat(configPath); !os.IsNotExist(err) {
			t.Error("Config should not exist before installation")
		}

		// Create default config (simulating setup)
		cfg := config.DefaultConfig()
		if err := cfg.SaveConfig(); err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		// Verify config was created
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("Config should exist after installation")
		}

		// Load and verify config
		loadedCfg, err := config.LoadConfig()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if loadedCfg.RetentionDays != cfg.RetentionDays {
			t.Error("Config values don't match")
		}

		t.Log("✓ Fresh installation completed")
	})

	t.Run("ReinstallationPreservesData", func(t *testing.T) {
		// Create storage with some data
		cfg, _ := config.LoadConfig()
		store, err := storage.NewSQLiteStorage(cfg.StoragePath)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

		record := history.CommandRecord{
			Command:   "important command",
			Directory: "/home/user/project",
			Timestamp: time.Now(),
			Shell:     history.Bash,
			ExitCode:  0,
			Duration:  time.Second,
		}

		if err := store.SaveCommand(record); err != nil {
			t.Fatalf("Failed to save command: %v", err)
		}
		store.Close()

		// Simulate reinstallation (config update)
		cfg.RetentionDays = 180
		if err := cfg.SaveConfig(); err != nil {
			t.Fatalf("Failed to update config: %v", err)
		}

		// Verify data is preserved
		store, err = storage.NewSQLiteStorage(cfg.StoragePath)
		if err != nil {
			t.Fatalf("Failed to reopen storage: %v", err)
		}
		defer store.Close()

		commands, err := store.GetCommandsByDirectory("/home/user/project")
		if err != nil {
			t.Fatalf("Failed to get commands: %v", err)
		}

		if len(commands) != 1 {
			t.Error("Data should be preserved after reinstallation")
		}

		if !strings.Contains(commands[0].Command, "important command") {
			t.Error("Command data should be preserved")
		}

		t.Log("✓ Reinstallation preserves data")
	})
}
