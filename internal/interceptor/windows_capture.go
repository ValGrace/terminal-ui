package interceptor

import (
	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"github.com/ValGrace/command-history-tracker/pkg/shell"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
    "strconv"
	"strings"
	"time"
)

// WindowsCapture provides Windows-specific command capture functionality
type WindowsCapture struct {
	*CommandCapture
	platform *shell.WindowsPlatform
}

// NewWindowsCapture creates a new Windows-specific command capture instance
func NewWindowsCapture(storage history.StorageEngine, cfg *config.Config) *WindowsCapture {
	if runtime.GOOS != "windows" {
		return nil
	}

	return &WindowsCapture{
		CommandCapture: NewCommandCapture(storage, cfg),
		platform:       shell.NewWindowsPlatform(),
	}
}

// CaptureWindowsCommand captures a command specifically from Windows shells
func (w *WindowsCapture) CaptureWindowsCommand() error {
	// Detect the current Windows shell
	shell, err := w.detectWindowsShell()
	if err != nil {
		return fmt.Errorf("failed to detect Windows shell: %w", err)
	}

	// Use shell-specific capture method
	switch shell {
	case history.PowerShell:
		return w.CapturePowerShellCommand()
	case history.Cmd:
		return w.captureCmdCommand()
	case history.Bash:
		return w.CaptureWindowsBashCommand()
	default:
		return w.CaptureCommand() // Fallback to generic capture
	}
}

// detectWindowsShell detects the specific Windows shell being used
func (w *WindowsCapture) detectWindowsShell() (history.ShellType, error) {
	// Check for PowerShell-specific environment variables
	if w.isInPowerShell() {
		return history.PowerShell, nil
	}

	// Check for Command Prompt
	if w.isInCmd() {
		return history.Cmd, nil
	}

	// Check for Bash (Git Bash, WSL, etc.)
	if w.isInWindowsBash() {
		return history.Bash, nil
	}

	return history.Unknown, fmt.Errorf("unable to detect Windows shell")
}

// isInPowerShell checks if running in PowerShell
func (w *WindowsCapture) isInPowerShell() bool {
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

	// Check if parent process is PowerShell
	return w.isParentProcessPowerShell()
}

// isInCmd checks if running in Command Prompt
func (w *WindowsCapture) isInCmd() bool {
	// Check COMSPEC environment variable
	comspec := os.Getenv("COMSPEC")
	if comspec != "" && strings.Contains(strings.ToLower(comspec), "cmd.exe") {
		// Additional check to ensure we're actually in cmd, not just have it available
		return w.isParentProcessCmd()
	}

	return false
}

// isInWindowsBash checks if running in Bash on Windows (Git Bash, WSL, etc.)
func (w *WindowsCapture) isInWindowsBash() bool {
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

	// Check for WSL
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}

	// Check for Git Bash
	if strings.Contains(strings.ToLower(os.Getenv("MSYSTEM")), "mingw") {
		return true
	}

	return false
}

// CapturePowerShellCommand captures commands from PowerShell
func (w *WindowsCapture) CapturePowerShellCommand() error {
	// PowerShell-specific environment variable extraction
	command := os.Getenv("CHT_COMMAND")
	if command == "" {
		// Try to get from PowerShell history
		command = w.getPowerShellLastCommand()
	}

	if command == "" {
		return fmt.Errorf("no PowerShell command found to capture")
	}

	// Get PowerShell-specific metadata
	metadata := w.getPowerShellMetadata()

	// Create enhanced command record
	cmdRecord := &history.CommandRecord{
		Command:   command,
		Directory: w.getCurrentDirectory(),
		Shell:     history.PowerShell,
		ExitCode:  w.getExitCode(),
		Duration:  w.getDuration(),
		Timestamp: time.Now(),
		Tags:      []string{"powershell", "windows"},
	}

	// Add PowerShell-specific tags
	w.addPowerShellTags(cmdRecord, metadata)

	// Generate ID and enhance
	cmdRecord.ID = w.envManager.GenerateCommandID(cmdRecord.Command, cmdRecord.Directory, cmdRecord.Timestamp)

	if err := w.enhanceCommandRecord(cmdRecord); err != nil {
		return fmt.Errorf("failed to enhance PowerShell command record: %w", err)
	}

	// Store the command
	return w.storage.SaveCommand(*cmdRecord)
}

// captureCmdCommand captures commands from Command Prompt
func (w *WindowsCapture) captureCmdCommand() error {
	// CMD has limited hooking capabilities, so we rely on environment variables
	command := os.Getenv("CHT_COMMAND")
	if command == "" {
		return fmt.Errorf("no CMD command found to capture")
	}

	// Create command record
	cmdRecord := &history.CommandRecord{
		Command:   command,
		Directory: w.getCurrentDirectory(),
		Shell:     history.Cmd,
		ExitCode:  w.getExitCode(),
		Duration:  w.getDuration(),
		Timestamp: time.Now(),
		Tags:      []string{"cmd", "windows"},
	}

	// Generate ID and enhance
	cmdRecord.ID = w.envManager.GenerateCommandID(cmdRecord.Command, cmdRecord.Directory, cmdRecord.Timestamp)

	if err := w.enhanceCommandRecord(cmdRecord); err != nil {
		return fmt.Errorf("failed to enhance CMD command record: %w", err)
	}

	// Store the command
	return w.storage.SaveCommand(*cmdRecord)
}

// CaptureWindowsBashCommand captures commands from Bash on Windows
func (w *WindowsCapture) CaptureWindowsBashCommand() error {
	// Get command from environment
	command := os.Getenv("CHT_COMMAND")
	if command == "" {
		return fmt.Errorf("no Bash command found to capture")
	}

	// Create command record with Windows-specific handling
	cmdRecord := &history.CommandRecord{
		Command:   command,
		Directory: w.getCurrentDirectory(),
		Shell:     history.Bash,
		ExitCode:  w.getExitCode(),
		Duration:  w.getDuration(),
		Timestamp: time.Now(),
		Tags:      []string{"bash", "windows"},
	}

	// Add Windows-specific tags
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		cmdRecord.AddTag("wsl")
	} else if strings.Contains(strings.ToLower(os.Getenv("MSYSTEM")), "mingw") {
		cmdRecord.AddTag("git-bash")
	}

	// Add Bash-specific command pattern tags
	command = strings.ToLower(cmdRecord.Command)
	if strings.Contains(command, "export ") {
		cmdRecord.AddTag("bash-export")
	}
	if strings.Contains(command, "source ") || strings.Contains(command, ". ") {
		cmdRecord.AddTag("bash-source")
	}

	// Generate ID and enhance
	cmdRecord.ID = w.envManager.GenerateCommandID(cmdRecord.Command, cmdRecord.Directory, cmdRecord.Timestamp)

	if err := w.enhanceCommandRecord(cmdRecord); err != nil {
		return fmt.Errorf("failed to enhance Windows Bash command record: %w", err)
	}

	// Store the command
	return w.storage.SaveCommand(*cmdRecord)
}

// getPowerShellLastCommand attempts to get the last executed PowerShell command
func (w *WindowsCapture) getPowerShellLastCommand() string {
	// Try to execute PowerShell to get the last command from history
	cmd := exec.Command("powershell", "-Command", "Get-History -Count 1 | Select-Object -ExpandProperty CommandLine")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// getPowerShellMetadata gets PowerShell-specific metadata
func (w *WindowsCapture) getPowerShellMetadata() map[string]string {
	metadata := make(map[string]string)

	// Get PowerShell version
	if psVersion := os.Getenv("PSVersionTable"); psVersion != "" {
		metadata["ps_version"] = psVersion
	}

	// Get PowerShell edition (Core vs Desktop)
	if psEdition := os.Getenv("PSEdition"); psEdition != "" {
		metadata["ps_edition"] = psEdition
	}

	// Get execution policy
	if execPolicy := os.Getenv("PSExecutionPolicyPreference"); execPolicy != "" {
		metadata["execution_policy"] = execPolicy
	}

	// Detect PowerShell variant
	metadata["ps_variant"] = w.platform.GetPowerShellVariant()

	return metadata
}

// addPowerShellTags adds PowerShell-specific tags to command record
func (w *WindowsCapture) addPowerShellTags(cmdRecord *history.CommandRecord, metadata map[string]string) {
	// Add PowerShell variant tag
	if variant, ok := metadata["ps_variant"]; ok && variant != "" {
		cmdRecord.AddTag(fmt.Sprintf("ps-%s", variant))
	}

	// Add execution policy tag
	if policy, ok := metadata["execution_policy"]; ok && policy != "" {
		cmdRecord.AddTag(fmt.Sprintf("exec-policy-%s", strings.ToLower(policy)))
	}

	// Add PowerShell edition tag
	if edition, ok := metadata["ps_edition"]; ok && edition != "" {
		cmdRecord.AddTag(fmt.Sprintf("ps-%s", strings.ToLower(edition)))
	}

	// Analyze PowerShell-specific command patterns
	command := strings.ToLower(cmdRecord.Command)

	if strings.Contains(command, "get-") || strings.Contains(command, "set-") ||
		strings.Contains(command, "new-") || strings.Contains(command, "remove-") {
		cmdRecord.AddTag("powershell-cmdlet")
	}

	if strings.Contains(command, "invoke-") {
		cmdRecord.AddTag("powershell-invoke")
	}

	if strings.Contains(command, "$") {
		cmdRecord.AddTag("powershell-variable")
	}

	if strings.Contains(command, "|") {
		cmdRecord.AddTag("powershell-pipeline")
	}
}

// getCurrentDirectory gets the current directory with Windows-specific handling
func (w *WindowsCapture) getCurrentDirectory() string {
	// Try environment variable first
	if dir := os.Getenv("CHT_DIRECTORY"); dir != "" {
		return w.platform.NormalizePath(dir)
	}

	// Get current working directory
	if currentDir, err := os.Getwd(); err == nil {
		return w.platform.NormalizePath(currentDir)
	}

	// Fallback to user profile directory
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		return w.platform.NormalizePath(userProfile)
	}

	return "C:\\"
}

// getExitCode gets the exit code with Windows-specific handling
func (w *WindowsCapture) getExitCode() int {
	if exitCodeStr := os.Getenv("CHT_EXIT_CODE"); exitCodeStr != "" {
		if code := w.parseIntSafe(exitCodeStr); code >= 0 {
			return code
		}
	}

	// Try to get ERRORLEVEL (CMD) or $LASTEXITCODE (PowerShell)
	if errorLevel := os.Getenv("ERRORLEVEL"); errorLevel != "" {
		if code := w.parseIntSafe(errorLevel); code >= 0 {
			return code
		}
	}

	return 0 // Default to success
}

// getDuration gets the command duration with Windows-specific handling
func (w *WindowsCapture) getDuration() time.Duration {
	if durationStr := os.Getenv("CHT_DURATION"); durationStr != "" {
		// Try parsing as milliseconds first
		if ms := w.parseIntSafe(durationStr); ms >= 0 {
			return time.Duration(ms) * time.Millisecond
		}

		// Try parsing as Go duration string
		if duration, err := time.ParseDuration(durationStr); err == nil {
			return duration
		}
	}

	return 0
}

// parseIntSafe safely parses an integer string
func (w *WindowsCapture) parseIntSafe(s string) int {
	// Prefer strconv.Atoi for robust parsing and explicit error handling
	if s == "" {
		return -1
	}
	if i, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return i
	}
	return -1
}

// isParentProcessPowerShell checks if the parent process is PowerShell
func (w *WindowsCapture) isParentProcessPowerShell() bool {
	return w.checkParentProcess([]string{"powershell.exe", "pwsh.exe"})
}

// isParentProcessCmd checks if the parent process is cmd.exe
func (w *WindowsCapture) isParentProcessCmd() bool {
	return w.checkParentProcess([]string{"cmd.exe"})
}

// checkParentProcess checks if the parent process matches any of the given names
func (w *WindowsCapture) checkParentProcess(processNames []string) bool {
	// This is a simplified check - in a full implementation, you might use
	// Windows APIs to get the actual parent process information

	// For now, we'll use a heuristic based on environment variables
	// and process information that's typically available

	// Check if we can determine the parent process from environment
	if parentCmd := os.Getenv("COMSPEC"); parentCmd != "" {
		parentName := strings.ToLower(filepath.Base(parentCmd))
		for _, name := range processNames {
			if parentName == strings.ToLower(name) {
				return true
			}
		}
	}

	// If environment heuristics failed, try to get more process info.
	// This references `getProcessInfo()` and uses the returned
	// `Executable` field when available to improve detection.
	if info, err := w.getProcessInfo(); err == nil && info != nil && info.Executable != "" {
		procName := strings.ToLower(filepath.Base(info.Executable))
		for _, name := range processNames {
			if procName == strings.ToLower(name) {
				return true
			}
		}
	}

	return false
}

// GetWindowsShellInfo returns detailed information about Windows shells
func (w *WindowsCapture) GetWindowsShellInfo() (*WindowsShellInfo, error) {
	info := &WindowsShellInfo{
		Platform: "windows",
		Shells:   make(map[string]WindowsShellDetails),
	}

	// Check PowerShell availability
	if psVariant := w.platform.GetPowerShellVariant(); psVariant != "" {
		info.Shells["powershell"] = WindowsShellDetails{
			Available: true,
			Variant:   psVariant,
			Path:      w.getPowerShellPath(psVariant),
		}
	}

	// Check CMD availability
	cmdPath := w.platform.GetCmdPath()
	if _, err := os.Stat(cmdPath); err == nil {
		info.Shells["cmd"] = WindowsShellDetails{
			Available: true,
			Variant:   "cmd",
			Path:      cmdPath,
		}
	}

	// Check Bash availability (Git Bash, WSL)
	if bashPath, err := w.findBashOnWindows(); err == nil {
		info.Shells["bash"] = WindowsShellDetails{
			Available: true,
			Variant:   w.detectBashVariant(),
			Path:      bashPath,
		}
	}

	return info, nil
}

// getPowerShellPath gets the path to the PowerShell executable
func (w *WindowsCapture) getPowerShellPath(variant string) string {
	if variant == "pwsh" {
		if path, err := exec.LookPath("pwsh.exe"); err == nil {
			return path
		}
	}
	if path, err := exec.LookPath("powershell.exe"); err == nil {
		return path
	}
	return ""
}

// findBashOnWindows finds Bash executable on Windows
func (w *WindowsCapture) findBashOnWindows() (string, error) {
	// Try common locations for Bash on Windows
	candidates := []string{
		"bash.exe",                              // In PATH
		"C:\\Program Files\\Git\\bin\\bash.exe", // Git Bash
		"C:\\Windows\\System32\\bash.exe",       // WSL
		"C:\\msys64\\usr\\bin\\bash.exe",        // MSYS2
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("bash not found on Windows")
}

// detectBashVariant detects which Bash variant is being used on Windows
func (w *WindowsCapture) detectBashVariant() string {
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return "wsl"
	}
	if strings.Contains(strings.ToLower(os.Getenv("MSYSTEM")), "mingw") {
		return "git-bash"
	}
	if os.Getenv("MSYS") != "" {
		return "msys2"
	}
	return "unknown"
}

// WindowsShellInfo contains information about available Windows shells
type WindowsShellInfo struct {
	Platform string                         `json:"platform"`
	Shells   map[string]WindowsShellDetails `json:"shells"`
}

// WindowsShellDetails contains details about a specific Windows shell
type WindowsShellDetails struct {
	Available bool   `json:"available"`
	Variant   string `json:"variant"`
	Path      string `json:"path"`
}

// getProcessInfo gets detailed process information (Windows-specific)
func (w *WindowsCapture) getProcessInfo() (*WindowsProcessInfo, error) {
	info := &WindowsProcessInfo{
		PID: uint32(os.Getpid()),
	}

	// Additional process information could be gathered here using Windows APIs
	// This is a placeholder for more advanced process introspection

	return info, nil
}

// WindowsProcessInfo contains Windows-specific process information
type WindowsProcessInfo struct {
	PID        uint32 `json:"pid"`
	ParentPID  uint32 `json:"parent_pid,omitempty"`
	Executable string `json:"executable,omitempty"`
}
