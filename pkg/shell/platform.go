package shell

import (
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Platform represents the operating system platform
type Platform int

const (
	PlatformUnknown Platform = iota
	PlatformWindows
	PlatformLinux
	PlatformDarwin
	PlatformFreeBSD
)

// String returns the string representation of Platform
func (p Platform) String() string {
	switch p {
	case PlatformWindows:
		return "windows"
	case PlatformLinux:
		return "linux"
	case PlatformDarwin:
		return "darwin"
	case PlatformFreeBSD:
		return "freebsd"
	default:
		return "unknown"
	}
}

// PlatformAbstraction provides platform-specific functionality
type PlatformAbstraction interface {
	// GetPlatform returns the current platform
	GetPlatform() Platform

	// GetSupportedShells returns shells supported on this platform
	GetSupportedShells() []history.ShellType

	// GetDefaultShell returns the default shell for this platform
	GetDefaultShell() history.ShellType

	// GetShellExecutableName returns the executable name for a shell
	GetShellExecutableName(shell history.ShellType) string

	// GetShellConfigPath returns the configuration file path for a shell
	GetShellConfigPath(shell history.ShellType) (string, error)

	// GetEnvironmentVariableSeparator returns the PATH separator for this platform
	GetEnvironmentVariableSeparator() string

	// NormalizePath normalizes a file path for this platform
	NormalizePath(path string) string

	// GetHomeDirectory returns the user's home directory
	GetHomeDirectory() (string, error)

	// IsExecutable checks if a file is executable on this platform
	IsExecutable(path string) bool

	// GetProcessEnvironment returns process-specific environment variables
	GetProcessEnvironment() map[string]string
}

// platformAbstraction implements PlatformAbstraction
type platformAbstraction struct {
	platform Platform
}

// NewPlatformAbstraction creates a new platform abstraction
func NewPlatformAbstraction() PlatformAbstraction {
	return &platformAbstraction{
		platform: detectPlatform(),
	}
}

// detectPlatform detects the current platform
func detectPlatform() Platform {
	switch runtime.GOOS {
	case "windows":
		return PlatformWindows
	case "linux":
		return PlatformLinux
	case "darwin":
		return PlatformDarwin
	case "freebsd":
		return PlatformFreeBSD
	default:
		return PlatformUnknown
	}
}

// GetPlatform returns the current platform
func (p *platformAbstraction) GetPlatform() Platform {
	return p.platform
}

// GetSupportedShells returns shells supported on this platform
func (p *platformAbstraction) GetSupportedShells() []history.ShellType {
	switch p.platform {
	case PlatformWindows:
		return []history.ShellType{
			history.PowerShell,
			history.Cmd,
			history.Bash, // Available through WSL, Git Bash, etc.
		}
	case PlatformLinux, PlatformDarwin, PlatformFreeBSD:
		return []history.ShellType{
			history.Bash,
			history.Zsh,
			history.PowerShell, // PowerShell Core is available on Unix
		}
	default:
		return []history.ShellType{history.Bash} // Fallback
	}
}

// GetDefaultShell returns the default shell for this platform
func (p *platformAbstraction) GetDefaultShell() history.ShellType {
	switch p.platform {
	case PlatformWindows:
		return history.PowerShell
	case PlatformDarwin:
		// macOS switched to Zsh as default in Catalina
		return history.Zsh
	case PlatformLinux, PlatformFreeBSD:
		return history.Bash
	default:
		return history.Bash
	}
}

// GetShellExecutableName returns the executable name for a shell
func (p *platformAbstraction) GetShellExecutableName(shell history.ShellType) string {
	switch shell {
	case history.PowerShell:
		if p.platform == PlatformWindows {
			// Try PowerShell Core first, then Windows PowerShell
			return "pwsh.exe" // Will fallback to powershell.exe in detection
		}
		return "pwsh" // PowerShell Core on Unix
	case history.Bash:
		if p.platform == PlatformWindows {
			return "bash.exe"
		}
		return "bash"
	case history.Zsh:
		if p.platform == PlatformWindows {
			return "zsh.exe"
		}
		return "zsh"
	case history.Cmd:
		return "cmd.exe"
	default:
		return ""
	}
}

// GetShellConfigPath returns the configuration file path for a shell
func (p *platformAbstraction) GetShellConfigPath(shell history.ShellType) (string, error) {
	homeDir, err := p.GetHomeDirectory()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	switch shell {
	case history.PowerShell:
		if p.platform == PlatformWindows {
			// Windows PowerShell profile
			return filepath.Join(homeDir, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"), nil
		}
		// PowerShell Core profile on Unix
		return filepath.Join(homeDir, ".config", "powershell", "Microsoft.PowerShell_profile.ps1"), nil

	case history.Bash:
		if p.platform == PlatformWindows {
			// Git Bash or WSL
			return filepath.Join(homeDir, ".bashrc"), nil
		}
		// Unix systems
		return filepath.Join(homeDir, ".bashrc"), nil

	case history.Zsh:
		return filepath.Join(homeDir, ".zshrc"), nil

	case history.Cmd:
		// CMD doesn't have a standard config file, use a custom one
		return filepath.Join(homeDir, "cht_cmd_init.bat"), nil

	default:
		return "", fmt.Errorf("unsupported shell type: %s", shell.String())
	}
}

// GetEnvironmentVariableSeparator returns the PATH separator for this platform
func (p *platformAbstraction) GetEnvironmentVariableSeparator() string {
	if p.platform == PlatformWindows {
		return ";"
	}
	return ":"
}

// NormalizePath normalizes a file path for this platform
func (p *platformAbstraction) NormalizePath(path string) string {
	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path // Use original if conversion fails
	}

	// Clean the path
	cleanPath := filepath.Clean(absPath)

	// Convert to forward slashes for consistency across platforms
	normalizedPath := filepath.ToSlash(cleanPath)

	return normalizedPath
}

// GetHomeDirectory returns the user's home directory
func (p *platformAbstraction) GetHomeDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return homeDir, nil
}

// IsExecutable checks if a file is executable on this platform
func (p *platformAbstraction) IsExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if p.platform == PlatformWindows {
		// On Windows, check file extension
		ext := strings.ToLower(filepath.Ext(path))
		executableExts := []string{".exe", ".bat", ".cmd", ".com", ".ps1"}
		for _, execExt := range executableExts {
			if ext == execExt {
				return true
			}
		}
		return false
	}

	// On Unix-like systems, check execute permission
	mode := info.Mode()
	return mode&0111 != 0 // Check if any execute bit is set
}

// GetProcessEnvironment returns process-specific environment variables
func (p *platformAbstraction) GetProcessEnvironment() map[string]string {
	env := make(map[string]string)

	// Add platform-specific environment variables
	env["CHT_PLATFORM"] = p.platform.String()
	env["CHT_ARCH"] = runtime.GOARCH

	// Add shell-specific environment detection
	if p.platform == PlatformWindows {
		// Windows-specific environment variables
		if psModulePath := os.Getenv("PSModulePath"); psModulePath != "" {
			env["CHT_PS_AVAILABLE"] = "true"
		}
		if comspec := os.Getenv("COMSPEC"); comspec != "" {
			env["CHT_CMD_AVAILABLE"] = "true"
		}
	} else {
		// Unix-like systems
		if shell := os.Getenv("SHELL"); shell != "" {
			env["CHT_DEFAULT_SHELL"] = filepath.Base(shell)
		}
	}

	return env
}

// WindowsPlatform provides Windows-specific functionality
type WindowsPlatform struct {
	*platformAbstraction
}

// NewWindowsPlatform creates a Windows-specific platform abstraction
func NewWindowsPlatform() *WindowsPlatform {
	return &WindowsPlatform{
		platformAbstraction: &platformAbstraction{platform: PlatformWindows},
	}
}

// GetPowerShellVariant detects which PowerShell variant is available
func (w *WindowsPlatform) GetPowerShellVariant() string {
	// Check for PowerShell Core first
	if _, err := w.findExecutable("pwsh.exe"); err == nil {
		return "pwsh"
	}
	// Fallback to Windows PowerShell
	if _, err := w.findExecutable("powershell.exe"); err == nil {
		return "powershell"
	}
	return ""
}

// GetCmdPath returns the path to cmd.exe
func (w *WindowsPlatform) GetCmdPath() string {
	if comspec := os.Getenv("COMSPEC"); comspec != "" {
		return comspec
	}
	return "C:\\Windows\\System32\\cmd.exe"
}

// findExecutable searches for an executable in the system PATH
func (w *WindowsPlatform) findExecutable(name string) (string, error) {
	path := os.Getenv("PATH")
	if path == "" {
		return "", fmt.Errorf("PATH environment variable not set")
	}

	for _, dir := range strings.Split(path, ";") {
		if dir == "" {
			continue
		}
		fullPath := filepath.Join(dir, name)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			return fullPath, nil
		}
	}

	return "", fmt.Errorf("executable '%s' not found in PATH", name)
}

// UnixPlatform provides Unix-like system functionality
type UnixPlatform struct {
	*platformAbstraction
}

// NewUnixPlatform creates a Unix-specific platform abstraction
func NewUnixPlatform() *UnixPlatform {
	platform := detectPlatform()
	return &UnixPlatform{
		platformAbstraction: &platformAbstraction{platform: platform},
	}
}

// GetShellFromEnvironment detects the shell from environment variables
func (u *UnixPlatform) GetShellFromEnvironment() history.ShellType {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return history.Unknown
	}

	shellName := filepath.Base(shell)
	switch strings.ToLower(shellName) {
	case "bash":
		return history.Bash
	case "zsh":
		return history.Zsh
	case "fish":
		// Fish shell - not currently supported but could be added
		return history.Unknown
	default:
		return history.Unknown
	}
}

// IsWSL detects if running under Windows Subsystem for Linux
func (u *UnixPlatform) IsWSL() bool {
	// Check for WSL-specific environment variables or files
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}

	// Check /proc/version for WSL signature
	if data, err := os.ReadFile("/proc/version"); err == nil {
		version := strings.ToLower(string(data))
		return strings.Contains(version, "microsoft") || strings.Contains(version, "wsl")
	}

	return false
}

// GetDistribution returns the Linux distribution name
func (u *UnixPlatform) GetDistribution() string {
	// Try to read from /etc/os-release
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "ID=") {
				return strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			}
		}
	}

	// Fallback to checking specific distribution files
	distFiles := map[string]string{
		"/etc/debian_version": "debian",
		"/etc/redhat-release": "redhat",
		"/etc/arch-release":   "arch",
		"/etc/gentoo-release": "gentoo",
	}

	for file, dist := range distFiles {
		if _, err := os.Stat(file); err == nil {
			return dist
		}
	}

	return "unknown"
}

// PlatformCapture provides platform-specific command capture functionality
type PlatformCapture struct {
	platform PlatformAbstraction
}

// NewPlatformCapture creates a new platform-specific capture instance
func NewPlatformCapture() *PlatformCapture {
	return &PlatformCapture{
		platform: NewPlatformAbstraction(),
	}
}

// GetPlatformMetadata returns platform-specific metadata for command records
func (pc *PlatformCapture) GetPlatformMetadata() map[string]string {
	metadata := make(map[string]string)

	// Add basic platform information
	metadata["platform"] = pc.platform.GetPlatform().String()
	metadata["arch"] = runtime.GOARCH

	// Add platform-specific details
	switch pc.platform.GetPlatform() {
	case PlatformWindows:
		if wp, ok := pc.platform.(*WindowsPlatform); ok {
			if variant := wp.GetPowerShellVariant(); variant != "" {
				metadata["powershell_variant"] = variant
			}
		}
	case PlatformLinux:
		if up, ok := pc.platform.(*UnixPlatform); ok {
			metadata["distribution"] = up.GetDistribution()
			if up.IsWSL() {
				metadata["wsl"] = "true"
			}
		}
	case PlatformDarwin:
		// macOS-specific metadata could be added here
		metadata["macos"] = "true"
	}

	return metadata
}

// ValidateShellSupport checks if a shell is supported on the current platform
func (pc *PlatformCapture) ValidateShellSupport(shell history.ShellType) error {
	supportedShells := pc.platform.GetSupportedShells()

	for _, supported := range supportedShells {
		if shell == supported {
			return nil
		}
	}

	return fmt.Errorf("shell %s is not supported on platform %s",
		shell.String(), pc.platform.GetPlatform().String())
}

// GetOptimalShellIntegration returns the best shell integration approach for the platform
func (pc *PlatformCapture) GetOptimalShellIntegration() (history.ShellType, string, error) {
	// Try to detect current shell first
	detector := NewDetector()
	currentShell, err := detector.DetectShell()
	if err == nil && currentShell != history.Unknown {
		if err := pc.ValidateShellSupport(currentShell); err == nil {
			return currentShell, "detected", nil
		}
	}

	// Fall back to platform default
	defaultShell := pc.platform.GetDefaultShell()
	if err := pc.ValidateShellSupport(defaultShell); err == nil {
		return defaultShell, "default", nil
	}

	// Use first supported shell as last resort
	supportedShells := pc.platform.GetSupportedShells()
	if len(supportedShells) > 0 {
		return supportedShells[0], "fallback", nil
	}

	return history.Unknown, "", fmt.Errorf("no supported shells found on platform %s",
		pc.platform.GetPlatform().String())
}
