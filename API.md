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

**Package**: `command-history-tracker/pkg/history`

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

**Package**: `command-history-tracker/pkg/history`

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

SQLite-based implementation of the StorageEngine interface.

**Package**: `command-history-tracker/internal/storage`

**Constructor**:
```go
func NewSQLiteStorage(storagePath string) (*SQLiteStorage, error)
```

**Example**:
```go
store, err := storage.NewSQLiteStorage("~/.command-history-tracker")
if err != nil {
    log.Fatal(err)
}
defer store.Close()
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
    "command-history-tracker/pkg/history"
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
    "command-history-tracker/pkg/history"
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

## Thread Safety

- **StorageEngine**: Thread-safe for concurrent reads and writes
- **CommandInterceptor**: Not thread-safe, use single instance per process
- **HistoryBrowser**: Not thread-safe, designed for single-threaded UI

## Performance Considerations

1. **Batch Operations**: Use transactions for multiple SaveCommand calls
2. **Indexing**: Storage automatically indexes by directory and timestamp
3. **Cleanup**: Run CleanupOldCommands periodically to maintain performance
4. **Connection Pooling**: SQLiteStorage uses a single connection, suitable for CLI usage

## Versioning

This API follows semantic versioning. Breaking changes will increment the major version.

Current version: 1.0.0

For more information, see the [Integration Guide](INTEGRATION.md).
