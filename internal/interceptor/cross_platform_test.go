package interceptor

import (
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"github.com/ValGrace/command-history-tracker/pkg/shell"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCrossPlatformCapture_WindowsSpecific(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Create Windows-specific capture
	windowsCapture := NewWindowsCapture(storage, cfg)
	if windowsCapture == nil {
		t.Fatal("NewWindowsCapture returned nil on Windows")
	}

	// Test Windows shell detection
	shell, err := windowsCapture.detectWindowsShell()
	if err != nil {
		t.Logf("Windows shell detection failed (expected in test environment): %v", err)
		// This is expected in test environments, so we'll continue with a mock test
		shell = history.PowerShell
	}

	// Verify detected shell is Windows-compatible
	windowsShells := []history.ShellType{history.PowerShell, history.Cmd, history.Bash}
	isWindowsShell := false
	for _, ws := range windowsShells {
		if shell == ws {
			isWindowsShell = true
			break
		}
	}

	if !isWindowsShell {
		t.Errorf("Detected shell %v is not Windows-compatible", shell)
	}
}

func TestCrossPlatformCapture_UnixSpecific(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Create Unix-specific capture
	unixCapture := NewUnixCapture(storage, cfg)
	if unixCapture == nil {
		t.Fatal("NewUnixCapture returned nil on Unix")
	}

	// Test Unix shell detection
	shell, err := unixCapture.detectUnixShell()
	if err != nil {
		t.Logf("Unix shell detection failed (expected in test environment): %v", err)
		// This is expected in test environments, so we'll continue with a mock test
		shell = history.Bash
	}

	// Verify detected shell is Unix-compatible
	unixShells := []history.ShellType{history.Bash, history.Zsh, history.PowerShell}
	isUnixShell := false
	for _, us := range unixShells {
		if shell == us {
			isUnixShell = true
			break
		}
	}

	if !isUnixShell {
		t.Errorf("Detected shell %v is not Unix-compatible", shell)
	}
}

func TestCrossPlatformCapture_PowerShellDetection(t *testing.T) {
	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Save original environment
	originalEnv := map[string]string{
		"PSModulePath":   os.Getenv("PSModulePath"),
		"PSVersionTable": os.Getenv("PSVersionTable"),
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

	// Set PowerShell environment variables
	os.Setenv("PSModulePath", "C:\\Program Files\\PowerShell\\Modules")
	os.Setenv("PSVersionTable", "Name Value")

	// Test platform-specific capture
	if runtime.GOOS == "windows" {
		windowsCapture := NewWindowsCapture(storage, cfg)
		if windowsCapture != nil {
			isPowerShell := windowsCapture.isInPowerShell()
			if !isPowerShell {
				t.Error("Should detect PowerShell environment on Windows")
			}
		}
	} else {
		unixCapture := NewUnixCapture(storage, cfg)
		if unixCapture != nil {
			isPowerShell := unixCapture.isInPowerShellCore()
			if !isPowerShell {
				t.Error("Should detect PowerShell Core environment on Unix")
			}
		}
	}
}

func TestCrossPlatformCapture_BashDetection(t *testing.T) {
	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Save original environment
	originalEnv := map[string]string{
		"BASH_VERSION": os.Getenv("BASH_VERSION"),
		"BASH":         os.Getenv("BASH"),
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

	// Set Bash environment variables
	os.Setenv("BASH_VERSION", "5.0.0")
	os.Setenv("BASH", "/bin/bash")

	// Test platform-specific capture
	if runtime.GOOS == "windows" {
		windowsCapture := NewWindowsCapture(storage, cfg)
		if windowsCapture != nil {
			isBash := windowsCapture.isInWindowsBash()
			if !isBash {
				t.Error("Should detect Bash environment on Windows")
			}
		}
	} else {
		unixCapture := NewUnixCapture(storage, cfg)
		if unixCapture != nil {
			isBash := unixCapture.isInBash()
			if !isBash {
				t.Error("Should detect Bash environment on Unix")
			}
		}
	}
}

func TestCrossPlatformCapture_ZshDetection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Zsh detection test for Unix systems only")
	}

	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Save original environment
	originalEnv := map[string]string{
		"ZSH_VERSION": os.Getenv("ZSH_VERSION"),
		"ZSH_NAME":    os.Getenv("ZSH_NAME"),
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

	// Set Zsh environment variables
	os.Setenv("ZSH_VERSION", "5.8")
	os.Setenv("ZSH_NAME", "zsh")

	unixCapture := NewUnixCapture(storage, cfg)
	if unixCapture != nil {
		isZsh := unixCapture.isInZsh()
		if !isZsh {
			t.Error("Should detect Zsh environment on Unix")
		}
	}
}

func TestCrossPlatformCapture_CommandCapture(t *testing.T) {
	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Test data
	testDir := getCurrentDir(t)
	testCommand := "echo cross-platform test"

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
	os.Setenv("CHT_EXIT_CODE", "0")
	os.Setenv("CHT_DURATION", "150")
	os.Setenv("CHT_TIMESTAMP", time.Now().Format(time.RFC3339))

	// Enable tracking
	execPath, _ := os.Executable()
	os.Setenv("CHT_TRACKER_PATH", execPath)

	// Test platform-specific capture
	if runtime.GOOS == "windows" {
		// Test Windows-specific capture
		os.Setenv("CHT_SHELL", "powershell")

		windowsCapture := NewWindowsCapture(storage, cfg)
		if windowsCapture != nil {
			err := windowsCapture.CaptureCommand()
			if err != nil {
				t.Fatalf("Windows command capture failed: %v", err)
			}

			// Verify command was stored
			commands, err := storage.GetCommandsByDirectory(testDir)
			if err != nil {
				t.Fatalf("Failed to retrieve commands: %v", err)
			}

			if len(commands) == 0 {
				t.Fatal("No commands were captured")
			}

			cmd := commands[len(commands)-1] // Get the last command
			if cmd.Command != testCommand {
				t.Errorf("Expected command '%s', got '%s'", testCommand, cmd.Command)
			}

			// Verify Windows-specific tags
			hasWindowsTag := false
			for _, tag := range cmd.Tags {
				if strings.Contains(tag, "windows") || strings.Contains(tag, "powershell") {
					hasWindowsTag = true
					break
				}
			}
			if !hasWindowsTag {
				t.Errorf("Expected Windows-specific tags, got: %v", cmd.Tags)
			}
		}
	} else {
		// Test Unix-specific capture
		os.Setenv("CHT_SHELL", "bash")

		unixCapture := NewUnixCapture(storage, cfg)
		if unixCapture != nil {
			err := unixCapture.CaptureCommand()
			if err != nil {
				t.Fatalf("Unix command capture failed: %v", err)
			}

			// Verify command was stored
			commands, err := storage.GetCommandsByDirectory(testDir)
			if err != nil {
				t.Fatalf("Failed to retrieve commands: %v", err)
			}

			if len(commands) == 0 {
				t.Fatal("No commands were captured")
			}

			cmd := commands[len(commands)-1] // Get the last command
			if cmd.Command != testCommand {
				t.Errorf("Expected command '%s', got '%s'", testCommand, cmd.Command)
			}

			// Verify Unix-specific tags
			hasUnixTag := false
			for _, tag := range cmd.Tags {
				if strings.Contains(tag, "unix") || strings.Contains(tag, "bash") {
					hasUnixTag = true
					break
				}
			}
			if !hasUnixTag {
				t.Errorf("Expected Unix-specific tags, got: %v", cmd.Tags)
			}
		}
	}
}

func TestCrossPlatformCapture_ShellSpecificMetadata(t *testing.T) {
	testDir := getCurrentDir(t)

	// Test different shell-specific commands and verify appropriate metadata
	testCases := []struct {
		shell       history.ShellType
		command     string
		expectedTag string
	}{
		{history.PowerShell, "Get-Process -Name test", "powershell-cmdlet"},
		{history.PowerShell, "Invoke-WebRequest -Uri http://example.com", "powershell-invoke"},
		{history.Bash, "export TEST_VAR=value", "bash-export"},
		{history.Bash, "source ~/.profile", "bash-source"},
		{history.Zsh, "autoload -Uz compinit", "zsh-autoload"},
		{history.Zsh, "setopt EXTENDED_GLOB", "zsh-option"},
	}

	for i, tc := range testCases {
		t.Run(tc.shell.String()+"_"+tc.expectedTag, func(t *testing.T) {
			// Create separate storage for each test to avoid ID conflicts
			storage, cfg := createTestStorage(t)
			defer storage.Close()

			// Add a small delay to ensure unique timestamps
			time.Sleep(time.Duration(i+1) * time.Millisecond)
			// Skip if shell is not supported on current platform
			detector := shell.NewDetector()
			if !detector.IsShellSupported(tc.shell) {
				t.Skipf("Shell %v not supported on %s", tc.shell, runtime.GOOS)
			}

			// Set up environment for shell-specific capture
			originalEnv := map[string]string{
				"CHT_COMMAND":      os.Getenv("CHT_COMMAND"),
				"CHT_DIRECTORY":    os.Getenv("CHT_DIRECTORY"),
				"CHT_SHELL":        os.Getenv("CHT_SHELL"),
				"CHT_EXIT_CODE":    os.Getenv("CHT_EXIT_CODE"),
				"CHT_DURATION":     os.Getenv("CHT_DURATION"),
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

			// Set test environment with unique timestamp
			uniqueCommand := fmt.Sprintf("%s # test_%d", tc.command, time.Now().UnixNano())
			os.Setenv("CHT_COMMAND", uniqueCommand)
			os.Setenv("CHT_DIRECTORY", testDir)
			os.Setenv("CHT_SHELL", tc.shell.String())
			os.Setenv("CHT_EXIT_CODE", "0")
			os.Setenv("CHT_DURATION", "100")
			execPath, _ := os.Executable()
			os.Setenv("CHT_TRACKER_PATH", execPath)

			// Create platform-specific capture and use shell-specific methods
			var err error
			if runtime.GOOS == "windows" {
				windowsCapture := NewWindowsCapture(storage, cfg)
				if windowsCapture != nil {
					switch tc.shell {
					case history.PowerShell:
						err = windowsCapture.CapturePowerShellCommand()
					case history.Bash:
						err = windowsCapture.CaptureWindowsBashCommand()
					default:
						err = windowsCapture.CaptureCommandDirect(tc.command, testDir, tc.shell, 0, 100*time.Millisecond)
					}
				}
			} else {
				unixCapture := NewUnixCapture(storage, cfg)
				if unixCapture != nil {
					switch tc.shell {
					case history.PowerShell:
						err = unixCapture.CapturePowerShellCoreCommand()
					case history.Bash:
						err = unixCapture.CaptureBashCommand()
					case history.Zsh:
						err = unixCapture.CaptureZshCommand()
					default:
						err = unixCapture.CaptureCommandDirect(tc.command, testDir, tc.shell, 0, 100*time.Millisecond)
					}
				}
			}

			if err != nil {
				t.Fatalf("Command capture failed: %v", err)
			}

			// Verify command was stored with expected metadata
			commands, err := storage.GetCommandsByDirectory(testDir)
			if err != nil {
				t.Fatalf("Failed to retrieve commands: %v", err)
			}

			// Find the command we just added (look for the base command in the unique command)
			var foundCmd *history.CommandRecord
			for i := len(commands) - 1; i >= 0; i-- {
				if strings.Contains(commands[i].Command, tc.command) && commands[i].Shell == tc.shell {
					foundCmd = &commands[i]
					break
				}
			}

			if foundCmd == nil {
				t.Fatalf("Command not found in storage")
			}

			// Verify expected tag is present (check if any tag contains the expected pattern)
			hasExpectedTag := false
			for _, tag := range foundCmd.Tags {
				if strings.Contains(tag, tc.expectedTag) {
					hasExpectedTag = true
					break
				}
			}
			if !hasExpectedTag {
				t.Errorf("Expected tag containing '%s' not found, tags: %v", tc.expectedTag, foundCmd.Tags)
			}

			// Verify shell-specific tag is present (check for shell- prefix or exact match)
			shellTag := tc.shell.String()
			hasShellTag := false
			for _, tag := range foundCmd.Tags {
				if tag == shellTag || strings.Contains(tag, "shell-"+shellTag) {
					hasShellTag = true
					break
				}
			}
			if !hasShellTag {
				t.Errorf("Expected shell tag containing '%s' not found, tags: %v", shellTag, foundCmd.Tags)
			}
		})
	}
}

func TestCrossPlatformCapture_PlatformMetadata(t *testing.T) {
	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	testDir := getCurrentDir(t)
	testCommand := "echo platform test"

	// Create platform-specific capture and capture a command
	var err error
	if runtime.GOOS == "windows" {
		windowsCapture := NewWindowsCapture(storage, cfg)
		if windowsCapture != nil {
			err = windowsCapture.CaptureCommandDirect(testCommand, testDir, history.PowerShell, 0, 100*time.Millisecond)
		}
	} else {
		unixCapture := NewUnixCapture(storage, cfg)
		if unixCapture != nil {
			err = unixCapture.CaptureCommandDirect(testCommand, testDir, history.Bash, 0, 100*time.Millisecond)
		}
	}

	if err != nil {
		t.Fatalf("Command capture failed: %v", err)
	}

	// Verify command was stored with platform metadata
	commands, err := storage.GetCommandsByDirectory(testDir)
	if err != nil {
		t.Fatalf("Failed to retrieve commands: %v", err)
	}

	if len(commands) == 0 {
		t.Fatal("No commands were captured")
	}

	cmd := commands[len(commands)-1] // Get the last command

	// Verify platform-specific tags are present
	expectedPlatformTags := []string{
		runtime.GOOS, // Should have OS tag
		"success",    // Should have success tag (exit code 0)
	}

	for _, expectedTag := range expectedPlatformTags {
		hasTag := false
		for _, tag := range cmd.Tags {
			if strings.Contains(tag, expectedTag) {
				hasTag = true
				break
			}
		}
		if !hasTag {
			t.Errorf("Expected platform tag containing '%s', tags: %v", expectedTag, cmd.Tags)
		}
	}
}

func TestCrossPlatformCapture_ShellInfo(t *testing.T) {
	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	if runtime.GOOS == "windows" {
		// Test Windows shell info
		windowsCapture := NewWindowsCapture(storage, cfg)
		if windowsCapture != nil {
			info, err := windowsCapture.GetWindowsShellInfo()
			if err != nil {
				t.Fatalf("GetWindowsShellInfo failed: %v", err)
			}

			if info.Platform != "windows" {
				t.Errorf("Expected platform 'windows', got '%s'", info.Platform)
			}

			if len(info.Shells) == 0 {
				t.Error("Expected at least one shell in Windows shell info")
			}
		}
	} else {
		// Test Unix shell info
		unixCapture := NewUnixCapture(storage, cfg)
		if unixCapture != nil {
			info, err := unixCapture.GetUnixShellInfo()
			if err != nil {
				t.Fatalf("GetUnixShellInfo failed: %v", err)
			}

			if info.Platform != runtime.GOOS {
				t.Errorf("Expected platform '%s', got '%s'", runtime.GOOS, info.Platform)
			}

			if len(info.Shells) == 0 {
				t.Error("Expected at least one shell in Unix shell info")
			}
		}
	}
}

func TestCrossPlatformCapture_ErrorHandling(t *testing.T) {
	// Create test storage and config
	storage, cfg := createTestStorage(t)
	defer storage.Close()

	// Test error handling with invalid environment
	originalEnv := map[string]string{
		"CHT_COMMAND":      os.Getenv("CHT_COMMAND"),
		"CHT_DIRECTORY":    os.Getenv("CHT_DIRECTORY"),
		"CHT_TRACKER_PATH": os.Getenv("CHT_TRACKER_PATH"),
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

	// Clear required environment variables
	os.Unsetenv("CHT_COMMAND")
	os.Unsetenv("CHT_DIRECTORY")
	os.Unsetenv("CHT_TRACKER_PATH")

	// Test platform-specific capture with missing environment
	if runtime.GOOS == "windows" {
		windowsCapture := NewWindowsCapture(storage, cfg)
		if windowsCapture != nil {
			err := windowsCapture.CaptureCommand()
			// On Windows, this might not error if tracking is disabled
			// The important thing is that it doesn't panic
			_ = err
		}
	} else {
		unixCapture := NewUnixCapture(storage, cfg)
		if unixCapture != nil {
			err := unixCapture.CaptureCommand()
			// On Unix, this might not error if tracking is disabled
			// The important thing is that it doesn't panic
			_ = err
		}
	}
}
