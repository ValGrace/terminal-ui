package storage

import (
	"testing"
	"time"

	"github.com/ValGrace/command-history-tracker/pkg/history"
)

func TestOptimizationEngine_ApplyCleanupPolicy(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	optimizer := NewOptimizationEngine(storage)

	// Create commands with different ages
	now := time.Now()

	// Old commands (should be cleaned up)
	oldCmd1 := createTestCommand("old-1", "old command 1", "/home/user", history.Bash)
	oldCmd1.Timestamp = now.AddDate(0, 0, -100)

	oldCmd2 := createTestCommand("old-2", "old command 2", "/home/user", history.Bash)
	oldCmd2.Timestamp = now.AddDate(0, 0, -95)

	// Recent commands (should be kept)
	recentCmd := createTestCommand("recent-1", "recent command", "/home/user", history.Bash)
	recentCmd.Timestamp = now.AddDate(0, 0, -10)

	// Git command (should be kept due to keep pattern)
	gitCmd := createTestCommand("git-1", "git status", "/home/user", history.Bash)
	gitCmd.Timestamp = now.AddDate(0, 0, -100)

	if err := storage.SaveCommand(oldCmd1); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(oldCmd2); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(recentCmd); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(gitCmd); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}

	// Apply cleanup policy
	policy := &CleanupPolicy{
		MaxAge:       90 * 24 * time.Hour, // 90 days
		MaxCommands:  1000,
		KeepPatterns: []string{"git"},
	}

	err := optimizer.ApplyCleanupPolicy(policy)
	if err != nil {
		t.Fatalf("ApplyCleanupPolicy failed: %v", err)
	}

	// Verify results
	commands, err := storage.GetCommandsByDirectory("/home/user")
	if err != nil {
		t.Fatalf("GetCommandsByDirectory failed: %v", err)
	}

	// Should have recent command and git command
	if len(commands) < 2 {
		t.Errorf("Expected at least 2 commands after cleanup, got %d", len(commands))
	}

	// Verify git command was kept
	gitFound := false
	recentFound := false
	for _, cmd := range commands {
		if cmd.ID == "git-1" {
			gitFound = true
		}
		if cmd.ID == "recent-1" {
			recentFound = true
		}
	}

	if !gitFound {
		t.Error("Git command should have been kept due to keep pattern")
	}
	if !recentFound {
		t.Error("Recent command should have been kept")
	}
}

func TestOptimizationEngine_GetStorageStats(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	optimizer := NewOptimizationEngine(storage)

	// Add commands in different directories with different shells
	cmd1 := createTestCommand("test-1", "ls", "/home/user", history.Bash)
	cmd2 := createTestCommand("test-2", "pwd", "/home/user", history.PowerShell)
	cmd3 := createTestCommand("test-3", "cd", "/tmp", history.Zsh)

	if err := storage.SaveCommand(cmd1); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(cmd2); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(cmd3); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}

	// Get storage stats
	stats, err := optimizer.GetStorageStats()
	if err != nil {
		t.Fatalf("GetStorageStats failed: %v", err)
	}

	// Verify stats
	if stats.TotalDirectories != 2 {
		t.Errorf("Expected 2 directories, got %d", stats.TotalDirectories)
	}

	if stats.TotalCommands != 3 {
		t.Errorf("Expected 3 total commands, got %d", stats.TotalCommands)
	}

	// Check directory-specific stats
	homeUserStats, exists := stats.DirectoryStats["/home/user"]
	if !exists {
		t.Error("Stats for /home/user not found")
	} else {
		if homeUserStats.CommandCount != 2 {
			t.Errorf("Expected 2 commands in /home/user, got %d", homeUserStats.CommandCount)
		}

		// Check shell counts
		if homeUserStats.ShellCounts[history.Bash] != 1 {
			t.Errorf("Expected 1 Bash command in /home/user, got %d", homeUserStats.ShellCounts[history.Bash])
		}
		if homeUserStats.ShellCounts[history.PowerShell] != 1 {
			t.Errorf("Expected 1 PowerShell command in /home/user, got %d", homeUserStats.ShellCounts[history.PowerShell])
		}
	}

	tmpStats, exists := stats.DirectoryStats["/tmp"]
	if !exists {
		t.Error("Stats for /tmp not found")
	} else {
		if tmpStats.CommandCount != 1 {
			t.Errorf("Expected 1 command in /tmp, got %d", tmpStats.CommandCount)
		}
		if tmpStats.ShellCounts[history.Zsh] != 1 {
			t.Errorf("Expected 1 Zsh command in /tmp, got %d", tmpStats.ShellCounts[history.Zsh])
		}
	}
}

func TestDefaultCleanupPolicy(t *testing.T) {
	policy := DefaultCleanupPolicy()

	if policy.MaxAge != 90*24*time.Hour {
		t.Errorf("Expected MaxAge to be 90 days, got %v", policy.MaxAge)
	}

	if policy.MaxCommands != 1000 {
		t.Errorf("Expected MaxCommands to be 1000, got %d", policy.MaxCommands)
	}

	if len(policy.ExcludeShells) != 0 {
		t.Errorf("Expected no excluded shells by default, got %d", len(policy.ExcludeShells))
	}

	expectedPatterns := []string{"git", "docker", "kubectl"}
	if len(policy.KeepPatterns) != len(expectedPatterns) {
		t.Errorf("Expected %d keep patterns, got %d", len(expectedPatterns), len(policy.KeepPatterns))
	}
}

func TestRetentionManager(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	policy := &CleanupPolicy{
		MaxAge:       1 * time.Hour, // Very short for testing
		MaxCommands:  5,
		KeepPatterns: []string{"important"},
	}

	manager := NewRetentionManager(storage, policy)

	// Add some test commands
	now := time.Now()
	oldCmd := createTestCommand("old-1", "old command", "/home/user", history.Bash)
	oldCmd.Timestamp = now.Add(-2 * time.Hour) // Older than policy

	importantCmd := createTestCommand("important-1", "important command", "/home/user", history.Bash)
	importantCmd.Timestamp = now.Add(-2 * time.Hour) // Old but should be kept

	recentCmd := createTestCommand("recent-1", "recent command", "/home/user", history.Bash)
	recentCmd.Timestamp = now.Add(-30 * time.Minute) // Recent

	if err := storage.SaveCommand(oldCmd); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(importantCmd); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}
	if err := storage.SaveCommand(recentCmd); err != nil {
		t.Fatalf("SaveCommand failed: %v", err)
	}

	// Test manual cleanup
	err := manager.optimization.ApplyCleanupPolicy(policy)
	if err != nil {
		t.Fatalf("Manual cleanup failed: %v", err)
	}

	// Verify results
	commands, err := storage.GetCommandsByDirectory("/home/user")
	if err != nil {
		t.Fatalf("GetCommandsByDirectory failed: %v", err)
	}

	// Should have recent and important commands
	if len(commands) < 2 {
		t.Errorf("Expected at least 2 commands after cleanup, got %d", len(commands))
	}

	// Test policy update
	newPolicy := &CleanupPolicy{
		MaxAge:       24 * time.Hour,
		MaxCommands:  100,
		KeepPatterns: []string{"git", "important"},
	}

	manager.UpdatePolicy(newPolicy)

	if manager.policy.MaxAge != newPolicy.MaxAge {
		t.Error("Policy was not updated correctly")
	}
}

func TestContainsPattern(t *testing.T) {
	tests := []struct {
		command  string
		pattern  string
		expected bool
	}{
		{"git status", "git", true},
		{"git commit -m 'test'", "git", true},
		{"ls -la", "git", false},
		{"docker run nginx", "docker", true},
		{"kubectl get pods", "kubectl", true},
		{"echo hello", "echo", true},
		{"echo hello world", "hello", true},
		{"test", "test", true},
		{"testing", "test", true},
		{"", "test", false},
		{"test", "", true}, // Empty pattern should match
	}

	for _, test := range tests {
		result := containsPattern(test.command, test.pattern)
		if result != test.expected {
			t.Errorf("containsPattern(%q, %q) = %v, expected %v",
				test.command, test.pattern, result, test.expected)
		}
	}
}

func TestOptimizationEngine_OptimizeStorage(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	optimizer := NewOptimizationEngine(storage)

	// Add some test data
	for i := 0; i < 5; i++ {
		cmd := createTestCommand("test-"+string(rune(i)), "command", "/home/user", history.Bash)
		if err := storage.SaveCommand(cmd); err != nil {
			t.Fatalf("SaveCommand failed: %v", err)
		}
	}

	// Test optimization
	err := optimizer.OptimizeStorage()
	if err != nil {
		t.Fatalf("OptimizeStorage failed: %v", err)
	}

	// Verify data is still accessible
	commands, err := storage.GetCommandsByDirectory("/home/user")
	if err != nil {
		t.Fatalf("GetCommandsByDirectory failed after optimization: %v", err)
	}

	if len(commands) != 5 {
		t.Errorf("Expected 5 commands after optimization, got %d", len(commands))
	}
}
