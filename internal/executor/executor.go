package executor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ValGrace/command-history-tracker/internal/errors"
	"github.com/ValGrace/command-history-tracker/pkg/history"
)

// Executor implements the CommandExecutor interface
type Executor struct {
	dangerousPatterns []*regexp.Regexp
	confirmRequired   []*regexp.Regexp
	storage           ExecutionLogger
	validator         *CommandValidator
	auditLogger       *AuditLogger
}

// ExecutionLogger defines interface for logging command executions
type ExecutionLogger interface {
	SaveCommand(cmd history.CommandRecord) error
}

// ExecutionResult contains the result of command execution
type ExecutionResult struct {
	Command   string
	ExitCode  int
	Duration  time.Duration
	Output    string
	Error     error
	Directory string
}

// NewExecutor creates a new command executor with default safety rules
func NewExecutor() *Executor {
	return &Executor{
		dangerousPatterns: compileDangerousPatterns(),
		confirmRequired:   compileConfirmPatterns(),
		validator:         NewCommandValidator(),
	}
}

// NewExecutorWithLogger creates a new command executor with execution logging
func NewExecutorWithLogger(logger ExecutionLogger) *Executor {
	return &Executor{
		dangerousPatterns: compileDangerousPatterns(),
		confirmRequired:   compileConfirmPatterns(),
		storage:           logger,
		validator:         NewCommandValidator(),
	}
}

// NewExecutorWithAudit creates a new command executor with audit logging
func NewExecutorWithAudit(logger ExecutionLogger, auditLogger *AuditLogger) *Executor {
	return &Executor{
		dangerousPatterns: compileDangerousPatterns(),
		confirmRequired:   compileConfirmPatterns(),
		storage:           logger,
		validator:         NewCommandValidator(),
		auditLogger:       auditLogger,
	}
}

// compileDangerousPatterns returns patterns for commands that should be blocked
func compileDangerousPatterns() []*regexp.Regexp {
	patterns := []string{
		`^rm\s+-rf\s+/\s*$`,              // rm -rf /
		`^rm\s+-rf\s+/\*`,                // rm -rf /*
		`^del\s+/[sS]\s+/[qQ]\s+[cC]:\\`, // del /s /q C:\
		`^format\s+[cC]:`,                // format C:
		`^dd\s+if=.*of=/dev/sd`,          // dd to disk devices
		`^mkfs\.`,                        // filesystem formatting
		`^:(){ :|:& };:`,                 // fork bomb
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if re, err := regexp.Compile(pattern); err == nil {
			compiled = append(compiled, re)
		}
	}
	return compiled
}

// compileConfirmPatterns returns patterns for commands that require confirmation
func compileConfirmPatterns() []*regexp.Regexp {
	patterns := []string{
		`^rm\s+-rf`,                // rm -rf
		`^rm\s+-r`,                 // rm -r
		`^del\s+/[sS]`,             // del /s
		`^rmdir\s+/[sS]`,           // rmdir /s
		`^git\s+push\s+.*--force`,  // git push --force
		`^git\s+reset\s+--hard`,    // git reset --hard
		`^docker\s+system\s+prune`, // docker system prune
		`^kubectl\s+delete`,        // kubectl delete
		`^DROP\s+DATABASE`,         // SQL DROP DATABASE
		`^DROP\s+TABLE`,            // SQL DROP TABLE
		`^TRUNCATE`,                // SQL TRUNCATE
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if re, err := regexp.Compile(`(?i)` + pattern); err == nil {
			compiled = append(compiled, re)
		}
	}
	return compiled
}

// ValidateCommand checks if a command is safe to execute
func (e *Executor) ValidateCommand(cmd *history.CommandRecord) error {
	if cmd == nil {
		return errors.NewValidationError("command record cannot be nil", nil)
	}

	if cmd.Command == "" {
		return errors.NewValidationError("command cannot be empty", nil)
	}

	// Check for dangerous patterns that should be blocked
	for _, pattern := range e.dangerousPatterns {
		if pattern.MatchString(cmd.Command) {
			// Log blocked command
			if e.auditLogger != nil {
				if err := e.auditLogger.LogBlocked(cmd.Command, cmd.Directory, cmd.Shell, "dangerous_pattern"); err != nil {
					fmt.Printf("Warning: failed to write audit log (blocked): %v\n", err)
				}
			}

			return errors.NewExecutionError(
				fmt.Sprintf("command blocked for safety: %s", cmd.Command),
				nil,
			).WithContext("command", cmd.Command).WithContext("reason", "dangerous_pattern")
		}
	}

	// Use validator if available
	if e.validator != nil {
		if err := e.validator.Validate(cmd.Command, cmd.Directory); err != nil {
			// Log blocked command
			if e.auditLogger != nil {
					if err := e.auditLogger.LogBlocked(cmd.Command, cmd.Directory, cmd.Shell, "validation_failed"); err != nil {
						fmt.Printf("Warning: failed to write audit log (validation_failed): %v\n", err)
					}
			}
			return err
		}
	}

	return nil
}

// PreviewCommand returns a preview of what will be executed
func (e *Executor) PreviewCommand(cmd *history.CommandRecord) string {
	if cmd == nil {
		return ""
	}

	var preview strings.Builder
	preview.WriteString("Command Preview:\n")
	preview.WriteString("================\n")
	preview.WriteString(fmt.Sprintf("Command:   %s\n", cmd.Command))
	preview.WriteString(fmt.Sprintf("Shell:     %s\n", cmd.Shell.String()))
	preview.WriteString(fmt.Sprintf("Directory: %s\n", cmd.Directory))
	preview.WriteString(fmt.Sprintf("Timestamp: %s\n", cmd.Timestamp.Format("2006-01-02 15:04:05")))

	if cmd.ExitCode != 0 {
		preview.WriteString(fmt.Sprintf("Previous Exit Code: %d (failed)\n", cmd.ExitCode))
	}

	// Check if confirmation is required
	if e.requiresConfirmation(cmd.Command) {
		preview.WriteString("\n⚠️  WARNING: This command may be destructive!\n")
	}

	return preview.String()
}

// ConfirmExecution prompts user for confirmation before execution
func (e *Executor) ConfirmExecution(cmd *history.CommandRecord) (bool, error) {
	if cmd == nil {
		return false, errors.NewValidationError("command record cannot be nil", nil)
	}

	// Validate command first
	if err := e.ValidateCommand(cmd); err != nil {
		return false, err
	}

	// Check if confirmation is required
	if !e.requiresConfirmation(cmd.Command) {
		return true, nil
	}

	// Show preview
	fmt.Println(e.PreviewCommand(cmd))
	fmt.Println()

	// Prompt for confirmation
	fmt.Print("Do you want to execute this command? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, errors.NewExecutionError("failed to read user input", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	confirmed := response == "yes" || response == "y"

	// Log cancellation if not confirmed
	if !confirmed && e.auditLogger != nil {
		if err := e.auditLogger.LogCancelled(cmd, "user_declined"); err != nil {
			fmt.Printf("Warning: failed to write audit log (cancelled): %v\n", err)
		}
	}

	return confirmed, nil
}

// requiresConfirmation checks if a command requires user confirmation
func (e *Executor) requiresConfirmation(command string) bool {
	for _, pattern := range e.confirmRequired {
		if pattern.MatchString(command) {
			return true
		}
	}
	return false
}

// IsDangerous checks if a command matches dangerous patterns
func (e *Executor) IsDangerous(command string) bool {
	for _, pattern := range e.dangerousPatterns {
		if pattern.MatchString(command) {
			return true
		}
	}
	return false
}

// RequiresConfirmation checks if a command requires confirmation
func (e *Executor) RequiresConfirmation(command string) bool {
	return e.requiresConfirmation(command)
}

// ExecuteCommand runs a command in the specified directory context
func (e *Executor) ExecuteCommand(cmd *history.CommandRecord, currentDir string) error {
	if cmd == nil {
		return errors.NewValidationError("command record cannot be nil", nil)
	}

	// Validate command
	if err := e.ValidateCommand(cmd); err != nil {
		return err
	}

	// Get confirmation if needed
	confirmed, err := e.ConfirmExecution(cmd)
	if err != nil {
		return err
	}
	if !confirmed {
		return errors.NewExecutionError("command execution cancelled by user", nil)
	}

	// Execute the command
	result, err := e.executeInContext(cmd.Command, currentDir, cmd.Shell)
	if err != nil {
		return errors.NewExecutionError("command execution failed", err).
			WithContext("command", cmd.Command).
			WithContext("directory", currentDir).
			WithContext("exit_code", result.ExitCode)
	}

	// Log execution if logger is available
	if e.storage != nil {
		executionRecord := history.CommandRecord{
			ID:        generateCommandID(),
			Command:   cmd.Command,
			Directory: currentDir,
			Timestamp: time.Now(),
			Shell:     cmd.Shell,
			ExitCode:  result.ExitCode,
			Duration:  result.Duration,
			Tags:      []string{"executed"},
		}

		if err := e.storage.SaveCommand(executionRecord); err != nil {
			// Log error but don't fail execution
			fmt.Printf("Warning: failed to log command execution: %v\n", err)
		}
	}

	// Log to audit trail
		if e.auditLogger != nil {
			if result.ExitCode == 0 {
				if err := e.auditLogger.LogSuccess(cmd, result.Duration); err != nil {
					fmt.Printf("Warning: failed to write audit log (success): %v\n", err)
				}
			} else {
				if err := e.auditLogger.LogFailure(cmd, result.ExitCode, result.Duration, "command_failed"); err != nil {
					fmt.Printf("Warning: failed to write audit log (failure): %v\n", err)
				}
			}
		}

	return nil
}

// executeInContext executes a command in a specific directory with proper context
func (e *Executor) executeInContext(command string, directory string, shell history.ShellType) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Command:   command,
		Directory: directory,
	}

	startTime := time.Now()

	// Resolve directory path
	absDir, err := filepath.Abs(directory)
	if err != nil {
		result.Error = fmt.Errorf("failed to resolve directory: %w", err)
		return result, result.Error
	}

	// Verify directory exists
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		result.Error = fmt.Errorf("directory does not exist: %s", absDir)
		return result, result.Error
	}

	// Prepare command based on shell type
	var cmd *exec.Cmd
	switch shell {
	case history.PowerShell:
		cmd = exec.Command("powershell", "-NoProfile", "-Command", command)
	case history.Cmd:
		cmd = exec.Command("cmd", "/C", command)
	case history.Bash:
		cmd = exec.Command("bash", "-c", command)
	case history.Zsh:
		cmd = exec.Command("zsh", "-c", command)
	default:
		// Default to system shell
		if isWindows() {
			cmd = exec.Command("cmd", "/C", command)
		} else {
			cmd = exec.Command("sh", "-c", command)
		}
	}

	// Set working directory
	cmd.Dir = absDir

	// Preserve environment variables
	cmd.Env = os.Environ()

	// Connect to standard streams
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Execute command
	execErr := cmd.Run()

	// Calculate duration
	result.Duration = time.Since(startTime)

	// Get exit code
	if execErr != nil {
		if exitErr, ok := execErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
		result.Error = execErr
	} else {
		result.ExitCode = 0
	}

	return result, nil
}

// ExecuteInDirectory executes a command in a specific directory
func (e *Executor) ExecuteInDirectory(command string, directory string, shell history.ShellType) (*ExecutionResult, error) {
	// Create a temporary command record for validation
	tempCmd := &history.CommandRecord{
		Command: command,
		Shell:   shell,
	}

	// Validate command
	if err := e.ValidateCommand(tempCmd); err != nil {
		return nil, err
	}

	// Execute
	return e.executeInContext(command, directory, shell)
}

// isWindows checks if the current platform is Windows
func isWindows() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}

// generateCommandID generates a unique ID for a command record
func generateCommandID() string {
	return fmt.Sprintf("cmd_%d", time.Now().UnixNano())
}

// SetValidator sets the command validator
func (e *Executor) SetValidator(validator *CommandValidator) {
	e.validator = validator
}

// GetValidator returns the command validator
func (e *Executor) GetValidator() *CommandValidator {
	return e.validator
}

// SetAuditLogger sets the audit logger
func (e *Executor) SetAuditLogger(logger *AuditLogger) {
	e.auditLogger = logger
}

// GetAuditLogger returns the audit logger
func (e *Executor) GetAuditLogger() *AuditLogger {
	return e.auditLogger
}

// Close closes any open resources
func (e *Executor) Close() error {
	if e.auditLogger != nil {
		return e.auditLogger.Close()
	}
	return nil
}
