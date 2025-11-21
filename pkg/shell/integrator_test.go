package shell

import (
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestIntegrator_GetIntegrationScript(t *testing.T) {
	integrator := NewIntegrator()

	tests := []struct {
		name     string
		shell    history.ShellType
		wantErr  bool
		contains []string
	}{
		{
			name:    "PowerShell script",
			shell:   history.PowerShell,
			wantErr: false,
			contains: []string{
				"Invoke-HistoryTracker",
				"$env:CHT_COMMAND",
				"tracker record",
			},
		},
		{
			name:    "Bash script",
			shell:   history.Bash,
			wantErr: false,
			contains: []string{
				"__cht_record_command",
				"CHT_COMMAND=",
				"tracker record",
			},
		},
		{
			name:    "Zsh script",
			shell:   history.Zsh,
			wantErr: false,
			contains: []string{
				"__cht_record_command",
				"add-zsh-hook",
				"CHT_COMMAND=",
			},
		},
		{
			name:    "Cmd script",
			shell:   history.Cmd,
			wantErr: false,
			contains: []string{
				"CHT_SHELL=cmd",
				"CHT_DIRECTORY=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script, err := integrator.GetIntegrationScript(tt.shell)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIntegrationScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				for _, expected := range tt.contains {
					if !strings.Contains(script, expected) {
						t.Errorf("GetIntegrationScript() script missing expected content: %s", expected)
					}
				}
			}
		})
	}
}

func TestIntegrator_GetShellConfigPath(t *testing.T) {
	integrator := NewIntegrator()

	tests := []struct {
		name     string
		shell    history.ShellType
		expected string
		wantErr  bool
	}{
		{
			name:    "Bash config path",
			shell:   history.Bash,
			wantErr: false,
		},
		{
			name:    "Zsh config path",
			shell:   history.Zsh,
			wantErr: false,
		},
		{
			name:    "PowerShell config path",
			shell:   history.PowerShell,
			wantErr: false,
		},
		{
			name:    "Cmd config path",
			shell:   history.Cmd,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := integrator.getShellConfigPath(tt.shell)
			if (err != nil) != tt.wantErr {
				t.Errorf("getShellConfigPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if path == "" {
					t.Error("getShellConfigPath() returned empty path")
				}

				// Verify path contains expected elements
				switch tt.shell {
				case history.Bash:
					if !strings.Contains(path, ".bashrc") {
						t.Errorf("Bash config path should contain .bashrc, got: %s", path)
					}
				case history.Zsh:
					if !strings.Contains(path, ".zshrc") {
						t.Errorf("Zsh config path should contain .zshrc, got: %s", path)
					}
				case history.PowerShell:
					if !strings.Contains(path, "profile.ps1") {
						t.Errorf("PowerShell config path should contain profile.ps1, got: %s", path)
					}
				case history.Cmd:
					if !strings.Contains(path, "cht_cmd_init.bat") {
						t.Errorf("Cmd config path should contain cht_cmd_init.bat, got: %s", path)
					}
				}
			}
		})
	}
}

func TestIntegrator_InstallAndRemoveIntegration(t *testing.T) {
	integrator := NewIntegrator()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cht_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test config file
	testConfigPath := filepath.Join(tempDir, "test_config")
	initialContent := "# Initial config content\nexport TEST_VAR=1\n"
	if err := os.WriteFile(testConfigPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Test script generation
	script, err := integrator.GetIntegrationScript(history.Bash)
	if err != nil {
		t.Fatalf("GetIntegrationScript() failed: %v", err)
	}

	// Manually test the install/remove logic by simulating it
	marker := integrator.getIntegrationMarker()
	endMarker := integrator.getIntegrationEndMarker()

	// Simulate installation
	integrationBlock := "\n" + marker + "\n" + script + "\n" + endMarker + "\n"
	newContent := initialContent + integrationBlock

	if err := os.WriteFile(testConfigPath, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Verify integration was installed
	content, err := os.ReadFile(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to read config after installation: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, marker) {
		t.Error("Integration marker not found after installation")
	}

	if !strings.Contains(contentStr, "__cht_record_command") {
		t.Error("Integration script not found after installation")
	}

	// Simulate removal
	startIdx := strings.Index(contentStr, marker)
	if startIdx == -1 {
		t.Fatal("Integration marker not found for removal test")
	}

	endIdx := strings.Index(contentStr[startIdx:], endMarker)
	if endIdx == -1 {
		t.Fatal("Integration end marker not found for removal test")
	}

	endIdx += startIdx + len(endMarker)
	removedContent := contentStr[:startIdx] + contentStr[endIdx:]

	if err := os.WriteFile(testConfigPath, []byte(removedContent), 0644); err != nil {
		t.Fatalf("Failed to write config after removal: %v", err)
	}

	// Verify integration was removed
	content, err = os.ReadFile(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to read config after removal: %v", err)
	}

	contentStr = string(content)
	if strings.Contains(contentStr, marker) {
		t.Error("Integration marker still found after removal")
	}

	// Verify original content is preserved
	if !strings.Contains(contentStr, "export TEST_VAR=1") {
		t.Error("Original config content was lost during integration removal")
	}
}

func TestIntegrator_SetupIntegration_UnsupportedShell(t *testing.T) {
	integrator := NewIntegrator()

	// Test with Zsh on Windows (should be unsupported)
	if runtime.GOOS == "windows" {
		err := integrator.SetupIntegration(history.Zsh)
		if err == nil {
			t.Error("SetupIntegration() should fail for unsupported shell on Windows")
		}
	}
}
