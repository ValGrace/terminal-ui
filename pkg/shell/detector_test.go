package shell

import (
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"os"
	"runtime"
	"testing"
)

func TestDetector_DetectShell(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name     string
		envVars  map[string]string
		expected history.ShellType
		wantErr  bool
		skipOS   string // Skip test on specific OS
	}{
		{
			name: "detect bash from SHELL env var",
			envVars: map[string]string{
				"SHELL": "/bin/bash",
			},
			expected: history.Bash,
			wantErr:  false,
			skipOS:   "windows", // Skip on Windows as it has different detection logic
		},
		{
			name: "detect zsh from SHELL env var",
			envVars: map[string]string{
				"SHELL": "/usr/local/bin/zsh",
			},
			expected: history.Zsh,
			wantErr:  false,
			skipOS:   "windows", // Skip on Windows as it has different detection logic
		},
		{
			name: "detect powershell from PSModulePath",
			envVars: map[string]string{
				"PSModulePath": "C:\\Program Files\\PowerShell\\Modules",
			},
			expected: history.PowerShell,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip test if running on specified OS
			if tt.skipOS != "" && runtime.GOOS == tt.skipOS {
				t.Skipf("Skipping test on %s", tt.skipOS)
			}

			// Save original environment
			originalEnv := make(map[string]string)

			// Save all environment variables that might affect detection
			allEnvVars := []string{"SHELL", "PSModulePath", "PSVersionTable", "COMSPEC",
				"BASH_VERSION", "ZSH_VERSION", "BASH", "ZSH_NAME"}

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

			got, err := detector.DetectShell()
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectShell() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("DetectShell() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDetector_IsShellSupported(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name     string
		shell    history.ShellType
		expected bool
	}{
		{
			name:     "PowerShell is supported on all platforms",
			shell:    history.PowerShell,
			expected: true,
		},
		{
			name:     "Bash is supported on all platforms",
			shell:    history.Bash,
			expected: true,
		},
		{
			name:     "Zsh support depends on platform",
			shell:    history.Zsh,
			expected: runtime.GOOS != "windows",
		},
		{
			name:     "Cmd is only supported on Windows",
			shell:    history.Cmd,
			expected: runtime.GOOS == "windows",
		},
		{
			name:     "Unknown shell is not supported",
			shell:    history.Unknown,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.IsShellSupported(tt.shell)
			if got != tt.expected {
				t.Errorf("IsShellSupported(%v) = %v, want %v", tt.shell, got, tt.expected)
			}
		})
	}
}

func TestDetector_GetShellPath(t *testing.T) {
	detector := NewDetector()

	// Test that we can get paths for supported shells
	supportedShells := []history.ShellType{
		history.PowerShell,
		history.Bash,
	}

	// Add platform-specific shells
	if runtime.GOOS != "windows" {
		supportedShells = append(supportedShells, history.Zsh)
	}
	if runtime.GOOS == "windows" {
		supportedShells = append(supportedShells, history.Cmd)
	}

	for _, shell := range supportedShells {
		t.Run("get path for "+shell.String(), func(t *testing.T) {
			if !detector.IsShellSupported(shell) {
				t.Skip("Shell not supported on this platform")
			}

			path, err := detector.GetShellPath(shell)
			if err != nil {
				// It's okay if the shell is not installed, just skip
				t.Skipf("Shell %s not found: %v", shell.String(), err)
			}

			if path == "" {
				t.Errorf("GetShellPath(%v) returned empty path", shell)
			}
		})
	}
}

func TestDetector_GetShellPath_UnsupportedShell(t *testing.T) {
	detector := NewDetector()

	// Test with an invalid shell type
	_, err := detector.GetShellPath(history.ShellType(999))
	if err == nil {
		t.Error("GetShellPath() should return error for unsupported shell")
	}
}
