package shell

import (
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Integrator implements ShellIntegrator interface
type Integrator struct {
	detector *Detector
	platform PlatformAbstraction
}

// NewIntegrator creates a new shell integrator
func NewIntegrator() *Integrator {
	return &Integrator{
		detector: NewDetector(),
		platform: NewPlatformAbstraction(),
	}
}

// SetupIntegration configures shell hooks for command capture
func (i *Integrator) SetupIntegration(shell history.ShellType) error {
	if !i.detector.IsShellSupported(shell) {
		return fmt.Errorf("shell %s is not supported on platform %s",
			shell.String(), i.platform.GetPlatform().String())
	}

	script, err := i.GetIntegrationScript(shell)
	if err != nil {
		return fmt.Errorf("failed to get integration script: %w", err)
	}

	return i.installShellHook(shell, script)
}

// RemoveIntegration removes shell hooks
func (i *Integrator) RemoveIntegration(shell history.ShellType) error {
	return i.removeShellHook(shell)
}

// GetIntegrationScript returns the script content for shell integration
func (i *Integrator) GetIntegrationScript(shell history.ShellType) (string, error) {
	switch shell {
	case history.Unknown:
		return "", fmt.Errorf("cannot get integration script for unknown shell type")
	case history.PowerShell:
		return i.getPowerShellScript(), nil
	case history.Bash:
		return i.getBashScript(), nil
	case history.Zsh:
		return i.getZshScript(), nil
	case history.Cmd:
		return i.getCmdScript(), nil
	default:
		return "", fmt.Errorf("unsupported shell type: %s", shell.String())
	}
}

// IsIntegrationActive checks if integration is currently active
func (i *Integrator) IsIntegrationActive(shell history.ShellType) (bool, error) {
	configPath, err := i.getShellConfigPath(shell)
	if err != nil {
		return false, err
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	// Check if our integration marker is present
	marker := i.getIntegrationMarker()
	return strings.Contains(string(content), marker), nil
}

// IsInstalled checks if the tracker is installed for the given shell
// This is an alias for IsIntegrationActive for backward compatibility
func (i *Integrator) IsInstalled(shell history.ShellType) (bool, error) {
	return i.IsIntegrationActive(shell)
}

// getPowerShellScript returns PowerShell integration script
func (i *Integrator) getPowerShellScript() string {
	return `# Command History Tracker Integration
function Invoke-HistoryTracker {
    param([string]$Command, [string]$Directory, [int]$ExitCode, [long]$Duration)
    
    $env:CHT_COMMAND = $Command
    $env:CHT_DIRECTORY = $Directory
    $env:CHT_EXIT_CODE = $ExitCode
    $env:CHT_DURATION = $Duration
    $env:CHT_SHELL = "powershell"
    $env:CHT_TIMESTAMP = (Get-Date).ToString("o")
    
    # Call the tracker executable to record the command
    if (Get-Command "tracker" -ErrorAction SilentlyContinue) {
        & tracker record 2>$null
    }
}

# Override the prompt function to capture commands
$global:OriginalPrompt = $function:prompt
function prompt {
    $lastCommand = Get-History -Count 1 -ErrorAction SilentlyContinue
    if ($lastCommand -and $global:LastCommandId -ne $lastCommand.Id) {
        $global:LastCommandId = $lastCommand.Id
        $duration = if ($lastCommand.EndExecutionTime -and $lastCommand.StartExecutionTime) {
            ($lastCommand.EndExecutionTime - $lastCommand.StartExecutionTime).TotalMilliseconds
        } else { 0 }
        
        Invoke-HistoryTracker -Command $lastCommand.CommandLine -Directory (Get-Location).Path -ExitCode $LASTEXITCODE -Duration $duration
    }
    
    & $global:OriginalPrompt
}`
}

// getBashScript returns Bash integration script
func (i *Integrator) getBashScript() string {
	return `# Command History Tracker Integration
__cht_record_command() {
    local exit_code=$?
    local end_time=$(date +%s%3N)
    local duration=$((end_time - ${__cht_start_time:-$end_time}))
    
    if [[ -n "$__cht_current_command" ]]; then
        export CHT_COMMAND="$__cht_current_command"
        export CHT_DIRECTORY="$PWD"
        export CHT_EXIT_CODE="$exit_code"
        export CHT_DURATION="$duration"
        export CHT_SHELL="bash"
        export CHT_TIMESTAMP=$(date -Iseconds)
        
        # Call the tracker executable to record the command
        if command -v tracker >/dev/null 2>&1; then
            tracker record 2>/dev/null
        fi
        
        unset __cht_current_command
    fi
    
    return $exit_code
}

__cht_preexec() {
    __cht_current_command="$1"
    __cht_start_time=$(date +%s%3N)
}

# Set up command capture hooks
if [[ -z "$__cht_installed" ]]; then
    export __cht_installed=1
    
    # Install preexec hook if available
    if [[ -n "$BASH_VERSION" ]]; then
        # For Bash, we need to use DEBUG trap
        __cht_debug_trap() {
            if [[ "$BASH_COMMAND" != "__cht_record_command" && "$BASH_COMMAND" != *"__cht_"* ]]; then
                __cht_preexec "$BASH_COMMAND"
            fi
        }
        trap '__cht_debug_trap' DEBUG
    fi
    
    # Install prompt command hook
    if [[ -z "$PROMPT_COMMAND" ]]; then
        PROMPT_COMMAND="__cht_record_command"
    else
        PROMPT_COMMAND="__cht_record_command; $PROMPT_COMMAND"
    fi
fi`
}

// getZshScript returns Zsh integration script
func (i *Integrator) getZshScript() string {
	return `# Command History Tracker Integration
__cht_record_command() {
    local exit_code=$?
    local end_time=$(date +%s%3N)
    local duration=$((end_time - ${__cht_start_time:-$end_time}))
    
    if [[ -n "$__cht_current_command" ]]; then
        export CHT_COMMAND="$__cht_current_command"
        export CHT_DIRECTORY="$PWD"
        export CHT_EXIT_CODE="$exit_code"
        export CHT_DURATION="$duration"
        export CHT_SHELL="zsh"
        export CHT_TIMESTAMP=$(date -Iseconds)
        
        # Call the tracker executable to record the command
        if command -v tracker >/dev/null 2>&1; then
            tracker record 2>/dev/null
        fi
        
        unset __cht_current_command
    fi
    
    return $exit_code
}

__cht_preexec() {
    __cht_current_command="$1"
    __cht_start_time=$(date +%s%3N)
}

# Set up command capture hooks
if [[ -z "$__cht_installed" ]]; then
    export __cht_installed=1
    
    # Add hooks to preexec and precmd arrays
    autoload -Uz add-zsh-hook
    add-zsh-hook preexec __cht_preexec
    add-zsh-hook precmd __cht_record_command
fi`
}

// getCmdScript returns Windows Command Prompt integration script
func (i *Integrator) getCmdScript() string {
	return `@echo off
REM Command History Tracker Integration for CMD
REM This creates a wrapper that captures commands

REM Set environment variables for the tracker
set CHT_SHELL=cmd
set CHT_DIRECTORY=%CD%

REM Note: CMD has limited hooking capabilities
REM Full integration requires using DOSKEY macros or wrapper scripts`
}

// installShellHook installs the integration script into the shell configuration
func (i *Integrator) installShellHook(shell history.ShellType, script string) error {
	configPath, err := i.getShellConfigPath(shell)
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Read existing config
	var existingContent []byte
	if _, err := os.Stat(configPath); err == nil {
		existingContent, err = os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read existing config: %w", err)
		}
	}

	// Check if integration is already installed
	marker := i.getIntegrationMarker()
	if strings.Contains(string(existingContent), marker) {
		return nil // Already installed
	}

	// Append integration script
	integrationBlock := fmt.Sprintf("\n%s\n%s\n%s\n",
		marker, script, i.getIntegrationEndMarker())

	newContent := append(existingContent, []byte(integrationBlock)...)

	return os.WriteFile(configPath, newContent, 0644)
}

// removeShellHook removes the integration script from shell configuration
func (i *Integrator) removeShellHook(shell history.ShellType) error {
	configPath, err := i.getShellConfigPath(shell)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to remove
		}
		return err
	}

	// Remove integration block
	marker := i.getIntegrationMarker()
	endMarker := i.getIntegrationEndMarker()

	contentStr := string(content)
	startIdx := strings.Index(contentStr, marker)
	if startIdx == -1 {
		return nil // Not installed
	}

	endIdx := strings.Index(contentStr[startIdx:], endMarker)
	if endIdx == -1 {
		return fmt.Errorf("malformed integration block in config file")
	}

	endIdx += startIdx + len(endMarker)
	newContent := contentStr[:startIdx] + contentStr[endIdx:]

	return os.WriteFile(configPath, []byte(newContent), 0644)
}

// getShellConfigPath returns the configuration file path for the given shell
func (i *Integrator) getShellConfigPath(shell history.ShellType) (string, error) {
	return i.platform.GetShellConfigPath(shell)
}

// getIntegrationMarker returns the marker used to identify integration blocks
func (i *Integrator) getIntegrationMarker() string {
	return "# >>> Command History Tracker Integration >>>"
}

// getIntegrationEndMarker returns the end marker for integration blocks
func (i *Integrator) getIntegrationEndMarker() string {
	return "# <<< Command History Tracker Integration <<<"
}
