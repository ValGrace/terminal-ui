package shell

import (
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Detector implements ShellDetector interface
type Detector struct {
	platform PlatformAbstraction
}

// NewDetector creates a new shell detector
func NewDetector() *Detector {
	return &Detector{
		platform: NewPlatformAbstraction(),
	}
}

// DetectShell identifies the current shell type based on environment variables and process information
func (d *Detector) DetectShell() (history.ShellType, error) {
	platform := d.platform.GetPlatform()

	// Platform-specific detection logic
	switch platform {
	case PlatformWindows:
		return d.detectWindowsShell()
	case PlatformLinux, PlatformDarwin, PlatformFreeBSD:
		return d.detectUnixShell()
	default:
		return d.detectGenericShell()
	}
}

// detectWindowsShell detects shell on Windows systems
func (d *Detector) detectWindowsShell() (history.ShellType, error) {
	// Check for PowerShell specific environment variables
	if psVersion := os.Getenv("PSVersionTable"); psVersion != "" {
		return history.PowerShell, nil
	}
	if psModulePath := os.Getenv("PSModulePath"); psModulePath != "" {
		return history.PowerShell, nil
	}

	// Check for PowerShell process
	if d.isRunningInPowerShell() {
		return history.PowerShell, nil
	}

	// Check for Windows Command Prompt
	if comspec := os.Getenv("COMSPEC"); comspec != "" {
		if strings.Contains(strings.ToLower(comspec), "cmd.exe") {
			return history.Cmd, nil
		}
	}

	// Check for Bash (Git Bash, WSL, etc.)
	if d.isRunningInBash() {
		return history.Bash, nil
	}

	// Default to PowerShell on Windows if no other shell detected
	return history.PowerShell, nil
}

// detectUnixShell detects shell on Unix-like systems
func (d *Detector) detectUnixShell() (history.ShellType, error) {
	// Check SHELL environment variable first
	if shell := os.Getenv("SHELL"); shell != "" {
		shellName := filepath.Base(shell)
		switch strings.ToLower(shellName) {
		case "bash":
			return history.Bash, nil
		case "zsh":
			return history.Zsh, nil
		}
	}

	// Check for PowerShell Core on Unix
	if d.isRunningInPowerShell() {
		return history.PowerShell, nil
	}

	// Check process-specific indicators
	if d.isRunningInZsh() {
		return history.Zsh, nil
	}
	if d.isRunningInBash() {
		return history.Bash, nil
	}

	// Platform-specific defaults
	platform := d.platform.GetPlatform()
	if platform == PlatformDarwin {
		// macOS default changed to Zsh in Catalina
		return history.Zsh, nil
	}

	// Default to Bash for other Unix-like systems
	return history.Bash, nil
}

// detectGenericShell provides fallback detection for unknown platforms
func (d *Detector) detectGenericShell() (history.ShellType, error) {
	// Try environment variable detection
	if shell := os.Getenv("SHELL"); shell != "" {
		shellName := filepath.Base(shell)
		switch strings.ToLower(shellName) {
		case "bash":
			return history.Bash, nil
		case "zsh":
			return history.Zsh, nil
		}
	}

	// Try PowerShell detection
	if psModulePath := os.Getenv("PSModulePath"); psModulePath != "" {
		return history.PowerShell, nil
	}

	return history.Unknown, fmt.Errorf("unable to detect shell type on platform %s", d.platform.GetPlatform().String())
}

// isRunningInPowerShell checks if currently running in PowerShell
func (d *Detector) isRunningInPowerShell() bool {
	// Check PowerShell-specific environment variables
	psEnvVars := []string{
		"PSVersionTable",
		"PSModulePath",
		"POWERSHELL_DISTRIBUTION_CHANNEL",
		"PSExecutionPolicyPreference",
	}

	for _, envVar := range psEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}

// isRunningInBash checks if currently running in Bash
func (d *Detector) isRunningInBash() bool {
	// Check Bash-specific environment variables
	bashEnvVars := []string{
		"BASH_VERSION",
		"BASH",
		"BASH_SUBSHELL",
	}

	for _, envVar := range bashEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	// Check if SHELL points to bash
	if shell := os.Getenv("SHELL"); shell != "" {
		return strings.Contains(strings.ToLower(shell), "bash")
	}

	return false
}

// isRunningInZsh checks if currently running in Zsh
func (d *Detector) isRunningInZsh() bool {
	// Check Zsh-specific environment variables
	zshEnvVars := []string{
		"ZSH_VERSION",
		"ZSH_NAME",
	}

	for _, envVar := range zshEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	// Check if SHELL points to zsh
	if shell := os.Getenv("SHELL"); shell != "" {
		return strings.Contains(strings.ToLower(shell), "zsh")
	}

	return false
}

// GetShellPath returns the path to the shell executable
func (d *Detector) GetShellPath(shell history.ShellType) (string, error) {
	if shell == history.Unknown {
		return "", fmt.Errorf("cannot get path for unknown shell type")
	}

	if !d.IsShellSupported(shell) {
		return "", fmt.Errorf("shell %s is not supported on platform %s",
			shell.String(), d.platform.GetPlatform().String())
	}

	executableName := d.platform.GetShellExecutableName(shell)
	if executableName == "" {
		return "", fmt.Errorf("no executable name defined for shell %s", shell.String())
	}

	// Handle special cases
	switch shell {
	case history.PowerShell:
		return d.findPowerShellExecutable()
	case history.Cmd:
		if d.platform.GetPlatform() == PlatformWindows {
			if comspec := os.Getenv("COMSPEC"); comspec != "" {
				return comspec, nil
			}
			return "C:\\Windows\\System32\\cmd.exe", nil
		}
		return "", fmt.Errorf("cmd.exe is only available on Windows")
	default:
		return d.findExecutable(executableName)
	}
}

// findPowerShellExecutable finds the best available PowerShell executable
func (d *Detector) findPowerShellExecutable() (string, error) {
	platform := d.platform.GetPlatform()

	if platform == PlatformWindows {
		// Try PowerShell Core first, then Windows PowerShell
		candidates := []string{"pwsh.exe", "powershell.exe"}
		for _, candidate := range candidates {
			if fullPath, err := d.findExecutable(candidate); err == nil {
				return fullPath, nil
			}
		}
		return "", fmt.Errorf("PowerShell executable not found")
	}

	// Unix-like systems - PowerShell Core only
	return d.findExecutable("pwsh")
}

// IsShellSupported checks if a shell type is supported on the current platform
func (d *Detector) IsShellSupported(shell history.ShellType) bool {
	if shell == history.Unknown {
		return false
	}

	supportedShells := d.platform.GetSupportedShells()
	for _, supported := range supportedShells {
		if shell == supported {
			return true
		}
	}

	return false
}

// findExecutable searches for an executable in the system PATH
func (d *Detector) findExecutable(name string) (string, error) {
	// Check if the name is already a full path
	if filepath.IsAbs(name) {
		if d.platform.IsExecutable(name) {
			return name, nil
		}
		return "", fmt.Errorf("executable not found at path: %s", name)
	}

	// Search in PATH
	path := os.Getenv("PATH")
	if path == "" {
		return "", fmt.Errorf("PATH environment variable not set")
	}

	pathSeparator := d.platform.GetEnvironmentVariableSeparator()

	// Add platform-specific extension if needed
	if d.platform.GetPlatform() == PlatformWindows {
		if !strings.HasSuffix(strings.ToLower(name), ".exe") {
			name += ".exe"
		}
	}

	for _, dir := range strings.Split(path, pathSeparator) {
		if dir == "" {
			continue
		}
		fullPath := filepath.Join(dir, name)
		if d.platform.IsExecutable(fullPath) {
			return fullPath, nil
		}
	}

	return "", fmt.Errorf("executable '%s' not found in PATH", name)
}
