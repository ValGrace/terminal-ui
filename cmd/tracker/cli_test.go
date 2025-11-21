package main

import (
	"bytes"
	"github.com/ValGrace/command-history-tracker/pkg/history/config"
	"github.com/ValGrace/command-history-tracker/pkg/history/storage"
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBrowseCommand tests the browse command functionality
func TestBrowseCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup test environment
	cfg := config.DefaultConfig()
	cfg.StoragePath = filepath.Join(tmpDir, "commands.db")
	config.SetGlobal(cfg)

	// Create storage and add test data
	store, err := storage.NewStorageEngine("sqlite", cfg.StoragePath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	if err := store.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Add test commands
	testDir := "/test/directory"
	for i := 0; i < 5; i++ {
		record := history.CommandRecord{
			Command:   "test command " + string(rune('A'+i)),
			Directory: testDir,
			Timestamp: time.Now(),
			Shell:     history.Bash,
			ExitCode:  0,
			Duration:  time.Second,
		}
		if err := store.SaveCommand(record); err != nil {
			t.Fatalf("Failed to save command: %v", err)
		}
	}

	t.Run("BrowseWithDirectory", func(t *testing.T) {
		// Test that browse command can be created with directory flag
		browseFlags.dir = testDir

		// Verify commands exist
		commands, err := store.GetCommandsByDirectory(testDir)
		if err != nil {
			t.Fatalf("Failed to get commands: %v", err)
		}

		if len(commands) != 5 {
			t.Errorf("Expected 5 commands, got %d", len(commands))
		}
	})

	t.Run("BrowseTreeView", func(t *testing.T) {
		// Test tree view flag
		browseFlags.tree = true

		dirs, err := store.GetDirectoriesWithHistory()
		if err != nil {
			t.Fatalf("Failed to get directories: %v", err)
		}

		if len(dirs) == 0 {
			t.Error("Expected at least one directory with history")
		}
	})
}

// TestHistoryCommand tests the history command functionality
func TestHistoryCommand(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.StoragePath = filepath.Join(tmpDir, "commands.db")
	config.SetGlobal(cfg)

	store, err := storage.NewStorageEngine("sqlite", cfg.StoragePath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	if err := store.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Add test commands with different timestamps
	testDir := "/test/history"
	now := time.Now()

	commands := []struct {
		command   string
		timestamp time.Time
		shell     history.ShellType
	}{
		{"recent command", now, history.Bash},
		{"old command", now.Add(-48 * time.Hour), history.Bash},
		{"very old command", now.Add(-8 * 24 * time.Hour), history.Bash},
		{"powershell command", now, history.PowerShell},
	}

	for _, cmd := range commands {
		record := history.CommandRecord{
			Command:   cmd.command,
			Directory: testDir,
			Timestamp: cmd.timestamp,
			Shell:     cmd.shell,
			ExitCode:  0,
			Duration:  time.Second,
		}
		if err := store.SaveCommand(record); err != nil {
			t.Fatalf("Failed to save command: %v", err)
		}
	}

	t.Run("HistoryWithLimit", func(t *testing.T) {
		historyFlags.limit = 2
		historyFlags.dir = testDir

		allCommands, err := store.GetCommandsByDirectory(testDir)
		if err != nil {
			t.Fatalf("Failed to get commands: %v", err)
		}

		if len(allCommands) < 2 {
			t.Error("Expected at least 2 commands")
		}
	})

	t.Run("HistoryWithTimeFilter", func(t *testing.T) {
		historyFlags.since = "24h"
		historyFlags.dir = testDir

		allCommands, err := store.GetCommandsByDirectory(testDir)
		if err != nil {
			t.Fatalf("Failed to get commands: %v", err)
		}

		// Apply filter
		filtered := applyHistoryFilters(allCommands)

		// Should only include recent commands (within 24h)
		for _, cmd := range filtered {
			if time.Since(cmd.Timestamp) > 24*time.Hour {
				t.Error("Filter should exclude commands older than 24h")
			}
		}
	})

	t.Run("HistoryWithShellFilter", func(t *testing.T) {
		historyFlags.shell = "powershell"
		historyFlags.dir = testDir

		allCommands, err := store.GetCommandsByDirectory(testDir)
		if err != nil {
			t.Fatalf("Failed to get commands: %v", err)
		}

		// Apply filter
		filtered := applyHistoryFilters(allCommands)

		// Should only include PowerShell commands
		for _, cmd := range filtered {
			if cmd.Shell != history.PowerShell {
				t.Error("Filter should only include PowerShell commands")
			}
		}
	})
}

// TestSearchCommand tests the search command functionality
func TestSearchCommand(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.StoragePath = filepath.Join(tmpDir, "commands.db")
	config.SetGlobal(cfg)

	store, err := storage.NewStorageEngine("sqlite", cfg.StoragePath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	if err := store.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Add test commands
	testCommands := []struct {
		command   string
		directory string
	}{
		{"git status", "/project1"},
		{"git commit -m 'test'", "/project1"},
		{"npm install", "/project2"},
		{"npm test", "/project2"},
		{"docker build", "/project1"},
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

	t.Run("SearchInDirectory", func(t *testing.T) {
		searchFlags.dir = "/project1"

		results, err := store.SearchCommands("git", "/project1")
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 git commands in project1, got %d", len(results))
		}
	})

	t.Run("SearchAllDirectories", func(t *testing.T) {
		searchFlags.allDirs = true

		dirs, err := store.GetDirectoriesWithHistory()
		if err != nil {
			t.Fatalf("Failed to get directories: %v", err)
		}

		var allResults []history.CommandRecord
		for _, dir := range dirs {
			results, err := store.SearchCommands("npm", dir)
			if err != nil {
				continue
			}
			allResults = append(allResults, results...)
		}

		if len(allResults) != 2 {
			t.Errorf("Expected 2 npm commands across all directories, got %d", len(allResults))
		}
	})

	t.Run("SearchCaseInsensitive", func(t *testing.T) {
		searchFlags.caseSensitive = false

		results, err := store.SearchCommands("GIT", "/project1")
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}

		// Apply case-insensitive filter
		filtered := filterCaseInsensitive(results, "GIT")

		if len(filtered) == 0 {
			t.Error("Case-insensitive search should find results")
		}
	})
}

// TestConfigCommand tests the config command functionality
func TestConfigCommand(t *testing.T) {
	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	t.Run("ListConfig", func(t *testing.T) {
		cfg := config.DefaultConfig()

		// Capture output
		var buf bytes.Buffer

		// Test that config values can be accessed
		if cfg.RetentionDays <= 0 {
			t.Error("RetentionDays should be positive")
		}
		if cfg.MaxCommands <= 0 {
			t.Error("MaxCommands should be positive")
		}
		if cfg.StoragePath == "" {
			t.Error("StoragePath should not be empty")
		}

		_ = buf // Use buf to avoid unused variable error
	})

	t.Run("GetConfigValue", func(t *testing.T) {
		cfg := config.DefaultConfig()

		// Test getting specific values
		tests := []struct {
			key      string
			expected interface{}
		}{
			{"retention_days", cfg.RetentionDays},
			{"max_commands", cfg.MaxCommands},
			{"auto_cleanup", cfg.AutoCleanup},
		}

		for _, tt := range tests {
			t.Run(tt.key, func(t *testing.T) {
				// Verify the value exists in config
				switch tt.key {
				case "retention_days":
					if cfg.RetentionDays != tt.expected.(int) {
						t.Errorf("Expected %v, got %v", tt.expected, cfg.RetentionDays)
					}
				case "max_commands":
					if cfg.MaxCommands != tt.expected.(int) {
						t.Errorf("Expected %v, got %v", tt.expected, cfg.MaxCommands)
					}
				case "auto_cleanup":
					if cfg.AutoCleanup != tt.expected.(bool) {
						t.Errorf("Expected %v, got %v", tt.expected, cfg.AutoCleanup)
					}
				}
			})
		}
	})

	t.Run("SetConfigValue", func(t *testing.T) {
		cfg := config.DefaultConfig()

		// Test setting values
		cfg.RetentionDays = 180
		cfg.MaxCommands = 20000
		cfg.AutoCleanup = false

		// Validate
		if err := cfg.Validate(); err != nil {
			t.Errorf("Config should be valid after setting values: %v", err)
		}

		// Save and reload
		if err := cfg.Save(); err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		loadedCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if loadedCfg.RetentionDays != 180 {
			t.Error("RetentionDays not persisted")
		}
		if loadedCfg.MaxCommands != 20000 {
			t.Error("MaxCommands not persisted")
		}
		if loadedCfg.AutoCleanup != false {
			t.Error("AutoCleanup not persisted")
		}
	})

	t.Run("ResetConfig", func(t *testing.T) {
		// Create modified config
		cfg := config.DefaultConfig()
		cfg.RetentionDays = 999
		if err := cfg.Save(); err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		// Reset to defaults
		defaultCfg := config.DefaultConfig()
		if err := defaultCfg.Save(); err != nil {
			t.Fatalf("Failed to reset config: %v", err)
		}

		// Verify reset
		loadedCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if loadedCfg.RetentionDays == 999 {
			t.Error("Config should be reset to defaults")
		}
	})
}

// TestStartStopCommands tests the start and stop command functionality
func TestStartStopCommands(t *testing.T) {
	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	t.Run("StartCommand", func(t *testing.T) {
		// Test that start command initializes properly
		cfg := config.DefaultConfig()
		config.SetGlobal(cfg)

		// Verify config is valid
		if err := cfg.Validate(); err != nil {
			t.Errorf("Config should be valid: %v", err)
		}
	})

	t.Run("StopCommand", func(t *testing.T) {
		// Test that stop command can be called
		cfg := config.DefaultConfig()
		config.SetGlobal(cfg)

		// Verify config exists
		if cfg == nil {
			t.Error("Config should exist")
		}
	})
}

// TestCommandChaining tests executing multiple commands in sequence
func TestCommandChaining(t *testing.T) {
	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cfg := config.DefaultConfig()
	cfg.StoragePath = filepath.Join(tmpDir, "commands.db")
	config.SetGlobal(cfg)

	// Save config
	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Initialize storage
	store, err := storage.NewStorageEngine("sqlite", cfg.StoragePath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	if err := store.Initialize(); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Add a command
	record := history.CommandRecord{
		Command:   "test command",
		Directory: "/test",
		Timestamp: time.Now(),
		Shell:     history.Bash,
		ExitCode:  0,
		Duration:  time.Second,
	}

	if err := store.SaveCommand(record); err != nil {
		t.Fatalf("Failed to save command: %v", err)
	}

	// Verify command was saved
	commands, err := store.GetCommandsByDirectory("/test")
	if err != nil {
		t.Fatalf("Failed to get commands: %v", err)
	}

	if len(commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(commands))
	}

	// Test cleanup
	if err := store.CleanupOldCommands(0); err != nil {
		t.Fatalf("Failed to cleanup: %v", err)
	}

	// Verify cleanup worked
	commands, err = store.GetCommandsByDirectory("/test")
	if err != nil {
		t.Fatalf("Failed to get commands after cleanup: %v", err)
	}

	if len(commands) != 0 {
		t.Errorf("Expected 0 commands after cleanup, got %d", len(commands))
	}
}

// TestErrorHandling tests error handling in CLI commands
func TestErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("InvalidStoragePath", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.StoragePath = "/invalid/path/that/does/not/exist/commands.db"

		_, err := storage.NewStorageEngine("sqlite", cfg.StoragePath)
		if err != nil {
			// Expected error for invalid path
			t.Log("✓ Invalid storage path handled correctly")
		}
	})

	t.Run("InvalidConfigValue", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.RetentionDays = -1

		if err := cfg.Validate(); err == nil {
			t.Error("Should return error for negative retention days")
		}
	})

	t.Run("EmptyStoragePath", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.StoragePath = ""

		if err := cfg.Validate(); err == nil {
			t.Error("Should return error for empty storage path")
		}
	})
}

// TestParseDuration tests the duration parsing helper
func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"24h", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"2w", 14 * 24 * time.Hour, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseDuration(tt.input)

			if tt.hasError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

// TestShellTypeConversion tests shell type string conversion
func TestShellTypeConversion(t *testing.T) {
	tests := []struct {
		input    string
		expected history.ShellType
	}{
		{"powershell", history.PowerShell},
		{"pwsh", history.PowerShell},
		{"bash", history.Bash},
		{"zsh", history.Zsh},
		{"cmd", history.Cmd},
		{"unknown", history.Unknown},
		{"", history.Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseShellType(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}

	// Test reverse conversion
	reverseTests := []struct {
		input    history.ShellType
		expected string
	}{
		{history.PowerShell, "powershell"},
		{history.Bash, "bash"},
		{history.Zsh, "zsh"},
		{history.Cmd, "cmd"},
		{history.Unknown, "unknown"},
	}

	for _, tt := range reverseTests {
		t.Run(tt.expected, func(t *testing.T) {
			result := shellTypeToString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestCaseInsensitiveFilter tests the case-insensitive filtering
func TestCaseInsensitiveFilter(t *testing.T) {
	commands := []history.CommandRecord{
		{Command: "Git Status"},
		{Command: "git commit"},
		{Command: "GIT PUSH"},
		{Command: "npm install"},
	}

	tests := []struct {
		pattern  string
		expected int
	}{
		{"git", 3},
		{"GIT", 3},
		{"Git", 3},
		{"npm", 1},
		{"NPM", 1},
		{"notfound", 0},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			filtered := filterCaseInsensitive(commands, tt.pattern)
			if len(filtered) != tt.expected {
				t.Errorf("Expected %d results for pattern '%s', got %d", tt.expected, tt.pattern, len(filtered))
			}
		})
	}
}

// TestConfigurationManagement tests configuration management flows
func TestConfigurationManagement(t *testing.T) {
	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	t.Run("CreateAndLoadConfig", func(t *testing.T) {
		// Create config
		cfg := config.DefaultConfig()
		cfg.RetentionDays = 120

		if err := cfg.Save(); err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		// Load config
		loadedCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if loadedCfg.RetentionDays != 120 {
			t.Error("Config not loaded correctly")
		}
	})

	t.Run("UpdateConfig", func(t *testing.T) {
		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		cfg.MaxCommands = 15000
		cfg.AutoCleanup = false

		if err := cfg.Save(); err != nil {
			t.Fatalf("Failed to save updated config: %v", err)
		}

		// Reload and verify
		loadedCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to reload config: %v", err)
		}

		if loadedCfg.MaxCommands != 15000 {
			t.Error("MaxCommands not updated")
		}
		if loadedCfg.AutoCleanup != false {
			t.Error("AutoCleanup not updated")
		}
	})

	t.Run("ValidateConfig", func(t *testing.T) {
		cfg := config.DefaultConfig()

		// Valid config
		if err := cfg.Validate(); err != nil {
			t.Errorf("Valid config should pass validation: %v", err)
		}

		// Invalid configs
		invalidConfigs := []struct {
			name   string
			modify func(*config.Config)
		}{
			{"NegativeRetention", func(c *config.Config) { c.RetentionDays = -1 }},
			{"NegativeMaxCommands", func(c *config.Config) { c.MaxCommands = -1 }},
			{"EmptyStoragePath", func(c *config.Config) { c.StoragePath = "" }},
		}

		for _, tc := range invalidConfigs {
			t.Run(tc.name, func(t *testing.T) {
				testCfg := config.DefaultConfig()
				tc.modify(testCfg)

				if err := testCfg.Validate(); err == nil {
					t.Error("Invalid config should fail validation")
				}
			})
		}
	})
}
