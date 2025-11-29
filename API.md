# API Documentation

This document provides comprehensive API documentation for the Command History Tracker package.

## Table of Contents

- [Core Interfaces](#core-interfaces)
- [Data Structures](#data-structures)
- [Storage Package](#storage-package)
- [Shell Package](#shell-package)
- [Error Handling](#error-handling)

## Core Interfaces

### StorageEngine

The `StorageEngine` interface defines methods for persisting and retrieving command history.

**Package**: `github.com/ValGrace/command-history-tracker/pkg/history`

```go
type StorageEngine interface {
    // SaveCommand persists a command record to storage
    SaveCommand(cmd CommandRecord) error
    
    // GetCommandsByDirectory retrieves all commands executed in a specific directory
    // Commands are returned in reverse chronological order (most recent first)
    GetCommandsByDirectory(dir string) ([]CommandRecord, error)
    
    // GetDirectoriesWithHistory returns a list of all directories that have command history
    GetDirectoriesWithHistory() ([]string, error)
    
    // CleanupOldCommands removes commands older than the specified retention period
    CleanupOldCommands(retentionDays int) error
    
    // Close releases any resources held by the storage engine
    Close() error
}
```

### CommandInterceptor

The `CommandInterceptor` interface handles command capture from shell environments.

**Package**: `command-history-tracker/internal/interceptor`

```go
type CommandInterceptor interface {
    // StartRecording begins capturing commands from the shell
    StartRecording() error
    
    // StopRecording stops capturing commands
    StopRecording() error
    
    // SetupShellIntegration configures shell-specific hooks for command capture
    SetupShellIntegration(shell ShellType) error
}
```

### HistoryBrowser

The `HistoryBrowser` interface provides interactive command browsing capabilities.

**Package**: `command-history-tracker/internal/browser`

```go
type HistoryBrowser interface {
    // ShowDirectoryHistory displays commands for a specific directory
    ShowDirectoryHistory(dir string) error
    
    // ShowDirectoryTree displays the directory tree with command counts
    ShowDirectoryTree() error
    
    // SelectCommand allows the user to select a command from history
    SelectCommand() (*CommandRecord, error)
    
    // FilterCommands filters displayed commands by a pattern
    FilterCommands(pattern string) error
}
```

**Directory Navigation Features**:

The browser supports hierarchical directory navigation with the following capabilities:

- **Tree Organization**: Directories are organized hierarchically with automatic parent expansion
- **Breadcrumb Navigation**: Visual breadcrumbs show the current path with navigation hints
- **Keyboard Navigation**: 
  - Arrow keys (↑/↓) for tree navigation
  - Right arrow (→) to expand/collapse directories
  - Left arrow (←) to navigate to parent directory
  - Enter to select and view directory history
- **Visual Indicators**: 
  - Current directory highlighting
  - Inactive directory dimming (for old/unused directories)
  - Command count display per directory
  - Indentation levels for hierarchy visualization
- **Cross-Platform Support**: Works with both Unix-style (`/home/user`) and Windows-style (`C:\Users\test`) paths

**Filtering Features**:

The browser provides powerful filtering capabilities:

- **Text Filtering**: Search commands by text pattern (case-insensitive substring matching)
- **Date Range Filtering**: Filter by execution time with presets (Today, Yesterday, This Week, Last Week, This Month, Last Month)
- **Shell Type Filtering**: Filter by shell type (PowerShell, Bash, Zsh, Cmd)
- **Combined Filtering**: Apply multiple filters simultaneously with AND logic
- **Real-time Updates**: Filters apply instantly as you type or change settings
- **Filter State Management**: Filters persist across view changes and can be cleared with a single command

**Filtering Keyboard Shortcuts**:
- `/` - Enter search mode for text filtering
- `f` - Toggle filter panel visibility
- `d` - Toggle date filter (when filter panel is visible)
- `s` - Cycle through shell type filters
- `1-6` - Select date preset (when date filter is enabled)
- `c` - Clear all active filters
- `Escape` - Cancel search mode
- `Enter` - Apply search filter

### CommandExecutor

The `CommandExecutor` interface provides safe command execution with validation and confirmation.

**Package**: `command-history-tracker/internal/executor`

```go
type CommandExecutor interface {
    // ValidateCommand checks if a command is safe to execute
    // Returns an error if the command matches dangerous patterns
    ValidateCommand(cmd *CommandRecord) error
    
    // PreviewCommand returns a formatted preview of the command
    PreviewCommand(cmd *CommandRecord) string
    
    // ConfirmExecution prompts the user for confirmation before execution
    // Returns true if the user confirms, false otherwise
    ConfirmExecution(cmd *CommandRecord) (bool, error)
}
```

**Safety Features**:
- Blocks dangerous commands (e.g., `rm -rf /`, `format C:`, fork bombs)
- Requires confirmation for destructive operations (e.g., `rm -rf`, `git push --force`)
- Provides command preview with context before execution
- Case-insensitive pattern matching for safety checks

## Data Structures

### CommandRecord

Represents a single command execution record.

**Package**: `github.com/ValGrace/command-history-tracker/pkg/history`

```go
type CommandRecord struct {
    ID        string        // Unique identifier
    Command   string        // Full command text
    Directory string        // Working directory
    Timestamp time.Time     // Execution time
    Shell     ShellType     // Shell type
    ExitCode  int           // Exit code
    Duration  time.Duration // Execution duration
}
```

### Config

Application configuration structure.

**Package**: `command-history-tracker/internal/config`

```go
type Config struct {
    StoragePath     string      // Storage directory path
    RetentionDays   int         // Days to retain history
    MaxCommands     int         // Max commands per directory
    EnabledShells   []ShellType // Enabled shell types
    ExcludePatterns []string    // Command exclusion patterns
    AutoCleanup     bool        // Enable automatic cleanup
}
```

**Methods**:
- `LoadConfig() (*Config, error)` - Load configuration from default location
- `SaveConfig() error` - Save configuration to default location
- `DefaultConfig() *Config` - Get default configuration
- `Validate() error` - Validate configuration

### ShellType

Enumeration of supported shell types.

**Package**: `command-history-tracker/pkg/shell`

```go
type ShellType int

const (
    Unknown ShellType = iota
    PowerShell
    Bash
    Zsh
    Cmd
)
```

## Storage Package

### SQLiteStorage

SQLite-based implementation of the StorageEngine interface with advanced filtering capabilities.

**Package**: `command-history-tracker/internal/storage`

**Constructor**:
```go
func NewSQLiteStorage(storagePath string) (*SQLiteStorage, error)
```

**Extended Filtering Methods**:

```go
// GetCommandsByTimeRange retrieves commands within a specific time range
func (s *SQLiteStorage) GetCommandsByTimeRange(startTime, endTime time.Time, dir string) ([]CommandRecord, error)

// GetCommandsByShell retrieves commands filtered by shell type
func (s *SQLiteStorage) GetCommandsByShell(shellType ShellType, dir string) ([]CommandRecord, error)

// FilterCommands retrieves commands with multiple filter criteria
func (s *SQLiteStorage) FilterCommands(filters CommandFilters) ([]CommandRecord, error)
```

**CommandFilters Structure**:

```go
type CommandFilters struct {
    Directory string        // Filter by directory (empty for all directories)
    Pattern   string        // Text pattern to search for
    ShellType ShellType     // Filter by shell type (Unknown for all shells)
    StartTime time.Time     // Start of time range (zero for no start limit)
    EndTime   time.Time     // End of time range (zero for no end limit)
    ExitCode  *int          // Filter by exit code (nil for all exit codes)
    Limit     int           // Maximum number of results (0 for no limit)
}
```

**Example - Basic Storage**:
```go
store, err := storage.NewSQLiteStorage("~/.command-history-tracker")
if err != nil {
    log.Fatal(err)
}
defer store.Close()
```

**Example - Text Search**:
```go
// Search for git commands in a specific directory
results, err := store.SearchCommands("git", "/home/user/project")
if err != nil {
    log.Fatal(err)
}
```

**Example - Shell Filtering**:
```go
// Get all Bash commands from a directory
bashCommands, err := store.GetCommandsByShell(history.Bash, "/home/user/project")
if err != nil {
    log.Fatal(err)
}
```

**Example - Date Range Filtering**:
```go
// Get commands from the last 24 hours
now := time.Now()
yesterday := now.Add(-24 * time.Hour)
recentCommands, err := store.GetCommandsByTimeRange(yesterday, now, "/home/user/project")
if err != nil {
    log.Fatal(err)
}
```

**Example - Combined Filtering**:
```go
// Find all git commands executed in Bash within the last week
filters := storage.CommandFilters{
    Directory: "/home/user/project",
    Pattern:   "git",
    ShellType: history.Bash,
    StartTime: time.Now().Add(-7 * 24 * time.Hour),
    EndTime:   time.Now(),
    Limit:     50,
}

results, err := store.FilterCommands(filters)
if err != nil {
    log.Fatal(err)
}

for _, cmd := range results {
    fmt.Printf("%s: %s\n", cmd.Timestamp.Format("2006-01-02 15:04"), cmd.Command)
}
```

## Shell Package

### ShellDetector

Detects the current shell environment.

**Package**: `command-history-tracker/pkg/shell`

```go
type ShellDetector interface {
    // DetectShell identifies the current shell type
    DetectShell() (ShellType, error)
    
    // GetShellPath returns the path to the shell executable
    GetShellPath(shell ShellType) (string, error)
}
```

### ShellIntegrator

Manages shell integration for command capture.

**Package**: `command-history-tracker/pkg/shell`

```go
type ShellIntegrator interface {
    // InstallHooks installs shell hooks for command capture
    InstallHooks(shell ShellType) error
    
    // RemoveHooks removes shell hooks
    RemoveHooks(shell ShellType) error
    
    // IsInstalled checks if hooks are installed
    IsInstalled(shell ShellType) (bool, error)
}
```

## Error Handling

### Error Types

**Package**: `command-history-tracker/internal/errors`

```go
// ErrNotFound indicates a resource was not found
var ErrNotFound = errors.New("not found")

// ErrInvalidConfig indicates invalid configuration
var ErrInvalidConfig = errors.New("invalid configuration")

// ErrStorageUnavailable indicates storage is unavailable
var ErrStorageUnavailable = errors.New("storage unavailable")

// ErrShellNotSupported indicates unsupported shell
var ErrShellNotSupported = errors.New("shell not supported")
```

### Error Handling Best Practices

1. Always check errors returned by API methods
2. Use appropriate error types for different failure scenarios
3. Provide context when wrapping errors
4. Handle storage errors gracefully with fallback mechanisms

**Example**:
```go
store, err := storage.NewSQLiteStorage(path)
if err != nil {
    if errors.Is(err, errors.ErrStorageUnavailable) {
        // Handle storage unavailable
        log.Println("Storage unavailable, using fallback")
    } else {
        // Handle other errors
        log.Fatal(err)
    }
}
```

## Executor Package

### Executor

Safe command execution with built-in validation and confirmation.

**Package**: `command-history-tracker/internal/executor`

**Constructor**:
```go
func NewExecutor() *Executor
```

**Additional Methods**:
```go
// IsDangerous checks if a command matches dangerous patterns
func (e *Executor) IsDangerous(command string) bool

// RequiresConfirmation checks if a command requires user confirmation
func (e *Executor) RequiresConfirmation(command string) bool
```

**Dangerous Patterns** (automatically blocked):
- `rm -rf /` - Root directory deletion
- `rm -rf /*` - All files deletion
- `del /s /q C:\` - Windows system drive deletion
- `format C:` - Disk formatting
- `dd if=...of=/dev/sd*` - Direct disk writes
- `mkfs.*` - Filesystem formatting
- `:(){ :|:& };:` - Fork bomb

**Confirmation Required Patterns**:
- `rm -rf` / `rm -r` - Recursive deletion
- `del /s` / `rmdir /s` - Windows recursive deletion
- `git push --force` - Force push
- `git reset --hard` - Hard reset
- `docker system prune` - Docker cleanup
- `kubectl delete` - Kubernetes deletion
- `DROP DATABASE` / `DROP TABLE` - SQL drops
- `TRUNCATE` - SQL truncate

**Example**:
```go
executor := executor.NewExecutor()

// Validate command safety
if err := executor.ValidateCommand(cmd); err != nil {
    log.Printf("Command blocked: %v", err)
    return
}

// Check if confirmation is needed
if executor.RequiresConfirmation(cmd.Command) {
    confirmed, err := executor.ConfirmExecution(cmd)
    if err != nil || !confirmed {
        log.Println("Command execution cancelled")
        return
    }
}

// Safe to execute
```

## Usage Examples

### Basic Command Recording

```go
package main

import (
    "log"
    "time"
    
    "command-history-tracker/internal/storage"
    "github.com/ValGrace/command-history-tracker/pkg/history"
)

func main() {
    store, err := storage.NewSQLiteStorage("~/.command-history-tracker")
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    record := history.CommandRecord{
        Command:   "go test ./...",
        Directory: "/home/user/project",
        Timestamp: time.Now(),
        Shell:     history.Bash,
        ExitCode:  0,
        Duration:  time.Second * 5,
    }
    
    if err := store.SaveCommand(record); err != nil {
        log.Fatal(err)
    }
}
```

### Querying History

```go
package main

import (
    "fmt"
    "log"
    
    "command-history-tracker/internal/storage"
)

func main() {
    store, err := storage.NewSQLiteStorage("~/.command-history-tracker")
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    commands, err := store.GetCommandsByDirectory("/home/user/project")
    if err != nil {
        log.Fatal(err)
    }
    
    for _, cmd := range commands {
        fmt.Printf("%s: %s\n", cmd.Timestamp.Format("15:04:05"), cmd.Command)
    }
}
```

### Shell Detection

```go
package main

import (
    "fmt"
    "log"
    
    "command-history-tracker/pkg/shell"
)

func main() {
    detector := shell.NewDetector()
    
    shellType, err := detector.DetectShell()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Detected shell: %s\n", shellType)
}
```

### Safe Command Execution

```go
package main

import (
    "fmt"
    "log"
    
    "command-history-tracker/internal/executor"
    "command-history-tracker/internal/storage"
    "github.com/ValGrace/command-history-tracker/pkg/history"
)

func main() {
    // Load command from history
    store, err := storage.NewSQLiteStorage("~/.command-history-tracker")
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    commands, err := store.GetCommandsByDirectory("/home/user/project")
    if err != nil {
        log.Fatal(err)
    }
    
    if len(commands) == 0 {
        fmt.Println("No commands found")
        return
    }
    
    // Create executor with safety checks
    exec := executor.NewExecutor()
    
    // Select first command
    cmd := &commands[0]
    
    // Validate command safety
    if err := exec.ValidateCommand(cmd); err != nil {
        log.Printf("Command validation failed: %v", err)
        return
    }
    
    // Show preview
    fmt.Println(exec.PreviewCommand(cmd))
    
    // Get confirmation if needed
    confirmed, err := exec.ConfirmExecution(cmd)
    if err != nil {
        log.Printf("Confirmation failed: %v", err)
        return
    }
    
    if !confirmed {
        fmt.Println("Command execution cancelled by user")
        return
    }
    
    fmt.Println("Command confirmed and ready for execution")
    // Execute command here...
}
```

## CLI Command Parameters

The CLI supports various parameter combinations for flexible command history management:

### History Command Flags

```bash
# View history with multiple filters
tracker history --dir /project --limit 10 --since 6h --shell bash

# Non-interactive mode for scripting
tracker history --no-interactive
```

**Available Flags**:
- `--dir`: Filter by specific directory
- `--limit`: Limit number of results
- `--since`: Show commands since time period (e.g., "6h", "2d", "1w")
- `--shell`: Filter by shell type (bash, zsh, powershell, cmd)
- `--no-interactive`: Disable interactive mode for scripting

### Search Command Flags

```bash
# Search across all directories
tracker search "git commit" --all-dirs --limit 20

# Case-sensitive search
tracker search "MyCommand" --case-sensitive
```

**Available Flags**:
- `--all-dirs`: Search across all directories (not just current)
- `--case-sensitive`: Enable case-sensitive matching
- `--limit`: Limit number of results
- `--no-interactive`: Disable interactive mode

### Browse Command Flags

```bash
# Browse with tree view
tracker browse --tree

# Browse specific directory
tracker browse --dir /home/user/project
```

**Available Flags**:
- `--tree`: Display directory tree view
- `--dir`: Browse specific directory

### Command Chaining

The CLI supports executing multiple operations in sequence:

```bash
# Record, then search
tracker record --command "test-cmd" --directory /test
tracker search "test-cmd"

# Configure, then verify
tracker config set retention-days 200
tracker config get retention-days

# Record, browse, cleanup workflow
tracker record --command "old-cmd" --directory /test
tracker browse --dir /test
tracker cleanup --days 2
```

## Configuration Management

### Configuration Workflows

The configuration system supports complete lifecycle management:

```go
// Create default configuration
cfg := config.DefaultConfig()
cfg.Save()

// Load and modify
loadedCfg, _ := config.Load()
loadedCfg.RetentionDays = 365
loadedCfg.MaxCommands = 50000
loadedCfg.Save()

// Validate configuration
if err := cfg.Validate(); err != nil {
    log.Printf("Invalid configuration: %v", err)
}

// Reset to defaults
defaultCfg := config.DefaultConfig()
defaultCfg.Save()
```

### Configuration Validation

The configuration system validates all settings:

**Valid Ranges**:
- `RetentionDays`: 1-365 days
- `MaxCommands`: 100-100000 commands
- `StoragePath`: Non-empty string
- `EnabledShells`: Valid shell types only

**Example Validation**:
```go
cfg := config.DefaultConfig()
cfg.RetentionDays = -1  // Invalid

if err := cfg.Validate(); err != nil {
    // Error: RetentionDays cannot be negative
}
```

## Error Handling Scenarios

### Comprehensive Error Coverage

The API handles various error scenarios gracefully:

**Missing Configuration**:
```go
cfg, err := config.Load()
if err != nil {
    // Falls back to default configuration
    cfg = config.DefaultConfig()
}
```

**Invalid Database Path**:
```go
store, err := storage.NewStorageEngine("sqlite", "/invalid/path/db")
if err != nil {
    log.Printf("Storage initialization failed: %v", err)
    // Handle error appropriately
}
```

**Nonexistent Directory Search**:
```go
results, err := store.SearchCommands("test", "/nonexistent")
// Returns empty results, no error
if len(results) == 0 {
    log.Println("No commands found")
}
```

**Concurrent Storage Access**:
```go
// Storage engine handles concurrent access safely
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        record := history.CommandRecord{
            Command: fmt.Sprintf("cmd-%d", id),
            // ... other fields
        }
        store.SaveCommand(record)
    }(i)
}
wg.Wait()
```

## Thread Safety

- **StorageEngine**: Thread-safe for concurrent reads and writes with SQLite WAL mode
- **CommandInterceptor**: Not thread-safe, use single instance per process
- **HistoryBrowser**: Not thread-safe, designed for single-threaded UI
- **Config**: Thread-safe for reads, serialize writes through Save()

## Performance Considerations

1. **Batch Operations**: Use transactions for multiple SaveCommand calls
2. **Indexing**: Storage automatically indexes by directory and timestamp
3. **Cleanup**: Run CleanupOldCommands periodically to maintain performance
4. **Connection Pooling**: SQLiteStorage uses connection pooling with configurable limits
5. **Concurrent Access**: SQLite WAL mode enables concurrent reads during writes
6. **Query Optimization**: Use filters (directory, time range, shell) to reduce result sets

## Testing

The package includes comprehensive test coverage:

- **Unit Tests**: Component-level testing for all packages
- **Integration Tests**: End-to-end workflow testing
- **CLI Tests**: Command-line interface parameter combinations
- **Error Handling Tests**: Comprehensive error scenario coverage
- **Concurrency Tests**: Thread-safety validation
- **Cross-Platform Tests**: Windows, macOS, and Linux compatibility

**Running Tests**:
```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test ./cmd/tracker/...

# Verbose output
go test -v ./...
```

## Versioning

This API follows semantic versioning. Breaking changes will increment the major version.

Current version: 1.0.0

For more information, see the [Integration Guide](INTEGRATION.md).
