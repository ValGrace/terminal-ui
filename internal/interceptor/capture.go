package interceptor

import (
	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"github.com/ValGrace/command-history-tracker/pkg/shell"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// CommandCapture handles the actual capturing and processing of commands
type CommandCapture struct {
	storage    history.StorageEngine
	envManager *shell.EnvironmentManager
	config     *config.Config
	detector   shell.ShellDetector
}

// NewCommandCapture creates a new command capture instance
func NewCommandCapture(storage history.StorageEngine, cfg *config.Config) *CommandCapture {
	return &CommandCapture{
		storage:    storage,
		envManager: shell.NewEnvironmentManager(),
		config:     cfg,
		detector:   shell.NewDetector(),
	}
}

// CaptureCommand captures a command from the current environment
func (c *CommandCapture) CaptureCommand() error {
	// Check if tracking is enabled
	if !c.envManager.IsTrackerEnabled() {
		return nil // Silently skip if tracking is disabled
	}

	// Validate environment has required variables
	if err := c.envManager.ValidateEnvironment(); err != nil {
		return fmt.Errorf("invalid capture environment: %w", err)
	}

	// Extract command record from environment
	cmdRecord, err := c.envManager.GetCommandFromEnvironment()
	if err != nil {
		return fmt.Errorf("failed to extract command from environment: %w", err)
	}

	// Enhance command record with additional metadata and directory context
	if err := c.enhanceCommandRecord(cmdRecord); err != nil {
		return fmt.Errorf("failed to enhance command record: %w", err)
	}

	// Collect additional metadata
	if err := c.collectMetadata(cmdRecord); err != nil {
		return fmt.Errorf("failed to collect metadata: %w", err)
	}

	// Validate the command record
	if err := cmdRecord.Validate(); err != nil {
		return fmt.Errorf("invalid command record: %w", err)
	}

	// Apply filters to determine if command should be recorded
	if c.shouldSkipCommand(cmdRecord) {
		return nil // Skip recording this command
	}

	// Store the command
	if err := c.storage.SaveCommand(*cmdRecord); err != nil {
		return fmt.Errorf("failed to save command: %w", err)
	}

	return nil
}

// CaptureCommandDirect captures a command directly with provided parameters
func (c *CommandCapture) CaptureCommandDirect(command, directory string, shell history.ShellType, exitCode int, duration time.Duration) error {
	// Create command record
	cmdRecord := &history.CommandRecord{
		Command:   command,
		Directory: directory,
		Shell:     shell,
		ExitCode:  exitCode,
		Duration:  duration,
		Timestamp: time.Now(),
		Tags:      []string{},
	}

	// Generate ID
	cmdRecord.ID = c.generateCommandID(cmdRecord)

	// Enhance with additional metadata
	if err := c.enhanceCommandRecord(cmdRecord); err != nil {
		return fmt.Errorf("failed to enhance command record: %w", err)
	}

	// Collect additional metadata
	if err := c.collectMetadata(cmdRecord); err != nil {
		return fmt.Errorf("failed to collect metadata: %w", err)
	}

	// Validate the command record
	if err := cmdRecord.Validate(); err != nil {
		return fmt.Errorf("invalid command record: %w", err)
	}

	// Apply filters
	if c.shouldSkipCommand(cmdRecord) {
		return nil
	}

	// Store the command
	if err := c.storage.SaveCommand(*cmdRecord); err != nil {
		return fmt.Errorf("failed to save command: %w", err)
	}

	return nil
}

// enhanceCommandRecord adds additional metadata to the command record
func (c *CommandCapture) enhanceCommandRecord(cmdRecord *history.CommandRecord) error {
	// Generate ID if not set
	if cmdRecord.ID == "" {
		cmdRecord.ID = c.generateCommandID(cmdRecord)
	}

	// Enhance directory context detection
	if err := c.enhanceDirectoryContext(cmdRecord); err != nil {
		return fmt.Errorf("failed to enhance directory context: %w", err)
	}

	// Add automatic tags based on command content and context
	c.addAutomaticTags(cmdRecord)

	// Set timestamp if not already set
	if cmdRecord.Timestamp.IsZero() {
		cmdRecord.Timestamp = time.Now()
	}

	return nil
}

// enhanceDirectoryContext improves directory context detection and normalization
func (c *CommandCapture) enhanceDirectoryContext(cmdRecord *history.CommandRecord) error {
	// If no directory is set, try to get current working directory
	if cmdRecord.Directory == "" {
		if currentDir, err := os.Getwd(); err == nil {
			cmdRecord.Directory = currentDir
		} else {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Normalize directory path for consistent storage
	normalizedDir, err := c.normalizeDirectory(cmdRecord.Directory)
	if err != nil {
		return fmt.Errorf("failed to normalize directory: %w", err)
	}
	cmdRecord.Directory = normalizedDir

	// Validate directory exists
	if _, err := os.Stat(cmdRecord.Directory); err != nil {
		// Directory might not exist anymore, but we'll still record it
		// Add a tag to indicate this
		cmdRecord.AddTag("directory-missing")
	}

	return nil
}

// collectMetadata gathers additional metadata about the command execution context
func (c *CommandCapture) collectMetadata(cmdRecord *history.CommandRecord) error {
	// Collect system metadata
	if err := c.collectSystemMetadata(cmdRecord); err != nil {
		// Don't fail the entire capture if metadata collection fails
		// Just add a tag to indicate incomplete metadata
		cmdRecord.AddTag("incomplete-metadata")
	}

	// Collect directory-specific metadata
	if err := c.collectDirectoryMetadata(cmdRecord); err != nil {
		// Non-fatal error, just log it
		cmdRecord.AddTag("directory-metadata-error")
	}

	// Collect command-specific metadata
	c.collectCommandMetadata(cmdRecord)

	return nil
}

// collectSystemMetadata gathers system-level metadata
func (c *CommandCapture) collectSystemMetadata(cmdRecord *history.CommandRecord) error {
	// Add system information tags
	cmdRecord.AddTag(fmt.Sprintf("os-%s", runtime.GOOS))
	cmdRecord.AddTag(fmt.Sprintf("arch-%s", runtime.GOARCH))

	// Add shell-specific metadata
	if cmdRecord.Shell != history.Unknown {
		cmdRecord.AddTag(fmt.Sprintf("shell-%s", cmdRecord.Shell.String()))
	}

	// Add timing metadata
	if cmdRecord.Duration > 0 {
		if cmdRecord.Duration > time.Minute {
			cmdRecord.AddTag("long-duration")
		} else if cmdRecord.Duration < 100*time.Millisecond {
			cmdRecord.AddTag("fast-execution")
		}
	}

	return nil
}

// collectDirectoryMetadata gathers directory-specific metadata
func (c *CommandCapture) collectDirectoryMetadata(cmdRecord *history.CommandRecord) error {
	// Check if directory is a project root
	if c.isProjectRoot(cmdRecord.Directory) {
		cmdRecord.AddTag("project-root")

		// Detect project type
		if projectType := c.detectProjectType(cmdRecord.Directory); projectType != "" {
			cmdRecord.AddTag(fmt.Sprintf("project-%s", projectType))
		}
	}

	// Check if directory is under version control
	if c.isUnderVersionControl(cmdRecord.Directory) {
		cmdRecord.AddTag("version-controlled")

		// Detect VCS type
		if vcsType := c.detectVCSType(cmdRecord.Directory); vcsType != "" {
			cmdRecord.AddTag(fmt.Sprintf("vcs-%s", vcsType))
		}
	}

	// Add directory depth information
	depth := c.calculateDirectoryDepth(cmdRecord.Directory)
	if depth > 5 {
		cmdRecord.AddTag("deep-directory")
	}

	return nil
}

// collectCommandMetadata gathers command-specific metadata
func (c *CommandCapture) collectCommandMetadata(cmdRecord *history.CommandRecord) {
	command := strings.TrimSpace(cmdRecord.Command)

	// Analyze command structure
	parts := strings.Fields(command)
	if len(parts) > 0 {
		baseCommand := parts[0]
		cmdRecord.AddTag(fmt.Sprintf("cmd-%s", baseCommand))

		// Check for common command patterns
		if len(parts) > 1 {
			if c.hasFlags(parts[1:]) {
				cmdRecord.AddTag("has-flags")
			}
			if c.hasRedirection(command) {
				cmdRecord.AddTag("has-redirection")
			}
			if c.hasPipes(command) {
				cmdRecord.AddTag("has-pipes")
			}
		}
	}

	// Analyze command complexity
	complexity := c.calculateCommandComplexity(command)
	if complexity > 3 {
		cmdRecord.AddTag("complex-command")
	}
}

// normalizeDirectory normalizes the directory path for consistent storage
func (c *CommandCapture) normalizeDirectory(dir string) (string, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return dir, err // Return original if conversion fails
	}

	// Clean the path (remove redundant elements)
	cleanPath := filepath.Clean(absPath)

	// Convert to forward slashes for consistency across platforms
	normalizedPath := filepath.ToSlash(cleanPath)

	return normalizedPath, nil
}

// addAutomaticTags adds tags based on command content and context
func (c *CommandCapture) addAutomaticTags(cmdRecord *history.CommandRecord) {
	command := cmdRecord.Command

	// Add tags based on command patterns
	if c.isGitCommand(command) {
		cmdRecord.AddTag("git")
	}

	if c.isDockerCommand(command) {
		cmdRecord.AddTag("docker")
	}

	if c.isPackageManagerCommand(command) {
		cmdRecord.AddTag("package-manager")
	}

	if c.isBuildCommand(command) {
		cmdRecord.AddTag("build")
	}

	if c.isTestCommand(command) {
		cmdRecord.AddTag("test")
	}

	// Add tag based on exit code
	if cmdRecord.ExitCode != 0 {
		cmdRecord.AddTag("failed")
	} else {
		cmdRecord.AddTag("success")
	}

	// Add tag based on execution duration
	if cmdRecord.Duration > 10*time.Second {
		cmdRecord.AddTag("long-running")
	}

	// Add tag based on directory context
	if c.isProjectRoot(cmdRecord.Directory) {
		cmdRecord.AddTag("project-root")
	}
}

// shouldSkipCommand determines if a command should be skipped from recording
func (c *CommandCapture) shouldSkipCommand(cmdRecord *history.CommandRecord) bool {
	command := cmdRecord.Command

	// Skip empty commands
	if command == "" {
		return true
	}

	// Skip commands that match exclude patterns from config
	if c.config != nil {
		for _, pattern := range c.config.ExcludePatterns {
			// Try exact match first
			if matched, _ := filepath.Match(pattern, command); matched {
				return true
			}
			// Try prefix match for commands with arguments
			if strings.HasPrefix(command, pattern+" ") || command == pattern {
				return true
			}
		}
	}

	// Skip internal tracker commands to avoid recursion
	if c.isTrackerCommand(command) {
		return true
	}

	// Skip common shell built-ins that don't provide value
	if c.isSkippableBuiltin(command) {
		return true
	}

	return false
}

// generateCommandID creates a unique identifier for the command
func (c *CommandCapture) generateCommandID(cmdRecord *history.CommandRecord) string {
	// Use the environment manager's ID generation
	return c.envManager.GenerateCommandID(
		cmdRecord.Command,
		cmdRecord.Directory,
		cmdRecord.Timestamp,
	)
}

// Helper functions for command classification

func (c *CommandCapture) isGitCommand(command string) bool {
	return len(command) >= 3 && command[:3] == "git"
}

func (c *CommandCapture) isDockerCommand(command string) bool {
	return len(command) >= 6 && command[:6] == "docker"
}

func (c *CommandCapture) isPackageManagerCommand(command string) bool {
	packageManagers := []string{"npm", "yarn", "pip", "go get", "go mod", "cargo", "composer"}
	for _, pm := range packageManagers {
		if len(command) >= len(pm) && command[:len(pm)] == pm {
			return true
		}
	}
	return false
}

func (c *CommandCapture) isBuildCommand(command string) bool {
	buildCommands := []string{"make", "go build", "npm run build", "yarn build", "cargo build"}
	for _, bc := range buildCommands {
		if len(command) >= len(bc) && command[:len(bc)] == bc {
			return true
		}
	}
	return false
}

func (c *CommandCapture) isTestCommand(command string) bool {
	testCommands := []string{"go test", "npm test", "yarn test", "pytest", "cargo test"}
	for _, tc := range testCommands {
		if len(command) >= len(tc) && command[:len(tc)] == tc {
			return true
		}
	}
	return false
}

func (c *CommandCapture) isTrackerCommand(command string) bool {
	trackerCommands := []string{"tracker", "cht"}
	for _, tc := range trackerCommands {
		if len(command) >= len(tc) && command[:len(tc)] == tc {
			return true
		}
	}
	return false
}

func (c *CommandCapture) isSkippableBuiltin(command string) bool {
	builtins := []string{"cd", "pwd", "ls", "dir", "echo", "exit", "clear", "cls"}
	for _, builtin := range builtins {
		if command == builtin {
			return true
		}
	}
	return false
}

func (c *CommandCapture) isProjectRoot(directory string) bool {
	// Check for common project root indicators
	indicators := []string{".git", "go.mod", "package.json", "Cargo.toml", "requirements.txt", "Makefile", "pom.xml", "build.gradle", "composer.json"}

	for _, indicator := range indicators {
		indicatorPath := filepath.Join(directory, indicator)
		if _, err := os.Stat(indicatorPath); err == nil {
			return true
		}
	}

	return false
}

// detectProjectType determines the type of project based on files in the directory
func (c *CommandCapture) detectProjectType(directory string) string {
	projectIndicators := map[string]string{
		"go.mod":           "go",
		"package.json":     "nodejs",
		"Cargo.toml":       "rust",
		"requirements.txt": "python",
		"setup.py":         "python",
		"pom.xml":          "java",
		"build.gradle":     "java",
		"composer.json":    "php",
		"Gemfile":          "ruby",
		"mix.exs":          "elixir",
	}

	for file, projectType := range projectIndicators {
		if _, err := os.Stat(filepath.Join(directory, file)); err == nil {
			return projectType
		}
	}

	return ""
}

// isUnderVersionControl checks if the directory is under version control
func (c *CommandCapture) isUnderVersionControl(directory string) bool {
	vcsIndicators := []string{".git", ".svn", ".hg", ".bzr"}

	// Check current directory and parent directories
	currentDir := directory
	for {
		for _, indicator := range vcsIndicators {
			vcsPath := filepath.Join(currentDir, indicator)
			if _, err := os.Stat(vcsPath); err == nil {
				return true
			}
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break // Reached root directory
		}
		currentDir = parentDir
	}

	return false
}

// detectVCSType determines the type of version control system
func (c *CommandCapture) detectVCSType(directory string) string {
	vcsTypes := map[string]string{
		".git": "git",
		".svn": "svn",
		".hg":  "mercurial",
		".bzr": "bazaar",
	}

	// Check current directory and parent directories
	currentDir := directory
	for {
		for indicator, vcsType := range vcsTypes {
			vcsPath := filepath.Join(currentDir, indicator)
			if _, err := os.Stat(vcsPath); err == nil {
				return vcsType
			}
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break // Reached root directory
		}
		currentDir = parentDir
	}

	return ""
}

// calculateDirectoryDepth calculates the depth of the directory path
func (c *CommandCapture) calculateDirectoryDepth(directory string) int {
	cleanPath := filepath.Clean(directory)
	if cleanPath == "/" || cleanPath == "." {
		return 0
	}

	// Count path separators
	return strings.Count(cleanPath, string(filepath.Separator))
}

// hasFlags checks if command arguments contain flags
func (c *CommandCapture) hasFlags(args []string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			return true
		}
	}
	return false
}

// hasRedirection checks if command contains redirection operators
func (c *CommandCapture) hasRedirection(command string) bool {
	redirectionOperators := []string{">", ">>", "<", "2>", "2>>", "&>"}
	for _, op := range redirectionOperators {
		if strings.Contains(command, op) {
			return true
		}
	}
	return false
}

// hasPipes checks if command contains pipe operators
func (c *CommandCapture) hasPipes(command string) bool {
	return strings.Contains(command, "|")
}

// calculateCommandComplexity estimates command complexity based on various factors
func (c *CommandCapture) calculateCommandComplexity(command string) int {
	complexity := 0

	// Base complexity for any command
	complexity++

	// Add complexity for length
	if len(command) > 50 {
		complexity++
	}
	if len(command) > 100 {
		complexity++
	}

	// Add complexity for special characters and operators
	if c.hasPipes(command) {
		complexity++
	}
	if c.hasRedirection(command) {
		complexity++
	}
	if strings.Contains(command, "&&") || strings.Contains(command, "||") {
		complexity++
	}
	if strings.Contains(command, ";") {
		complexity++
	}

	// Add complexity for quotes (indicating complex arguments)
	if strings.Contains(command, "\"") || strings.Contains(command, "'") {
		complexity++
	}

	// Add complexity for variable substitution
	if strings.Contains(command, "$") {
		complexity++
	}

	return complexity
}

// GetCaptureStats returns statistics about captured commands
func (c *CommandCapture) GetCaptureStats() (*CaptureStats, error) {
	// Get directories with history
	directories, err := c.storage.GetDirectoriesWithHistory()
	if err != nil {
		return nil, fmt.Errorf("failed to get directories: %w", err)
	}

	stats := &CaptureStats{
		TotalDirectories: len(directories),
		TotalCommands:    0,
	}

	// Count total commands across all directories
	for _, dir := range directories {
		commands, err := c.storage.GetCommandsByDirectory(dir)
		if err != nil {
			continue // Skip directories with errors
		}
		stats.TotalCommands += len(commands)
	}

	return stats, nil
}

// CaptureStats represents statistics about command capture
type CaptureStats struct {
	TotalDirectories int `json:"total_directories"`
	TotalCommands    int `json:"total_commands"`
}
