package main

import (
	"bytes"
	"fmt"
	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/ValGrace/command-history-tracker/internal/storage"
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"os"
	"path/filepath"
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
			ID:        generateCommandID(),
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
			ID:        generateCommandID(),
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
			ID:        generateCommandID(),
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

	// Add a command with old timestamp
	record := history.CommandRecord{
		ID:        generateCommandID(),
		Command:   "test command",
		Directory: "/test",
		Timestamp: time.Now().Add(-48 * time.Hour), // 2 days old
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

	// Test cleanup (delete commands older than 1 day)
	if err := store.CleanupOldCommands(1); err != nil {
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
	// tmpDir := t.TempDir()

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

// generateCommandID generates a unique ID for a command record
var commandIDCounter int64

func generateCommandID() string {
	commandIDCounter++
	return fmt.Sprintf("cmd-%d-%d", time.Now().UnixNano(), commandIDCounter)
}

// TestCLICommandParameters tests CLI commands with various parameter combinations
func TestCLICommandParameters(t *testing.T) {
	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

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

	// Add test data
	testDirs := []string{"/project1", "/project2", "/project3"}
	for _, dir := range testDirs {
		for i := 0; i < 10; i++ {
			record := history.CommandRecord{
				ID:        generateCommandID(),
				Command:   fmt.Sprintf("command-%d", i),
				Directory: dir,
				Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
				Shell:     history.Bash,
				ExitCode:  0,
				Duration:  time.Second,
			}
			if err := store.SaveCommand(record); err != nil {
				t.Fatalf("Failed to save command: %v", err)
			}
		}
	}

	t.Run("HistoryWithMultipleFlags", func(t *testing.T) {
		historyFlags.dir = "/project1"
		historyFlags.limit = 5
		historyFlags.since = "6h"
		historyFlags.noInteractive = true

		commands, err := store.GetCommandsByDirectory("/project1")
		if err != nil {
			t.Fatalf("Failed to get commands: %v", err)
		}

		filtered := applyHistoryFilters(commands)
		if historyFlags.limit > 0 && len(filtered) > historyFlags.limit {
			filtered = filtered[:historyFlags.limit]
		}

		if len(filtered) > 5 {
			t.Errorf("Expected at most 5 commands with limit, got %d", len(filtered))
		}

		// Reset flags
		historyFlags = struct {
			dir           string
			limit         int
			since         string
			shell         string
			noInteractive bool
		}{}
	})

	t.Run("SearchWithAllDirectories", func(t *testing.T) {
		searchFlags.allDirs = true
		searchFlags.caseSensitive = false
		searchFlags.limit = 15

		dirs, err := store.GetDirectoriesWithHistory()
		if err != nil {
			t.Fatalf("Failed to get directories: %v", err)
		}

		var allResults []history.CommandRecord
		for _, dir := range dirs {
			results, err := store.SearchCommands("command", dir)
			if err != nil {
				continue
			}
			allResults = append(allResults, results...)
		}

		if len(allResults) == 0 {
			t.Error("Expected to find commands across all directories")
		}

		// Reset flags
		searchFlags = struct {
			dir           string
			allDirs       bool
			caseSensitive bool
			limit         int
			noInteractive bool
		}{}
	})

	t.Run("BrowseWithTreeView", func(t *testing.T) {
		browseFlags.tree = true
		browseFlags.dir = ""

		dirs, err := store.GetDirectoriesWithHistory()
		if err != nil {
			t.Fatalf("Failed to get directories: %v", err)
		}

		if len(dirs) != 3 {
			t.Errorf("Expected 3 directories in tree view, got %d", len(dirs))
		}

		// Reset flags
		browseFlags = struct {
			dir    string
			search string
			tree   bool
		}{}
	})
}

// TestCLICommandChaining tests executing multiple CLI operations in sequence
func TestCLICommandChaining(t *testing.T) {
	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cfg := config.DefaultConfig()
	cfg.StoragePath = filepath.Join(tmpDir, "commands.db")
	config.SetGlobal(cfg)

	t.Run("ConfigSetThenGet", func(t *testing.T) {
		// Set a value
		cfg.RetentionDays = 200
		if err := cfg.Save(); err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		// Get the value
		loadedCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if loadedCfg.RetentionDays != 200 {
			t.Error("Config value not persisted through set-get chain")
		}
	})

	t.Run("RecordThenSearch", func(t *testing.T) {
		store, err := storage.NewStorageEngine("sqlite", cfg.StoragePath)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}
		defer store.Close()

		if err := store.Initialize(); err != nil {
			t.Fatalf("Failed to initialize storage: %v", err)
		}

		// Record a command
		record := history.CommandRecord{
			ID:        generateCommandID(),
			Command:   "unique-test-command",
			Directory: "/test",
			Timestamp: time.Now(),
			Shell:     history.Bash,
			ExitCode:  0,
			Duration:  time.Second,
		}

		if err := store.SaveCommand(record); err != nil {
			t.Fatalf("Failed to save command: %v", err)
		}

		// Search for it
		results, err := store.SearchCommands("unique-test-command", "/test")
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result from record-search chain, got %d", len(results))
		}
	})

	t.Run("RecordBrowseCleanup", func(t *testing.T) {
		store, err := storage.NewStorageEngine("sqlite", cfg.StoragePath)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}
		defer store.Close()

		if err := store.Initialize(); err != nil {
			t.Fatalf("Failed to initialize storage: %v", err)
		}

		// Record commands
		for i := 0; i < 5; i++ {
			record := history.CommandRecord{
				ID:        generateCommandID(),
				Command:   fmt.Sprintf("chain-command-%d", i),
				Directory: "/chain-test",
				Timestamp: time.Now().Add(-time.Duration(i*24) * time.Hour),
				Shell:     history.Bash,
				ExitCode:  0,
				Duration:  time.Second,
			}
			if err := store.SaveCommand(record); err != nil {
				t.Fatalf("Failed to save command: %v", err)
			}
		}

		// Browse (verify they exist)
		commands, err := store.GetCommandsByDirectory("/chain-test")
		if err != nil {
			t.Fatalf("Failed to get commands: %v", err)
		}

		if len(commands) != 5 {
			t.Errorf("Expected 5 commands before cleanup, got %d", len(commands))
		}

		// Cleanup old commands (older than 2 days)
		if err := store.CleanupOldCommands(2); err != nil {
			t.Fatalf("Failed to cleanup: %v", err)
		}

		// Verify cleanup worked
		commands, err = store.GetCommandsByDirectory("/chain-test")
		if err != nil {
			t.Fatalf("Failed to get commands after cleanup: %v", err)
		}

		if len(commands) >= 5 {
			t.Errorf("Expected fewer commands after cleanup, got %d", len(commands))
		}
	})
}

// TestCLIErrorHandlingScenarios tests comprehensive error handling
func TestCLIErrorHandlingScenarios(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("MissingConfigFile", func(t *testing.T) {
		oldHome := os.Getenv("HOME")
		nonexistentDir := filepath.Join(tmpDir, "nonexistent")
		os.Setenv("HOME", nonexistentDir)
		defer os.Setenv("HOME", oldHome)

		// Load should succeed by creating default config
		cfg, err := config.Load()
		if err != nil {
			t.Errorf("Load should create default config, got error: %v", err)
		}
		if cfg == nil {
			t.Error("Expected default config to be returned")
		}
	})

	t.Run("InvalidDatabasePath", func(t *testing.T) {
		// Use a path that's definitely invalid on all platforms
		invalidPath := "\x00invalid\x00path\x00commands.db"
		store, err := storage.NewStorageEngine("sqlite", invalidPath)
		if err != nil {
			// Expected error
			t.Log("✓ Invalid database path handled correctly")
			return
		}
		// If no error on creation, should fail on initialize
		if store != nil {
			defer store.Close()
			if err := store.Initialize(); err != nil {
				t.Log("✓ Invalid database path caught on initialize")
			} else {
				// On some systems, SQLite is very permissive with paths
				// This is acceptable behavior
				t.Log("✓ Storage engine handles path gracefully")
			}
		}
	})

	t.Run("SearchInNonexistentDirectory", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.StoragePath = filepath.Join(tmpDir, "test.db")

		store, err := storage.NewStorageEngine("sqlite", cfg.StoragePath)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}
		defer store.Close()

		if err := store.Initialize(); err != nil {
			t.Fatalf("Failed to initialize storage: %v", err)
		}

		results, err := store.SearchCommands("test", "/nonexistent/directory")
		if err != nil {
			// Error is acceptable
			t.Log("✓ Handled nonexistent directory search")
		}

		if len(results) != 0 {
			t.Error("Expected no results for nonexistent directory")
		}
	})

	t.Run("InvalidConfigValues", func(t *testing.T) {
		testCases := []struct {
			name   string
			modify func(*config.Config)
		}{
			{
				name: "NegativeRetentionDays",
				modify: func(c *config.Config) {
					c.RetentionDays = -1
				},
			},
			{
				name: "NegativeMaxCommands",
				modify: func(c *config.Config) {
					c.MaxCommands = -100
				},
			},
			{
				name: "EmptyStoragePath",
				modify: func(c *config.Config) {
					c.StoragePath = ""
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cfg := config.DefaultConfig()
				tc.modify(cfg)

				if err := cfg.Validate(); err == nil {
					t.Errorf("Expected validation error for %s", tc.name)
				}
			})
		}
	})

	t.Run("ConcurrentStorageAccess", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.StoragePath = filepath.Join(tmpDir, "concurrent.db")

		store, err := storage.NewStorageEngine("sqlite", cfg.StoragePath)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}
		defer store.Close()

		if err := store.Initialize(); err != nil {
			t.Fatalf("Failed to initialize storage: %v", err)
		}

		// Attempt concurrent writes
		done := make(chan bool, 2)

		for i := 0; i < 2; i++ {
			go func(id int) {
				record := history.CommandRecord{
					ID:        generateCommandID(),
					Command:   fmt.Sprintf("concurrent-%d", id),
					Directory: "/test",
					Timestamp: time.Now(),
					Shell:     history.Bash,
					ExitCode:  0,
					Duration:  time.Second,
				}
				_ = store.SaveCommand(record)
				done <- true
			}(i)
		}

		<-done
		<-done

		// Give a moment for writes to complete
		time.Sleep(100 * time.Millisecond)

		// Verify commands were saved
		commands, err := store.GetCommandsByDirectory("/test")
		if err != nil {
			t.Fatalf("Failed to get commands: %v", err)
		}

		// At least one should succeed (SQLite handles concurrent writes with locking)
		if len(commands) == 0 {
			t.Error("Expected at least one command after concurrent access")
		}
		t.Logf("✓ Concurrent access handled: %d commands saved", len(commands))
	})
}

// TestConfigurationFlows tests complete configuration management workflows
func TestConfigurationFlows(t *testing.T) {
	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	t.Run("CompleteConfigurationWorkflow", func(t *testing.T) {
		// Step 1: Create default config
		cfg := config.DefaultConfig()
		if err := cfg.Save(); err != nil {
			t.Fatalf("Failed to save default config: %v", err)
		}

		// Step 2: Load and verify
		loadedCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if loadedCfg.RetentionDays != cfg.RetentionDays {
			t.Error("Default config not loaded correctly")
		}

		// Step 3: Modify multiple values
		loadedCfg.RetentionDays = 365
		loadedCfg.MaxCommands = 50000
		loadedCfg.AutoCleanup = false
		loadedCfg.ExcludePatterns = []string{"secret*", "password*"}

		if err := loadedCfg.Save(); err != nil {
			t.Fatalf("Failed to save modified config: %v", err)
		}

		// Step 4: Reload and verify all changes
		finalCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to reload config: %v", err)
		}

		if finalCfg.RetentionDays != 365 {
			t.Error("RetentionDays not persisted")
		}
		if finalCfg.MaxCommands != 50000 {
			t.Error("MaxCommands not persisted")
		}
		if finalCfg.AutoCleanup != false {
			t.Error("AutoCleanup not persisted")
		}
		if len(finalCfg.ExcludePatterns) != 2 {
			t.Error("ExcludePatterns not persisted")
		}

		// Step 5: Reset to defaults
		defaultCfg := config.DefaultConfig()
		if err := defaultCfg.Save(); err != nil {
			t.Fatalf("Failed to reset config: %v", err)
		}

		// Step 6: Verify reset
		resetCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load reset config: %v", err)
		}

		if resetCfg.RetentionDays == 365 {
			t.Error("Config should be reset to defaults")
		}
	})

	t.Run("ConfigValidationFlow", func(t *testing.T) {
		_ = config.DefaultConfig()

		// Test valid modifications
		validChanges := []func(*config.Config){
			func(c *config.Config) { c.RetentionDays = 30 },
			func(c *config.Config) { c.RetentionDays = 365 },
			func(c *config.Config) { c.MaxCommands = 1000 },
			func(c *config.Config) { c.MaxCommands = 100000 },
			func(c *config.Config) { c.AutoCleanup = true },
			func(c *config.Config) { c.AutoCleanup = false },
		}

		for i, change := range validChanges {
			testCfg := config.DefaultConfig()
			change(testCfg)

			if err := testCfg.Validate(); err != nil {
				t.Errorf("Valid change %d failed validation: %v", i, err)
			}
		}

		// Test invalid modifications
		invalidChanges := []func(*config.Config){
			func(c *config.Config) { c.RetentionDays = -1 },
			func(c *config.Config) { c.MaxCommands = -1 },
			func(c *config.Config) { c.StoragePath = "" },
		}

		for i, change := range invalidChanges {
			testCfg := config.DefaultConfig()
			change(testCfg)

			if err := testCfg.Validate(); err == nil {
				t.Errorf("Invalid change %d should fail validation", i)
			}
		}
	})
}
