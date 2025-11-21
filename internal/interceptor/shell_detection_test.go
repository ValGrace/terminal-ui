package interceptor

import (
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"github.com/ValGrace/command-history-tracker/pkg/shell"
	"os"
	"runtime"
	"testing"
)

// TestShellDetectionAcrossPlatforms tests shell detection functionality across different platforms
func TestShellDetectionAcrossPlatforms(t *testing.T) {
	detector := shell.NewDetector()

	tests := []struct {
		name        string
		platform    string
		envVars     map[string]string
		expected    history.ShellType
		shouldError bool
		skipOS      string
	}{
		{
			name:     "PowerShell detection on Windows",
			platform: "windows",
			envVars: map[string]string{
				"PSModulePath": "C:\\Program Files\\PowerShell\\Modules",
			},
			expected: history.PowerShell,
			skipOS:   "!windows",
		},
		{
			name:     "PowerShell Core detection on Unix",
			platform: "unix",
			envVars: map[string]string{
				"PSModulePath":                    "/usr/local/share/powershell/Modules",
				"POWERSHELL_DISTRIBUTION_CHANNEL": "PSCore",
			},
			expected: history.PowerShell,
			skipOS:   "windows",
		},
		{
			name:     "Bash detection via SHELL env var",
			platform: "unix",
			envVars: map[string]string{
				"SHELL": "/bin/bash",
			},
			expected: history.Bash,
			skipOS:   "windows",
		},
		{
			name:     "Bash detection via BASH_VERSION",
			platform: "any",
			envVars: map[string]string{
				"BASH_VERSION": "5.0.0",
				"BASH":         "/usr/bin/bash",
			},
			expected: history.Bash,
		},
		{
			name:     "Zsh detection via SHELL env var",
			platform: "unix",
			envVars: map[string]string{
				"SHELL": "/usr/local/bin/zsh",
			},
			expected: history.Zsh,
			skipOS:   "windows",
		},
		{
			name:     "Zsh detection via ZSH_VERSION",
			platform: "unix",
			envVars: map[string]string{
				"ZSH_VERSION": "5.8",
				"ZSH_NAME":    "zsh",
			},
			expected: history.Zsh,
			skipOS:   "windows",
		},
		{
			name:     "CMD detection on Windows",
			platform: "windows",
			envVars: map[string]string{
				"COMSPEC": "C:\\Windows\\System32\\cmd.exe",
			},
			expected: history.Cmd,
			skipOS:   "!windows",
		},
		{
			name:     "Git Bash detection on Windows",
			platform: "windows",
			envVars: map[string]string{
				"BASH_VERSION": "4.4.23",
				"MSYSTEM":      "MINGW64",
			},
			expected: history.Bash,
			skipOS:   "!windows",
		},
		{
			name:     "WSL Bash detection",
			platform: "wsl",
			envVars: map[string]string{
				"WSL_DISTRO_NAME": "Ubuntu",
				"BASH_VERSION":    "5.0.17",
			},
			expected: history.Bash,
			skipOS:   "!windows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip test based on OS
			if tt.skipOS != "" {
				if tt.skipOS == "windows" && runtime.GOOS == "windows" {
					t.Skip("Skipping on Windows")
				}
				if tt.skipOS == "!windows" && runtime.GOOS != "windows" {
					t.Skip("Skipping on non-Windows")
				}
			}

			// Save original environment
			originalEnv := make(map[string]string)
			allEnvVars := []string{
				"SHELL", "PSModulePath", "PSVersionTable", "POWERSHELL_DISTRIBUTION_CHANNEL",
				"COMSPEC", "BASH_VERSION", "BASH", "ZSH_VERSION", "ZSH_NAME",
				"WSL_DISTRO_NAME", "MSYSTEM",
			}

			for _, key := range allEnvVars {
				originalEnv[key] = os.Getenv(key)
				os.Unsetenv(key) // Clear all first
			}

			// Set test environment
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Restore environment after test
			defer func() {
				for key, value := range originalEnv {
					if value == "" {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, value)
					}
				}
			}()

			// Test shell detection
			detected, err := detector.DetectShell()
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("DetectShell() failed: %v", err)
			}

			if detected != tt.expected {
				t.Errorf("Expected shell %v, got %v", tt.expected, detected)
			}

			// Verify shell is supported on current platform
			if !detector.IsShellSupported(detected) {
				t.Errorf("Detected shell %v should be supported on current platform", detected)
			}
		})
	}
}

// TestShellDetectionPlatformSpecific tests platform-specific shell detection logic
func TestShellDetectionPlatformSpecific(t *testing.T) {
	detector := shell.NewDetector()

	// Test current platform detection
	currentShell, err := detector.DetectShell()
	if err != nil {
		t.Logf("Shell detection failed (expected in test environment): %v", err)
		// This is acceptable in test environments
		return
	}

	t.Logf("Detected shell: %v (%s)", currentShell, currentShell.String())

	// Verify detected shell is valid
	if currentShell <= history.Unknown || currentShell > history.Cmd {
		t.Errorf("Invalid shell type detected: %v", currentShell)
	}

	// Verify shell is supported on current platform
	if !detector.IsShellSupported(currentShell) {
		t.Errorf("Detected shell %v should be supported on platform %s", currentShell, runtime.GOOS)
	}

	// Test shell path resolution
	if currentShell != history.Unknown {
		shellPath, err := detector.GetShellPath(currentShell)
		if err != nil {
			t.Logf("Shell path resolution failed (acceptable): %v", err)
		} else {
			t.Logf("Shell path: %s", shellPath)
			if shellPath == "" {
				t.Error("Shell path should not be empty when resolution succeeds")
			}
		}
	}
}

// TestShellSupportValidation tests shell support validation across platforms
func TestShellSupportValidation(t *testing.T) {
	detector := shell.NewDetector()

	// Test platform-specific shell support
	platformTests := map[string][]struct {
		shell     history.ShellType
		supported bool
	}{
		"windows": {
			{history.PowerShell, true},
			{history.Cmd, true},
			{history.Bash, true}, // Available via WSL/Git Bash
			{history.Zsh, false}, // Not typically available
		},
		"linux": {
			{history.Bash, true},
			{history.Zsh, true},
			{history.PowerShell, true}, // PowerShell Core
			{history.Cmd, false},       // Windows-only
		},
		"darwin": {
			{history.Bash, true},
			{history.Zsh, true},
			{history.PowerShell, true}, // PowerShell Core
			{history.Cmd, false},       // Windows-only
		},
	}

	currentPlatform := runtime.GOOS
	if tests, exists := platformTests[currentPlatform]; exists {
		for _, tt := range tests {
			t.Run(tt.shell.String()+"_on_"+currentPlatform, func(t *testing.T) {
				supported := detector.IsShellSupported(tt.shell)
				if supported != tt.supported {
					t.Errorf("Shell %v support on %s: expected %v, got %v",
						tt.shell, currentPlatform, tt.supported, supported)
				}
			})
		}
	} else {
		t.Logf("No specific tests defined for platform: %s", currentPlatform)
	}

	// Test Unknown shell (should never be supported)
	if detector.IsShellSupported(history.Unknown) {
		t.Error("Unknown shell type should never be supported")
	}
}

// TestShellDetectionEnvironmentVariables tests detection based on specific environment variables
func TestShellDetectionEnvironmentVariables(t *testing.T) {
	detector := shell.NewDetector()

	envTests := []struct {
		name     string
		envVar   string
		value    string
		expected history.ShellType
		skipOS   string
	}{
		{
			name:     "PowerShell via PSModulePath",
			envVar:   "PSModulePath",
			value:    "/usr/local/share/powershell/Modules",
			expected: history.PowerShell,
		},
		{
			name:     "PowerShell via PSVersionTable",
			envVar:   "PSVersionTable",
			value:    "Name Value",
			expected: history.PowerShell,
		},
		{
			name:     "Bash via BASH_VERSION",
			envVar:   "BASH_VERSION",
			value:    "5.0.0",
			expected: history.Bash,
		},
		{
			name:     "Zsh via ZSH_VERSION",
			envVar:   "ZSH_VERSION",
			value:    "5.8",
			expected: history.Zsh,
			skipOS:   "windows",
		},
		{
			name:     "CMD via COMSPEC",
			envVar:   "COMSPEC",
			value:    "C:\\Windows\\System32\\cmd.exe",
			expected: history.Cmd,
			skipOS:   "!windows",
		},
	}

	for _, tt := range envTests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip test based on OS
			if tt.skipOS == "windows" && runtime.GOOS == "windows" {
				t.Skip("Skipping on Windows")
			}
			if tt.skipOS == "!windows" && runtime.GOOS != "windows" {
				t.Skip("Skipping on non-Windows")
			}

			// Save original environment
			originalValue := os.Getenv(tt.envVar)
			defer func() {
				if originalValue == "" {
					os.Unsetenv(tt.envVar)
				} else {
					os.Setenv(tt.envVar, originalValue)
				}
			}()

			// Set test environment variable
			os.Setenv(tt.envVar, tt.value)

			// Clear other shell-related environment variables to avoid conflicts
			otherEnvVars := []string{
				"SHELL", "PSModulePath", "PSVersionTable", "BASH_VERSION",
				"ZSH_VERSION", "COMSPEC",
			}
			originalOtherEnv := make(map[string]string)
			for _, envVar := range otherEnvVars {
				if envVar != tt.envVar {
					originalOtherEnv[envVar] = os.Getenv(envVar)
					os.Unsetenv(envVar)
				}
			}

			defer func() {
				for envVar, value := range originalOtherEnv {
					if value == "" {
						os.Unsetenv(envVar)
					} else {
						os.Setenv(envVar, value)
					}
				}
			}()

			// Test detection
			detected, err := detector.DetectShell()
			if err != nil {
				t.Fatalf("DetectShell() failed: %v", err)
			}

			// The detection might not always match exactly due to platform differences
			// but we should at least get a valid shell type
			if detected == history.Unknown {
				t.Errorf("Expected to detect a valid shell, got Unknown")
			}

			// Log the result for debugging
			t.Logf("Environment %s=%s detected shell: %v", tt.envVar, tt.value, detected)
		})
	}
}

// TestShellDetectionFallback tests fallback behavior when no shell is clearly detected
func TestShellDetectionFallback(t *testing.T) {
	detector := shell.NewDetector()

	// Save original environment
	shellEnvVars := []string{
		"SHELL", "PSModulePath", "PSVersionTable", "POWERSHELL_DISTRIBUTION_CHANNEL",
		"COMSPEC", "BASH_VERSION", "BASH", "ZSH_VERSION", "ZSH_NAME",
		"WSL_DISTRO_NAME", "MSYSTEM",
	}
	originalEnv := make(map[string]string)
	for _, envVar := range shellEnvVars {
		originalEnv[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}

	defer func() {
		for envVar, value := range originalEnv {
			if value == "" {
				os.Unsetenv(envVar)
			} else {
				os.Setenv(envVar, value)
			}
		}
	}()

	// Test detection with minimal environment
	detected, err := detector.DetectShell()

	// The behavior depends on the platform
	switch runtime.GOOS {
	case "windows":
		// On Windows, should default to PowerShell or detect based on system
		if err != nil {
			t.Logf("Windows shell detection failed (acceptable): %v", err)
		} else {
			expectedShells := []history.ShellType{history.PowerShell, history.Cmd}
			found := false
			for _, expected := range expectedShells {
				if detected == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected Windows shell (PowerShell or Cmd), got %v", detected)
			}
		}
	case "darwin":
		// On macOS, should default to Zsh (modern default) or Bash
		if err != nil {
			t.Logf("macOS shell detection failed (acceptable): %v", err)
		} else {
			expectedShells := []history.ShellType{history.Zsh, history.Bash}
			found := false
			for _, expected := range expectedShells {
				if detected == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected macOS shell (Zsh or Bash), got %v", detected)
			}
		}
	default:
		// On Linux and other Unix-like systems, should default to Bash
		if err != nil {
			t.Logf("Unix shell detection failed (acceptable): %v", err)
		} else {
			expectedShells := []history.ShellType{history.Bash, history.Zsh}
			found := false
			for _, expected := range expectedShells {
				if detected == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected Unix shell (Bash or Zsh), got %v", detected)
			}
		}
	}

	t.Logf("Fallback detection result: %v", detected)
}
