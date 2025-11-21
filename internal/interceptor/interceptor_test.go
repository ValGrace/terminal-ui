package interceptor

import (
	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/ValGrace/command-history-tracker/internal/storage"
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"os"
	"path/filepath"
	"testing"
)

func TestInterceptor_NewInterceptor(t *testing.T) {
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	interceptor := NewInterceptor(storage, cfg)

	if interceptor == nil {
		t.Fatal("NewInterceptor() returned nil")
	}

	if interceptor.IsRecording() {
		t.Error("New interceptor should not be recording initially")
	}

	if interceptor.GetCurrentShell() != history.Unknown {
		t.Error("New interceptor should have Unknown shell initially")
	}
}

func TestInterceptor_DetectShell(t *testing.T) {
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	interceptor := NewInterceptor(storage, cfg)

	shell, err := interceptor.DetectShell()
	if err != nil {
		t.Fatalf("DetectShell() failed: %v", err)
	}

	// Should detect a valid shell type
	if shell <= history.Unknown || shell > history.Cmd {
		t.Errorf("DetectShell() returned invalid shell type: %v", shell)
	}
}

func TestInterceptor_SetupShellIntegration(t *testing.T) {
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	interceptor := NewInterceptor(storage, cfg)

	// Test with a supported shell (Bash should be supported on most platforms)
	err := interceptor.SetupShellIntegration(history.Bash)
	if err != nil {
		t.Fatalf("SetupShellIntegration() failed: %v", err)
	}

	if interceptor.GetCurrentShell() != history.Bash {
		t.Error("Shell should be set after SetupShellIntegration()")
	}
}

func TestInterceptor_StartStopRecording(t *testing.T) {
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	interceptor := NewInterceptor(storage, cfg)

	// Initially not recording
	if interceptor.IsRecording() {
		t.Error("Should not be recording initially")
	}

	// Start recording
	err := interceptor.StartRecording()
	if err != nil {
		t.Fatalf("StartRecording() failed: %v", err)
	}

	if !interceptor.IsRecording() {
		t.Error("Should be recording after StartRecording()")
	}

	// Should have detected and set a shell
	currentShell := interceptor.GetCurrentShell()
	t.Logf("Current shell after StartRecording: %v (%s)", currentShell, currentShell.String())
	if currentShell <= history.Unknown || currentShell > history.Cmd {
		t.Errorf("Invalid shell type detected: %v", currentShell)
	}

	// Stop recording
	err = interceptor.StopRecording()
	if err != nil {
		t.Fatalf("StopRecording() failed: %v", err)
	}

	if interceptor.IsRecording() {
		t.Error("Should not be recording after StopRecording()")
	}
}

func TestInterceptor_StartRecording_AlreadyRecording(t *testing.T) {
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	interceptor := NewInterceptor(storage, cfg)

	// Start recording
	err := interceptor.StartRecording()
	if err != nil {
		t.Fatalf("First StartRecording() failed: %v", err)
	}

	// Start recording again (should not error)
	err = interceptor.StartRecording()
	if err != nil {
		t.Fatalf("Second StartRecording() failed: %v", err)
	}

	if !interceptor.IsRecording() {
		t.Error("Should still be recording")
	}
}

func TestInterceptor_StopRecording_NotRecording(t *testing.T) {
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	interceptor := NewInterceptor(storage, cfg)

	// Stop recording when not recording (should not error)
	err := interceptor.StopRecording()
	if err != nil {
		t.Fatalf("StopRecording() when not recording failed: %v", err)
	}

	if interceptor.IsRecording() {
		t.Error("Should not be recording")
	}
}

func TestInterceptor_ProcessCommandFromArgs(t *testing.T) {
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	interceptor := NewInterceptor(storage, cfg)

	// Use current directory for testing to avoid path issues
	currentDir := getCurrentDir(t)

	// Test processing command from arguments
	args := []string{
		"command=echo hello world",
		"directory=" + currentDir,
		"shell=bash",
		"exit_code=0",
		"duration=100",
	}

	err := interceptor.ProcessCommandFromArgs(args)
	if err != nil {
		t.Fatalf("ProcessCommandFromArgs() failed: %v", err)
	}

	// Verify command was stored
	commands, err := storage.GetCommandsByDirectory(currentDir)
	if err != nil {
		t.Fatalf("Failed to get commands: %v", err)
	}

	if len(commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(commands))
	}

	cmd := commands[0]
	if cmd.Command != "echo hello world" {
		t.Errorf("Expected command 'echo hello world', got '%s'", cmd.Command)
	}
	if cmd.Directory != currentDir {
		t.Errorf("Expected directory '%s', got '%s'", currentDir, cmd.Directory)
	}
	if cmd.Shell != history.Bash {
		t.Errorf("Expected shell Bash, got %v", cmd.Shell)
	}
	if cmd.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", cmd.ExitCode)
	}
}

func TestInterceptor_ProcessCommandInteractive(t *testing.T) {
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	interceptor := NewInterceptor(storage, cfg)

	// Test processing command interactively
	command := "echo hello"
	err := interceptor.ProcessCommandInteractive(command)
	if err != nil {
		t.Fatalf("ProcessCommandInteractive() failed: %v", err)
	}

	// Get current directory for verification
	currentDir := getCurrentDir(t)

	// Verify command was stored
	commands, err := storage.GetCommandsByDirectory(currentDir)
	if err != nil {
		t.Fatalf("Failed to get commands: %v", err)
	}

	if len(commands) != 1 {
		// Debug: check all directories
		allDirs, _ := storage.GetDirectoriesWithHistory()
		t.Logf("All directories with history: %v", allDirs)
		t.Logf("Looking for commands in: %s", currentDir)
		t.Fatalf("Expected 1 command, got %d", len(commands))
	}

	cmd := commands[0]
	if cmd.Command != "echo hello" {
		t.Errorf("Expected command 'echo hello', got '%s'", cmd.Command)
	}
	if cmd.Directory != currentDir {
		t.Errorf("Expected directory '%s', got '%s'", currentDir, cmd.Directory)
	}
}

func TestInterceptor_ValidateCommand(t *testing.T) {
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	interceptor := NewInterceptor(storage, cfg)

	// Test valid command
	err := interceptor.ValidateCommand("echo hello")
	if err != nil {
		t.Errorf("ValidateCommand() failed for valid command: %v", err)
	}

	// Test empty command
	err = interceptor.ValidateCommand("")
	if err == nil {
		t.Error("ValidateCommand() should fail for empty command")
	}

	// Test excluded command (based on config exclude patterns)
	err = interceptor.ValidateCommand("cd")
	if err == nil {
		t.Error("ValidateCommand() should fail for excluded command 'cd'")
	}
}

func TestInterceptor_GetRecentCommands(t *testing.T) {
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	interceptor := NewInterceptor(storage, cfg)

	// Add some test commands
	commands := []string{"echo test1", "echo test2", "echo test3"}
	for _, cmd := range commands {
		err := interceptor.ProcessCommandInteractive(cmd)
		if err != nil {
			t.Fatalf("Failed to process command '%s': %v", cmd, err)
		}
	}

	// Get recent commands
	recentCommands, err := interceptor.GetRecentCommands(2)
	if err != nil {
		t.Fatalf("GetRecentCommands() failed: %v", err)
	}

	if len(recentCommands) > 2 {
		t.Errorf("Expected at most 2 recent commands, got %d", len(recentCommands))
	}

	// Verify we have at least some commands
	if len(recentCommands) == 0 {
		// Check if commands were stored at all
		currentDir := getCurrentDir(t)
		allCommands, err := storage.GetCommandsByDirectory(currentDir)
		if err != nil {
			t.Fatalf("Failed to get all commands: %v", err)
		}
		t.Logf("Total commands in current directory: %d", len(allCommands))
		if len(allCommands) > 0 {
			t.Logf("First command: %+v", allCommands[0])
		}
		t.Error("Expected at least some recent commands")
	}
}

func TestInterceptor_GetStatus(t *testing.T) {
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	interceptor := NewInterceptor(storage, cfg)

	// Get status
	status, err := interceptor.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() failed: %v", err)
	}

	if status == nil {
		t.Fatal("GetStatus() returned nil status")
	}

	// Verify status fields
	if status.CurrentShell <= history.Unknown || status.CurrentShell > history.Cmd {
		t.Errorf("Invalid current shell in status: %v", status.CurrentShell)
	}
}

// getCurrentDir gets the current directory for testing
func getCurrentDir(t *testing.T) string {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	absDir, err := filepath.Abs(currentDir)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	return filepath.ToSlash(absDir)
}

// createTestStorage creates a test storage instance and configuration
func createTestStorage(t *testing.T) (history.StorageEngine, *config.Config) {
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create storage
	storage := storage.NewSQLiteStorage(dbPath)
	if err := storage.Initialize(); err != nil {
		t.Fatalf("Failed to initialize test storage: %v", err)
	}

	// Create test configuration
	cfg := &config.Config{
		StoragePath:     dbPath,
		RetentionDays:   30,
		MaxCommands:     1000,
		EnabledShells:   []history.ShellType{history.PowerShell, history.Bash, history.Zsh, history.Cmd},
		ExcludePatterns: []string{"cd", "ls", "pwd"},
		AutoCleanup:     false,
	}

	return storage, cfg
}
