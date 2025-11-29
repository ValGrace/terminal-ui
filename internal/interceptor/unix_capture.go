package interceptor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"github.com/ValGrace/command-history-tracker/pkg/shell"
)

// UnixCapture provides Unix-specific command capture functionality
type UnixCapture struct {
	*CommandCapture
	platform *shell.UnixPlatform
}

// NewUnixCapture creates a new Unix-specific command capture instance
func NewUnixCapture(storage history.StorageEngine, cfg *config.Config) *UnixCapture {
	if runtime.GOOS == "windows" {
		return nil
	}

	return &UnixCapture{
		CommandCapture: NewCommandCapture(storage, cfg),
		platform:       shell.NewUnixPlatform(),
	}
}

// CaptureUnixCommand captures a command specifically from Unix shells
func (u *UnixCapture) CaptureUnixCommand() error {
	// Detect the current Unix shell
	shell, err := u.detectUnixShell()
	if err != nil {
		return fmt.Errorf("failed to detect Unix shell: %w", err)
	}

	// Use shell-specific capture method
	switch shell {
	case history.Bash:
		return u.CaptureBashCommand()
	case history.Zsh:
		return u.CaptureZshCommand()
	case history.PowerShell:
		return u.CapturePowerShellCoreCommand()
	default:
		return u.CaptureCommand() // Fallback to generic capture
	}
}

// detectUnixShell detects the specific Unix shell being used
func (u *UnixCapture) detectUnixShell() (history.ShellType, error) {
	// Use platform-specific detection
	if shell := u.platform.GetShellFromEnvironment(); shell != history.Unknown {
		return shell, nil
	}

	// Check for PowerShell Core
	if u.isInPowerShellCore() {
		return history.PowerShell, nil
	}

	// Check process-specific indicators
	if u.isInZsh() {
		return history.Zsh, nil
	}

	if u.isInBash() {
		return history.Bash, nil
	}

	// If environment-based checks failed, try to infer from process info
	if info, err := u.getProcessInfo(); err == nil && info != nil && info.Platform != "" {
		execName := strings.ToLower(filepath.Base(info.Platform))
		if strings.Contains(execName, "zsh") {
			return history.Zsh, nil
		}
		if strings.Contains(execName, "bash") {
			return history.Bash, nil
		}
		if strings.Contains(execName, "pwsh") || strings.Contains(execName, "powershell") {
			return history.PowerShell, nil
		}
	}

	return history.Unknown, fmt.Errorf("unable to detect Unix shell")
}

// isInBash checks if running in Bash
func (u *UnixCapture) isInBash() bool {
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

// isInZsh checks if running in Zsh
func (u *UnixCapture) isInZsh() bool {
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

// isInPowerShellCore checks if running in PowerShell Core on Unix
func (u *UnixCapture) isInPowerShellCore() bool {
	// Check PowerShell-specific environment variables
	psEnvVars := []string{
		"PSVersionTable",
		"PSModulePath",
		"POWERSHELL_DISTRIBUTION_CHANNEL",
	}

	for _, envVar := range psEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}

// CaptureBashCommand captures commands from Bash
func (u *UnixCapture) CaptureBashCommand() error {
	// Get command from environment or Bash history
	command := os.Getenv("CHT_COMMAND")
	if command == "" {
		command = u.getBashLastCommand()
	}

	if command == "" {
		return fmt.Errorf("no Bash command found to capture")
	}

	// Get Bash-specific metadata
	metadata := u.getBashMetadata()

	// Create enhanced command record
	cmdRecord := &history.CommandRecord{
		Command:   command,
		Directory: u.getCurrentDirectory(),
		Shell:     history.Bash,
		ExitCode:  u.getExitCode(),
		Duration:  u.getDuration(),
		Timestamp: time.Now(),
		Tags:      []string{"bash", "unix"},
	}

	// Add Bash-specific tags
	u.addBashTags(cmdRecord, metadata)

	// Generate ID and enhance
	cmdRecord.ID = u.envManager.GenerateCommandID(cmdRecord.Command, cmdRecord.Directory, cmdRecord.Timestamp)

	if err := u.enhanceCommandRecord(cmdRecord); err != nil {
		return fmt.Errorf("failed to enhance Bash command record: %w", err)
	}

	// Store the command
	return u.storage.SaveCommand(*cmdRecord)
}

// CaptureZshCommand captures commands from Zsh
func (u *UnixCapture) CaptureZshCommand() error {
	// Get command from environment or Zsh history
	command := os.Getenv("CHT_COMMAND")
	if command == "" {
		command = u.getZshLastCommand()
	}

	if command == "" {
		return fmt.Errorf("no Zsh command found to capture")
	}

	// Get Zsh-specific metadata
	metadata := u.getZshMetadata()

	// Create enhanced command record
	cmdRecord := &history.CommandRecord{
		Command:   command,
		Directory: u.getCurrentDirectory(),
		Shell:     history.Zsh,
		ExitCode:  u.getExitCode(),
		Duration:  u.getDuration(),
		Timestamp: time.Now(),
		Tags:      []string{"zsh", "unix"},
	}

	// Add Zsh-specific tags
	u.addZshTags(cmdRecord, metadata)

	// Generate ID and enhance
	cmdRecord.ID = u.envManager.GenerateCommandID(cmdRecord.Command, cmdRecord.Directory, cmdRecord.Timestamp)

	if err := u.enhanceCommandRecord(cmdRecord); err != nil {
		return fmt.Errorf("failed to enhance Zsh command record: %w", err)
	}

	// Store the command
	return u.storage.SaveCommand(*cmdRecord)
}

// CapturePowerShellCoreCommand captures commands from PowerShell Core on Unix
func (u *UnixCapture) CapturePowerShellCoreCommand() error {
	// PowerShell Core on Unix
	command := os.Getenv("CHT_COMMAND")
	if command == "" {
		command = u.getPowerShellCoreLastCommand()
	}

	if command == "" {
		return fmt.Errorf("no PowerShell Core command found to capture")
	}

	// Create command record
	cmdRecord := &history.CommandRecord{
		Command:   command,
		Directory: u.getCurrentDirectory(),
		Shell:     history.PowerShell,
		ExitCode:  u.getExitCode(),
		Duration:  u.getDuration(),
		Timestamp: time.Now(),
		Tags:      []string{"powershell", "powershell-core", "unix"},
	}

	// Add PowerShell Core specific tags
	u.addPowerShellCoreTags(cmdRecord)

	// Generate ID and enhance
	cmdRecord.ID = u.envManager.GenerateCommandID(cmdRecord.Command, cmdRecord.Directory, cmdRecord.Timestamp)

	if err := u.enhanceCommandRecord(cmdRecord); err != nil {
		return fmt.Errorf("failed to enhance PowerShell Core command record: %w", err)
	}

	// Store the command
	return u.storage.SaveCommand(*cmdRecord)
}

// getBashLastCommand attempts to get the last executed Bash command
func (u *UnixCapture) getBashLastCommand() string {
	// Try to get from BASH_COMMAND environment variable
	if bashCmd := os.Getenv("BASH_COMMAND"); bashCmd != "" {
		return bashCmd
	}

	// Try to read from Bash history file
	return u.getLastCommandFromHistoryFile("~/.bash_history")
}

// getZshLastCommand attempts to get the last executed Zsh command
func (u *UnixCapture) getZshLastCommand() string {
	// Try to read from Zsh history file
	histFile := os.Getenv("HISTFILE")
	if histFile == "" {
		histFile = "~/.zsh_history"
	}

	return u.getLastCommandFromHistoryFile(histFile)
}

// getPowerShellCoreLastCommand attempts to get the last executed PowerShell Core command
func (u *UnixCapture) getPowerShellCoreLastCommand() string {
	// Try to execute pwsh to get the last command from history
	cmd := exec.Command("pwsh", "-Command", "Get-History -Count 1 | Select-Object -ExpandProperty CommandLine")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// getLastCommandFromHistoryFile reads the last command from a shell history file
func (u *UnixCapture) getLastCommandFromHistoryFile(historyFile string) string {
	// Expand tilde to home directory
	if strings.HasPrefix(historyFile, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		historyFile = filepath.Join(homeDir, historyFile[2:])
	}

	file, err := os.Open(historyFile)
	if err != nil {
		return ""
	}
	defer file.Close()

	var lastLine string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			// For Zsh extended history format, extract just the command
			if strings.Contains(line, ";") && strings.HasPrefix(line, ":") {
				parts := strings.SplitN(line, ";", 2)
				if len(parts) == 2 {
					line = parts[1]
				}
			}
			lastLine = line
		}
	}

	return lastLine
}

// getBashMetadata gets Bash-specific metadata
func (u *UnixCapture) getBashMetadata() map[string]string {
	metadata := make(map[string]string)

	// Get Bash version
	if bashVersion := os.Getenv("BASH_VERSION"); bashVersion != "" {
		metadata["bash_version"] = bashVersion
	}

	// Get shell level
	if shlvl := os.Getenv("SHLVL"); shlvl != "" {
		metadata["shell_level"] = shlvl
	}

	// Check if in subshell
	if subshell := os.Getenv("BASH_SUBSHELL"); subshell != "" && subshell != "0" {
		metadata["subshell"] = subshell
	}

	// Get terminal info
	if term := os.Getenv("TERM"); term != "" {
		metadata["terminal"] = term
	}

	return metadata
}

// getZshMetadata gets Zsh-specific metadata
func (u *UnixCapture) getZshMetadata() map[string]string {
	metadata := make(map[string]string)

	// Get Zsh version
	if zshVersion := os.Getenv("ZSH_VERSION"); zshVersion != "" {
		metadata["zsh_version"] = zshVersion
	}

	// Get shell level
	if shlvl := os.Getenv("SHLVL"); shlvl != "" {
		metadata["shell_level"] = shlvl
	}

	// Check for Oh My Zsh
	if zshDir := os.Getenv("ZSH"); zshDir != "" {
		metadata["oh_my_zsh"] = "true"
	}

	// Get Zsh theme
	if zshTheme := os.Getenv("ZSH_THEME"); zshTheme != "" {
		metadata["zsh_theme"] = zshTheme
	}

	return metadata
}

// addBashTags adds Bash-specific tags to command record
func (u *UnixCapture) addBashTags(cmdRecord *history.CommandRecord, metadata map[string]string) {
	// Add Bash version tag
	if version, ok := metadata["bash_version"]; ok && version != "" {
		// Extract major version
		if parts := strings.Split(version, "."); len(parts) > 0 {
			cmdRecord.AddTag(fmt.Sprintf("bash-%s", parts[0]))
		}
	}

	// Add subshell tag
	if subshell, ok := metadata["subshell"]; ok && subshell != "" && subshell != "0" {
		cmdRecord.AddTag("bash-subshell")
	}

	// Add terminal tag
	if terminal, ok := metadata["terminal"]; ok && terminal != "" {
		cmdRecord.AddTag(fmt.Sprintf("term-%s", terminal))
	}

	// Analyze Bash-specific command patterns
	command := strings.ToLower(cmdRecord.Command)

	if strings.Contains(command, "[[") || strings.Contains(command, "]]") {
		cmdRecord.AddTag("bash-test")
	}

	if strings.Contains(command, "source ") || strings.Contains(command, ". ") {
		cmdRecord.AddTag("bash-source")
	}

	if strings.Contains(command, "export ") {
		cmdRecord.AddTag("bash-export")
	}

	if strings.Contains(command, "function ") || strings.Contains(command, "() {") {
		cmdRecord.AddTag("bash-function")
	}
}

// addZshTags adds Zsh-specific tags to command record
func (u *UnixCapture) addZshTags(cmdRecord *history.CommandRecord, metadata map[string]string) {
	// Add Zsh version tag
	if version, ok := metadata["zsh_version"]; ok && version != "" {
		// Extract major version
		if parts := strings.Split(version, "."); len(parts) > 0 {
			cmdRecord.AddTag(fmt.Sprintf("zsh-%s", parts[0]))
		}
	}

	// Add Oh My Zsh tag
	if _, ok := metadata["oh_my_zsh"]; ok {
		cmdRecord.AddTag("oh-my-zsh")
	}

	// Add theme tag
	if theme, ok := metadata["zsh_theme"]; ok && theme != "" {
		cmdRecord.AddTag(fmt.Sprintf("zsh-theme-%s", theme))
	}

	// Analyze Zsh-specific command patterns
	command := strings.ToLower(cmdRecord.Command)

	if strings.Contains(command, "autoload") {
		cmdRecord.AddTag("zsh-autoload")
	}

	if strings.Contains(command, "setopt") || strings.Contains(command, "unsetopt") {
		cmdRecord.AddTag("zsh-option")
	}

	if strings.Contains(command, "compinit") {
		cmdRecord.AddTag("zsh-completion")
	}

	if strings.Contains(command, "bindkey") {
		cmdRecord.AddTag("zsh-keybind")
	}
}

// addPowerShellCoreTags adds PowerShell Core specific tags
func (u *UnixCapture) addPowerShellCoreTags(cmdRecord *history.CommandRecord) {
	cmdRecord.AddTag("powershell-core")

	// Add platform-specific tag
	cmdRecord.AddTag(fmt.Sprintf("platform-%s", runtime.GOOS))

	// Analyze PowerShell command patterns
	command := strings.ToLower(cmdRecord.Command)

	if strings.Contains(command, "get-") || strings.Contains(command, "set-") ||
		strings.Contains(command, "new-") || strings.Contains(command, "remove-") {
		cmdRecord.AddTag("powershell-cmdlet")
	}

	if strings.Contains(command, "|") {
		cmdRecord.AddTag("powershell-pipeline")
	}
}

// getCurrentDirectory gets the current directory with Unix-specific handling
func (u *UnixCapture) getCurrentDirectory() string {
	// Try environment variable first
	if dir := os.Getenv("CHT_DIRECTORY"); dir != "" {
		return u.platform.NormalizePath(dir)
	}

	// Get current working directory
	if currentDir, err := os.Getwd(); err == nil {
		return u.platform.NormalizePath(currentDir)
	}

	// Fallback to home directory
	if homeDir, err := u.platform.GetHomeDirectory(); err == nil {
		return homeDir
	}

	return "/"
}

// getExitCode gets the exit code with Unix-specific handling
func (u *UnixCapture) getExitCode() int {
	if exitCodeStr := os.Getenv("CHT_EXIT_CODE"); exitCodeStr != "" {
		if exitCode, err := strconv.Atoi(exitCodeStr); err == nil {
			return exitCode
		}
	}

	// Try to get from shell-specific variables
	if exitCode := os.Getenv("?"); exitCode != "" {
		if code, err := strconv.Atoi(exitCode); err == nil {
			return code
		}
	}

	return 0 // Default to success
}

// getDuration gets the command duration with Unix-specific handling
func (u *UnixCapture) getDuration() time.Duration {
	if durationStr := os.Getenv("CHT_DURATION"); durationStr != "" {
		// Try parsing as milliseconds first
		if ms, err := strconv.ParseInt(durationStr, 10, 64); err == nil {
			return time.Duration(ms) * time.Millisecond
		}

		// Try parsing as Go duration string
		if duration, err := time.ParseDuration(durationStr); err == nil {
			return duration
		}
	}

	return 0
}

// GetUnixShellInfo returns detailed information about Unix shells
func (u *UnixCapture) GetUnixShellInfo() (*UnixShellInfo, error) {
	info := &UnixShellInfo{
		Platform:     runtime.GOOS,
		Distribution: u.platform.GetDistribution(),
		IsWSL:        u.platform.IsWSL(),
		Shells:       make(map[string]UnixShellDetails),
	}

	// Check Bash availability
	if bashPath, err := exec.LookPath("bash"); err == nil {
		info.Shells["bash"] = UnixShellDetails{
			Available: true,
			Path:      bashPath,
			Version:   u.getBashVersion(),
		}
	}

	// Check Zsh availability
	if zshPath, err := exec.LookPath("zsh"); err == nil {
		info.Shells["zsh"] = UnixShellDetails{
			Available: true,
			Path:      zshPath,
			Version:   u.getZshVersion(),
		}
	}

	// Check PowerShell Core availability
	if pwshPath, err := exec.LookPath("pwsh"); err == nil {
		info.Shells["powershell"] = UnixShellDetails{
			Available: true,
			Path:      pwshPath,
			Version:   u.getPowerShellCoreVersion(),
		}
	}

	return info, nil
}

// getBashVersion gets the Bash version
func (u *UnixCapture) getBashVersion() string {
	if version := os.Getenv("BASH_VERSION"); version != "" {
		return version
	}

	cmd := exec.Command("bash", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}

	return "unknown"
}

// getZshVersion gets the Zsh version
func (u *UnixCapture) getZshVersion() string {
	if version := os.Getenv("ZSH_VERSION"); version != "" {
		return version
	}

	cmd := exec.Command("zsh", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(output))
}

// getPowerShellCoreVersion gets the PowerShell Core version
func (u *UnixCapture) getPowerShellCoreVersion() string {
	cmd := exec.Command("pwsh", "-Command", "$PSVersionTable.PSVersion.ToString()")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(output))
}

// getProcessInfo gets Unix-specific process information
func (u *UnixCapture) getProcessInfo() (*UnixProcessInfo, error) {
	info := &UnixProcessInfo{
		PID:  os.Getpid(),
		PPID: os.Getppid(),
		UID:  os.Getuid(),
		GID:  os.Getgid(),
	}

	// Note: Process group and session ID retrieval would require
	// platform-specific syscalls that are not available in all Go builds
	// This is left as a placeholder for future enhancement

	return info, nil
}

// UnixShellInfo contains information about available Unix shells
type UnixShellInfo struct {
	Platform     string                      `json:"platform"`
	Distribution string                      `json:"distribution"`
	IsWSL        bool                        `json:"is_wsl"`
	Shells       map[string]UnixShellDetails `json:"shells"`
}

// UnixShellDetails contains details about a specific Unix shell
type UnixShellDetails struct {
	Available bool   `json:"available"`
	Path      string `json:"path"`
	Version   string `json:"version"`
}

// UnixProcessInfo contains Unix-specific process information
type UnixProcessInfo struct {
	PID      int    `json:"pid"`
	PPID     int    `json:"ppid"`
	UID      int    `json:"uid"`
	GID      int    `json:"gid"`
	PGID     int    `json:"pgid,omitempty"`
	SID      int    `json:"sid,omitempty"`
	Platform string `json:"platform"`
}
