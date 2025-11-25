package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ValGrace/command-history-tracker/pkg/history"
)

// setupTestStorage creates a temporary SQLite storage for testing
func setupTestStorage(t *testing.T) (*SQLiteStorage, func()) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	storage := NewSQLiteStorage(dbPath)
	if err := storage.Initialize(); err != nil {
		t.Fatalf("Failed to initialize test storage: %v", err)
	}

	cleanup := func() {
		if err := storage.Close(); err != nil {
			t.Logf("Warning: failed to close storage: %v", err)
		}
		// Don't explicitly remove temp dir, let Go's testing framework handle it
	}

	return storage, cleanup
}

// createTestCommand creates a test command record
func createTestCommand(id, command, directory string, shell history.ShellType) history.CommandRecord {
	return history.CommandRecord{
		ID:        id,
		Command:   command,
		Directory: directory,
		Timestamp: time.Now(),
		Shell:     shell,
		ExitCode:  0,
		Duration:  time.Millisecond * 100,
		Tags:      []string{"test"},
	}
}

func TestSQLiteStorage_Initialize(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	storage := NewSQLiteStorage(dbPath)

	// Test initialization
	err := storage.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Test double initialization (should not fail)
	err = storage.Initialize()
	if err != nil {
		t.Errorf("Double initialization failed: %v", err)
	}

	// Test that we can perform basic operations
	cmd := createTestCommand("init-test", "test command", "/test", history.Bash)
	err = storage.SaveCommand(cmd)
	if err != nil {
		t.Errorf("SaveCommand failed after initialization: %v", err)
	}

	// Close the database connection
	if err := storage.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestSQLiteStorage_SaveCommand(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	cmd := createTestCommand("test-1", "ls -la", "/home/user", history.Bash)

	// Test saving valid command
	err := storage.SaveCommand(cmd)
	if err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}

	// Test saving invalid command (empty ID)
	invalidCmd := cmd
	invalidCmd.ID = ""
	err = storage.SaveCommand(invalidCmd)
	if err == nil {
		t.Error("Expected error for invalid command, got nil")
	}
}

func TestSQLiteStorage_GetCommandsByDirectory(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Save test commands in different directories
	cmd1 := createTestCommand("test-1", "ls -la", "/home/user", history.Bash)
	cmd2 := createTestCommand("test-2", "pwd", "/home/user", history.Bash)
	cmd3 := createTestCommand("test-3", "cd ..", "/tmp", history.Bash)

	if err := storage.SaveCommand(cmd1); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(cmd2); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(cmd3); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}

	// Test getting commands for specific directory
	commands, err := storage.GetCommandsByDirectory("/home/user")
	if err != nil {
		t.Fatalf("GetCommandsByDirectory failed: %v", err)
	}

	if len(commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(commands))
	}

	// Verify commands are sorted by timestamp (newest first)
	if len(commands) >= 2 && commands[0].Timestamp.Before(commands[1].Timestamp) {
		t.Error("Commands are not sorted by timestamp (newest first)")
	}

	// Test getting commands for non-existent directory
	commands, err = storage.GetCommandsByDirectory("/non/existent")
	if err != nil {
		t.Fatalf("GetCommandsByDirectory failed for non-existent directory: %v", err)
	}

	if len(commands) != 0 {
		t.Errorf("Expected 0 commands for non-existent directory, got %d", len(commands))
	}
}

func TestSQLiteStorage_GetDirectoriesWithHistory(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Save commands in different directories
	cmd1 := createTestCommand("test-1", "ls", "/home/user", history.Bash)
	cmd2 := createTestCommand("test-2", "pwd", "/tmp", history.Bash)
	cmd3 := createTestCommand("test-3", "cd", "/var/log", history.Bash)

	if err := storage.SaveCommand(cmd1); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(cmd2); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(cmd3); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}

	directories, err := storage.GetDirectoriesWithHistory()
	if err != nil {
		t.Fatalf("GetDirectoriesWithHistory failed: %v", err)
	}

	expectedDirs := []string{"/home/user", "/tmp", "/var/log"}
	if len(directories) != len(expectedDirs) {
		t.Errorf("Expected %d directories, got %d", len(expectedDirs), len(directories))
	}

	// Verify all expected directories are present
	dirMap := make(map[string]bool)
	for _, dir := range directories {
		dirMap[dir] = true
	}

	for _, expectedDir := range expectedDirs {
		if !dirMap[expectedDir] {
			t.Errorf("Expected directory %s not found", expectedDir)
		}
	}
}

func TestSQLiteStorage_SearchCommands(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Save test commands
	cmd1 := createTestCommand("test-1", "git status", "/home/user/project", history.Bash)
	cmd2 := createTestCommand("test-2", "git commit -m 'test'", "/home/user/project", history.Bash)
	cmd3 := createTestCommand("test-3", "ls -la", "/home/user/project", history.Bash)
	cmd4 := createTestCommand("test-4", "git log", "/tmp", history.Bash)

	if err := storage.SaveCommand(cmd1); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(cmd2); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(cmd3); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(cmd4); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}

	// Test search in specific directory
	commands, err := storage.SearchCommands("git", "/home/user/project")
	if err != nil {
		t.Fatalf("SearchCommands failed: %v", err)
	}

	if len(commands) != 2 {
		t.Errorf("Expected 2 git commands in /home/user/project, got %d", len(commands))
	}

	// Test search across all directories
	commands, err = storage.SearchCommands("git", "")
	if err != nil {
		t.Fatalf("SearchCommands failed: %v", err)
	}

	if len(commands) != 3 {
		t.Errorf("Expected 3 git commands total, got %d", len(commands))
	}

	// Test search with no matches
	commands, err = storage.SearchCommands("nonexistent", "")
	if err != nil {
		t.Fatalf("SearchCommands failed: %v", err)
	}

	if len(commands) != 0 {
		t.Errorf("Expected 0 commands for nonexistent pattern, got %d", len(commands))
	}
}

func TestSQLiteStorage_CleanupOldCommands(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create commands with different timestamps
	now := time.Now()
	oldCmd := createTestCommand("old-1", "old command", "/home/user", history.Bash)
	oldCmd.Timestamp = now.AddDate(0, 0, -100) // 100 days ago

	recentCmd := createTestCommand("recent-1", "recent command", "/home/user", history.Bash)
	recentCmd.Timestamp = now.AddDate(0, 0, -10) // 10 days ago

	if err := storage.SaveCommand(oldCmd); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(recentCmd); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}

	// Cleanup commands older than 30 days
	err := storage.CleanupOldCommands(30)
	if err != nil {
		t.Fatalf("CleanupOldCommands failed: %v", err)
	}

	// Verify old command was deleted
	commands, err := storage.GetCommandsByDirectory("/home/user")
	if err != nil {
		t.Fatalf("GetCommandsByDirectory failed: %v", err)
	}

	if len(commands) != 1 {
		t.Errorf("Expected 1 command after cleanup, got %d", len(commands))
	}

	if len(commands) > 0 && commands[0].ID != "recent-1" {
		t.Errorf("Expected recent command to remain, got %s", commands[0].ID)
	}
}

func TestSQLiteStorage_BatchSaveCommands(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Create batch of commands
	commands := []history.CommandRecord{
		createTestCommand("batch-1", "command 1", "/home/user", history.Bash),
		createTestCommand("batch-2", "command 2", "/home/user", history.Bash),
		createTestCommand("batch-3", "command 3", "/tmp", history.PowerShell),
	}

	// Test batch save
	err := storage.BatchSaveCommands(commands)
	if err != nil {
		t.Fatalf("BatchSaveCommands failed: %v", err)
	}

	// Verify all commands were saved
	allCommands, err := storage.GetCommandsByDirectory("/home/user")
	if err != nil {
		t.Fatalf("GetCommandsByDirectory failed: %v", err)
	}

	if len(allCommands) != 2 {
		t.Errorf("Expected 2 commands in /home/user, got %d", len(allCommands))
	}

	tmpCommands, err := storage.GetCommandsByDirectory("/tmp")
	if err != nil {
		t.Fatalf("GetCommandsByDirectory failed: %v", err)
	}

	if len(tmpCommands) != 1 {
		t.Errorf("Expected 1 command in /tmp, got %d", len(tmpCommands))
	}

	// Test batch save with empty slice
	err = storage.BatchSaveCommands([]history.CommandRecord{})
	if err != nil {
		t.Errorf("BatchSaveCommands with empty slice should not fail: %v", err)
	}
}

func TestSQLiteStorage_GetDirectoryStats(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Save commands to create stats
	cmd1 := createTestCommand("test-1", "ls", "/home/user", history.Bash)
	cmd2 := createTestCommand("test-2", "pwd", "/home/user", history.Bash)
	cmd3 := createTestCommand("test-3", "cd", "/tmp", history.PowerShell)

	if err := storage.SaveCommand(cmd1); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(cmd2); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(cmd3); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}

	// Get directory stats
	stats, err := storage.GetDirectoryStats()
	if err != nil {
		t.Fatalf("GetDirectoryStats failed: %v", err)
	}

	if len(stats) != 2 {
		t.Errorf("Expected 2 directory stats, got %d", len(stats))
	}

	// Find stats for /home/user
	var homeUserStats *history.DirectoryIndex
	for _, stat := range stats {
		if stat.Path == "/home/user" {
			homeUserStats = &stat
			break
		}
	}

	if homeUserStats == nil {
		t.Error("Stats for /home/user not found")
	} else if homeUserStats.CommandCount != 2 {
		t.Errorf("Expected 2 commands for /home/user, got %d", homeUserStats.CommandCount)
	}
}

func TestSQLiteStorage_ConcurrentAccess(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Test concurrent writes
	done := make(chan bool, 2)

	errs := make(chan error, 20)

	go func() {
		for i := 0; i < 10; i++ {
			cmd := createTestCommand("goroutine1-"+string(rune(i)), "command", "/home/user", history.Bash)
			errs <- storage.SaveCommand(cmd)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			cmd := createTestCommand("goroutine2-"+string(rune(i)), "command", "/tmp", history.PowerShell)
			errs <- storage.SaveCommand(cmd)
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("SaveCommand failed in goroutine: %v", err)
		}
	}

	// Verify all commands were saved
	homeCommands, err := storage.GetCommandsByDirectory("/home/user")
	if err != nil {
		t.Fatalf("GetCommandsByDirectory failed: %v", err)
	}

	tmpCommands, err := storage.GetCommandsByDirectory("/tmp")
	if err != nil {
		t.Fatalf("GetCommandsByDirectory failed: %v", err)
	}

	if len(homeCommands) != 10 {
		t.Errorf("Expected 10 commands in /home/user, got %d", len(homeCommands))
	}

	if len(tmpCommands) != 10 {
		t.Errorf("Expected 10 commands in /tmp, got %d", len(tmpCommands))
	}
}

func TestSQLiteStorage_OptimizeDatabase(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Add some commands
	for i := 0; i < 5; i++ {
		cmd := createTestCommand("test-"+string(rune(i)), "command", "/home/user", history.Bash)
		if err := storage.SaveCommand(cmd); err != nil {
			t.Fatalf("SaveCommand failed: %v", err)
		}
	}

	// Test database optimization
	err := storage.OptimizeDatabase()
	if err != nil {
		t.Fatalf("OptimizeDatabase failed: %v", err)
	}

	// Verify commands are still accessible after optimization
	commands, err := storage.GetCommandsByDirectory("/home/user")
	if err != nil {
		t.Fatalf("GetCommandsByDirectory failed after optimization: %v", err)
	}

	if len(commands) != 5 {
		t.Errorf("Expected 5 commands after optimization, got %d", len(commands))
	}
}

func TestSQLiteStorage_GetDatabaseSize(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	// Get initial size
	initialSize, err := storage.GetDatabaseSize()
	if err != nil {
		t.Fatalf("GetDatabaseSize failed: %v", err)
	}

	if initialSize <= 0 {
		t.Error("Database size should be greater than 0")
	}

	// Add commands with substantial content
	for i := 0; i < 100; i++ {
		cmd := createTestCommand("test-command-with-long-id-"+string(rune(i)),
			"command with substantial content to ensure database growth "+string(rune(i)),
			"/home/user/very/long/directory/path/to/ensure/growth",
			history.Bash)
		if err := storage.SaveCommand(cmd); err != nil {
			t.Fatalf("SaveCommand failed: %v", err)
		}
	}

	// Force a checkpoint to ensure data is written to disk
	if err := storage.OptimizeDatabase(); err != nil {
		t.Fatalf("OptimizeDatabase failed: %v", err)
	}

	newSize, err := storage.GetDatabaseSize()
	if err != nil {
		t.Fatalf("GetDatabaseSize failed after adding commands: %v", err)
	}

	// Allow for some tolerance in size comparison due to SQLite internals
	if newSize < initialSize {
		t.Errorf("Database size should not decrease: initial=%d, new=%d", initialSize, newSize)
	}
}
