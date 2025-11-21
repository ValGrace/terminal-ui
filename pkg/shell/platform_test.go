package shell

import (
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"runtime"
	"strings"
	"testing"
)

func TestPlatformAbstraction_GetPlatform(t *testing.T) {
	platform := NewPlatformAbstraction()

	currentPlatform := platform.GetPlatform()

	// Verify platform detection matches runtime.GOOS
	switch runtime.GOOS {
	case "windows":
		if currentPlatform != PlatformWindows {
			t.Errorf("Expected PlatformWindows, got %v", currentPlatform)
		}
	case "linux":
		if currentPlatform != PlatformLinux {
			t.Errorf("Expected PlatformLinux, got %v", currentPlatform)
		}
	case "darwin":
		if currentPlatform != PlatformDarwin {
			t.Errorf("Expected PlatformDarwin, got %v", currentPlatform)
		}
	case "freebsd":
		if currentPlatform != PlatformFreeBSD {
			t.Errorf("Expected PlatformFreeBSD, got %v", currentPlatform)
		}
	default:
		if currentPlatform != PlatformUnknown {
			t.Errorf("Expected PlatformUnknown for unsupported OS, got %v", currentPlatform)
		}
	}
}

func TestPlatformAbstraction_GetSupportedShells(t *testing.T) {
	platform := NewPlatformAbstraction()

	supportedShells := platform.GetSupportedShells()

	if len(supportedShells) == 0 {
		t.Error("Expected at least one supported shell")
	}

	// Verify platform-specific shell support
	currentPlatform := platform.GetPlatform()
	switch currentPlatform {
	case PlatformWindows:
		expectedShells := []history.ShellType{history.PowerShell, history.Cmd, history.Bash}
		for _, expected := range expectedShells {
			found := false
			for _, supported := range supportedShells {
				if supported == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected shell %v to be supported on Windows", expected)
			}
		}
	case PlatformLinux, PlatformDarwin, PlatformFreeBSD:
		expectedShells := []history.ShellType{history.Bash, history.Zsh, history.PowerShell}
		for _, expected := range expectedShells {
			found := false
			for _, supported := range supportedShells {
				if supported == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected shell %v to be supported on Unix-like systems", expected)
			}
		}
	}
}

func TestPlatformAbstraction_GetDefaultShell(t *testing.T) {
	platform := NewPlatformAbstraction()

	defaultShell := platform.GetDefaultShell()

	// Verify platform-specific defaults
	currentPlatform := platform.GetPlatform()
	switch currentPlatform {
	case PlatformWindows:
		if defaultShell != history.PowerShell {
			t.Errorf("Expected PowerShell as default on Windows, got %v", defaultShell)
		}
	case PlatformDarwin:
		if defaultShell != history.Zsh {
			t.Errorf("Expected Zsh as default on macOS, got %v", defaultShell)
		}
	case PlatformLinux, PlatformFreeBSD:
		if defaultShell != history.Bash {
			t.Errorf("Expected Bash as default on Linux/FreeBSD, got %v", defaultShell)
		}
	}
}

func TestPlatformAbstraction_GetShellExecutableName(t *testing.T) {
	platform := NewPlatformAbstraction()

	tests := []struct {
		shell    history.ShellType
		expected string
	}{
		{history.Bash, "bash"},
		{history.Zsh, "zsh"},
		{history.PowerShell, "pwsh"},
	}

	// Adjust expectations for Windows
	if platform.GetPlatform() == PlatformWindows {
		tests = []struct {
			shell    history.ShellType
			expected string
		}{
			{history.Bash, "bash.exe"},
			{history.Zsh, "zsh.exe"},
			{history.PowerShell, "pwsh.exe"},
			{history.Cmd, "cmd.exe"},
		}
	}

	for _, tt := range tests {
		t.Run(tt.shell.String(), func(t *testing.T) {
			execName := platform.GetShellExecutableName(tt.shell)
			if execName != tt.expected {
				t.Errorf("Expected executable name %s for %v, got %s",
					tt.expected, tt.shell, execName)
			}
		})
	}
}

func TestPlatformAbstraction_GetShellConfigPath(t *testing.T) {
	platform := NewPlatformAbstraction()

	shells := []history.ShellType{
		history.Bash,
		history.Zsh,
		history.PowerShell,
	}

	// Add Cmd for Windows
	if platform.GetPlatform() == PlatformWindows {
		shells = append(shells, history.Cmd)
	}

	for _, shell := range shells {
		t.Run(shell.String(), func(t *testing.T) {
			configPath, err := platform.GetShellConfigPath(shell)
			if err != nil {
				t.Fatalf("GetShellConfigPath failed for %v: %v", shell, err)
			}

			if configPath == "" {
				t.Errorf("Expected non-empty config path for %v", shell)
			}

			// Verify path contains expected elements
			switch shell {
			case history.Bash:
				if !strings.Contains(configPath, ".bashrc") {
					t.Errorf("Bash config path should contain .bashrc, got: %s", configPath)
				}
			case history.Zsh:
				if !strings.Contains(configPath, ".zshrc") {
					t.Errorf("Zsh config path should contain .zshrc, got: %s", configPath)
				}
			case history.PowerShell:
				if !strings.Contains(configPath, "profile.ps1") {
					t.Errorf("PowerShell config path should contain profile.ps1, got: %s", configPath)
				}
			case history.Cmd:
				if !strings.Contains(configPath, "cht_cmd_init.bat") {
					t.Errorf("Cmd config path should contain cht_cmd_init.bat, got: %s", configPath)
				}
			}
		})
	}
}

func TestPlatformAbstraction_GetEnvironmentVariableSeparator(t *testing.T) {
	platform := NewPlatformAbstraction()

	separator := platform.GetEnvironmentVariableSeparator()

	expectedSeparator := ":"
	if platform.GetPlatform() == PlatformWindows {
		expectedSeparator = ";"
	}

	if separator != expectedSeparator {
		t.Errorf("Expected separator %s, got %s", expectedSeparator, separator)
	}
}

func TestPlatformAbstraction_NormalizePath(t *testing.T) {
	platform := NewPlatformAbstraction()

	tests := []struct {
		name     string
		input    string
		contains string // What the normalized path should contain
	}{
		{
			name:     "relative path",
			input:    "test/path",
			contains: "test/path",
		},
		{
			name:     "path with dots",
			input:    "test/../path/./file",
			contains: "path/file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := platform.NormalizePath(tt.input)

			if normalized == "" {
				t.Error("NormalizePath returned empty string")
			}

			if !strings.Contains(normalized, tt.contains) {
				t.Errorf("Normalized path %s should contain %s", normalized, tt.contains)
			}

			// Verify forward slashes are used for consistency
			if strings.Contains(normalized, "\\") {
				t.Errorf("Normalized path should use forward slashes, got: %s", normalized)
			}
		})
	}
}

func TestPlatformAbstraction_GetHomeDirectory(t *testing.T) {
	platform := NewPlatformAbstraction()

	homeDir, err := platform.GetHomeDirectory()
	if err != nil {
		t.Fatalf("GetHomeDirectory failed: %v", err)
	}

	if homeDir == "" {
		t.Error("Expected non-empty home directory")
	}
}

func TestPlatformAbstraction_IsExecutable(t *testing.T) {
	platform := NewPlatformAbstraction()

	// Test with a known executable (the current test binary)
	// This is a basic test - in practice, you'd test with known executables

	// Test with non-existent file
	if platform.IsExecutable("/non/existent/file") {
		t.Error("IsExecutable should return false for non-existent file")
	}
}

func TestPlatformAbstraction_GetProcessEnvironment(t *testing.T) {
	platform := NewPlatformAbstraction()

	env := platform.GetProcessEnvironment()

	if len(env) == 0 {
		t.Error("Expected non-empty process environment")
	}

	// Verify required environment variables
	if platform := env["CHT_PLATFORM"]; platform == "" {
		t.Error("Expected CHT_PLATFORM in process environment")
	}

	if arch := env["CHT_ARCH"]; arch == "" {
		t.Error("Expected CHT_ARCH in process environment")
	}
}

func TestWindowsPlatform_GetPowerShellVariant(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	platform := NewWindowsPlatform()

	variant := platform.GetPowerShellVariant()

	// Should return either "pwsh", "powershell", or empty string
	if variant != "" && variant != "pwsh" && variant != "powershell" {
		t.Errorf("Unexpected PowerShell variant: %s", variant)
	}
}

func TestWindowsPlatform_GetCmdPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	platform := NewWindowsPlatform()

	cmdPath := platform.GetCmdPath()

	if cmdPath == "" {
		t.Error("Expected non-empty cmd.exe path")
	}

	if !strings.Contains(strings.ToLower(cmdPath), "cmd.exe") {
		t.Errorf("Expected path to contain cmd.exe, got: %s", cmdPath)
	}
}

func TestUnixPlatform_GetShellFromEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	platform := NewUnixPlatform()

	shell := platform.GetShellFromEnvironment()

	// Should return a valid shell type or Unknown
	if shell < history.Unknown || shell > history.Cmd {
		t.Errorf("Invalid shell type returned: %v", shell)
	}
}

func TestUnixPlatform_IsWSL(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	platform := NewUnixPlatform()

	// This test just verifies the method doesn't panic
	// The actual result depends on the environment
	isWSL := platform.IsWSL()

	// Result should be a boolean (no error expected)
	_ = isWSL
}

func TestUnixPlatform_GetDistribution(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific test")
	}

	platform := NewUnixPlatform()

	distribution := platform.GetDistribution()

	if distribution == "" {
		t.Error("Expected non-empty distribution name")
	}
}

func TestPlatformCapture_ValidateShellSupport(t *testing.T) {
	capture := NewPlatformCapture()

	// Test with supported shells
	supportedShells := capture.platform.GetSupportedShells()
	for _, shell := range supportedShells {
		err := capture.ValidateShellSupport(shell)
		if err != nil {
			t.Errorf("ValidateShellSupport failed for supported shell %v: %v", shell, err)
		}
	}

	// Test with Unknown shell (should fail)
	err := capture.ValidateShellSupport(history.Unknown)
	if err == nil {
		t.Error("ValidateShellSupport should fail for Unknown shell")
	}
}

func TestPlatformCapture_GetOptimalShellIntegration(t *testing.T) {
	capture := NewPlatformCapture()

	shell, method, err := capture.GetOptimalShellIntegration()
	if err != nil {
		t.Fatalf("GetOptimalShellIntegration failed: %v", err)
	}

	if shell == history.Unknown {
		t.Error("Expected valid shell type from GetOptimalShellIntegration")
	}

	if method == "" {
		t.Error("Expected non-empty method from GetOptimalShellIntegration")
	}

	// Verify the returned shell is supported
	err = capture.ValidateShellSupport(shell)
	if err != nil {
		t.Errorf("Optimal shell %v should be supported: %v", shell, err)
	}
}

func TestPlatformCapture_GetPlatformMetadata(t *testing.T) {
	capture := NewPlatformCapture()

	metadata := capture.GetPlatformMetadata()

	if len(metadata) == 0 {
		t.Error("Expected non-empty platform metadata")
	}

	// Verify required metadata fields
	if platform := metadata["platform"]; platform == "" {
		t.Error("Expected platform in metadata")
	}

	if arch := metadata["arch"]; arch == "" {
		t.Error("Expected arch in metadata")
	}
}

func TestPlatform_String(t *testing.T) {
	tests := []struct {
		platform Platform
		expected string
	}{
		{PlatformWindows, "windows"},
		{PlatformLinux, "linux"},
		{PlatformDarwin, "darwin"},
		{PlatformFreeBSD, "freebsd"},
		{PlatformUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.platform.String(); got != tt.expected {
				t.Errorf("Platform.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}
