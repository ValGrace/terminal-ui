# Command History Tracker

A Go package that provides comprehensive terminal command history tracking and management capabilities. The system automatically records all terminal commands executed across different directories and provides an interactive CLI interface for browsing, searching, and re-executing commands from history.

## Features

- **Automatic Command Recording**: Captures all terminal commands with directory context and timestamps
- **Cross-Platform Support**: Works with PowerShell, Bash, Zsh, and Cmd across Windows, macOS, and Linux
- **Interactive History Browser**: Terminal-based UI for navigating command history by directory
- **Hierarchical Directory Navigation**: 
  - Tree view with expand/collapse functionality
  - Breadcrumb navigation with parent directory support
  - Visual indicators for current and inactive directories
  - Keyboard shortcuts for efficient navigation
- **Advanced Filtering System**:
  - Text pattern search with real-time updates
  - Date range filtering with presets (Today, Yesterday, This Week, etc.)
  - Shell type filtering (PowerShell, Bash, Zsh, Cmd)
  - Combined multi-criteria filtering with AND logic
  - Storage-level optimized queries for large datasets
- **Cross-Directory Navigation**: Browse and execute commands from any directory in your project structure
- **Safe Command Execution**: Multi-layer safety system with validation, dangerous command blocking, and confirmation prompts
- **Efficient Storage**: SQLite-based storage with retention policies and cleanup mechanisms

## Installation

### Quick Install (Recommended)

**Linux/macOS:**
```bash
./install.sh
```

**Windows (PowerShell):**
```powershell
.\install.ps1
```

### Using go install

```bash
go install command-history-tracker/cmd/tracker@latest
```

### Building from source

```bash
git clone <repository-url>
cd command-history-tracker
make build
make install
```

### Post-Installation Setup

After installation, run the setup wizard to configure shell integration:

```bash
tracker setup
```

This will automatically configure your shell profile for command tracking.

## Quick Start

1. **Set up shell integration**:
   ```bash
   tracker setup
   ```

2. **Start recording commands** (automatic after setup):
   ```bash
   # Just use your terminal normally - commands are automatically recorded
   cd my-project
   git status
   npm test
   ```

3. **Browse command history**:
   ```bash
   tracker browse
   ```

4. **View history for current directory**:
   ```bash
   tracker history
   ```

5. **Search across all commands**:
   ```bash
   tracker search "git commit"
   ```

6. **Check tracker status**:
   ```bash
   tracker status
   ```

## Project Structure

```
command-history-tracker/
├── cmd/                    # CLI application entry points
│   └── tracker/           # Main CLI application
├── internal/              # Private application code
│   ├── interceptor/       # Command capture implementation
│   ├── storage/           # SQLite storage engine
│   ├── browser/           # Terminal UI for history browsing
│   ├── executor/          # Command execution engine
│   └── config/            # Configuration management
├── pkg/                   # Public library interfaces
│   ├── history/           # Core history tracking interfaces
│   └── shell/             # Shell integration utilities
├── go.mod                 # Go module definition
├── Makefile              # Build automation
└── README.md             # Project documentation
```

## Development

### Prerequisites

- Go 1.21 or later
- Make (optional, for using Makefile)

### Building

```bash
# Build the application
make build

# Run tests
make test

# Format code and run checks
make check

# Install locally
make install
```

### Testing

The project includes comprehensive test coverage:

- **Unit Tests**: Component-level testing for all packages
- **Integration Tests**: End-to-end workflow testing
- **CLI Tests**: Command-line interface with parameter combinations and command chaining
- **UI Comprehensive Tests**: Complete terminal UI interaction testing including:
  - Keyboard navigation (arrow keys, vim-style keys, page navigation)
  - Command display and selection across different shells and exit codes
  - Directory navigation flows (parent navigation, tree expansion/collapse)
  - View mode switching (history view ↔ tree view)
  - Breadcrumb updates and boundary conditions
  - Refresh and quit functionality
- **Directory Navigation Tests**: Hierarchical tree organization, breadcrumb navigation, and keyboard interaction testing
- **Filtering Tests**: Complete filtering workflow with text, date, and shell filters
- **Error Handling Tests**: Comprehensive error scenario coverage including missing configs, invalid paths, and concurrent access
- **Configuration Tests**: Complete configuration lifecycle management and validation
- **Cross-Platform Tests**: Shell detection and path handling across Windows, macOS, and Linux

Run specific test suites:
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run CLI tests specifically
go test ./cmd/tracker/...

# Run browser tests
go test ./internal/browser/...

# Run UI comprehensive tests
go test ./internal/browser -v -run Comprehensive

# Run filtering tests
go test ./internal/browser -v -run Filter
go test ./internal/storage -v -run Filter

# Run with verbose output
go test -v ./...

# Run integration tests only
go test -tags=integration ./...
```

**Test Coverage Areas**:
- CLI command parameters and flag combinations
- Command chaining workflows (record → search → cleanup)
- Configuration management (set → get → validate → reset)
- UI keyboard navigation (arrow keys, vim keys, page scrolling, boundary conditions)
- Command display formatting (shell indicators, exit codes, selection states)
- Directory navigation (parent/child navigation, tree expansion, breadcrumbs)
- View mode transitions (history ↔ tree view switching)
- Filtering workflows (text + date + shell combinations)
- Storage-level filtering with optimized queries
- Error handling scenarios (missing files, invalid paths, concurrent access)
- Storage operations (save, retrieve, search, cleanup)
- Shell integration and detection
- Safe command execution with validation

### Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/charmbracelet/bubbletea` - Terminal UI
- `modernc.org/sqlite` - SQLite driver

## Configuration

The application stores its configuration in `~/.command-history-tracker/config.json`. Default configuration includes:

- **Storage Path**: `~/.command-history-tracker`
- **Retention**: 90 days
- **Max Commands**: 10,000 per directory
- **Enabled Shells**: PowerShell, Bash, Zsh, Cmd
- **Auto Cleanup**: Enabled

### Customizing Configuration

Edit the configuration file directly or use the CLI:

```bash
# View current configuration
tracker config show

# Set retention period
tracker config set retention-days 180

# Set max commands per directory
tracker config set max-commands 50000
```

## Integration Examples

### Using as a Go Library

You can integrate the command history tracker into your own Go applications:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/ValGrace/command-history-tracker/pkg/history"
    "command-history-tracker/internal/storage"
    "command-history-tracker/internal/config"
)

func main() {
    // Load configuration
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatal(err)
    }
    
    // Initialize storage
    store, err := storage.NewSQLiteStorage(cfg.StoragePath)
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Query command history
    commands, err := store.GetCommandsByDirectory("/home/user/project")
    if err != nil {
        log.Fatal(err)
    }
    
    // Display commands
    for _, cmd := range commands {
        fmt.Printf("%s: %s\n", cmd.Timestamp.Format("15:04:05"), cmd.Command)
    }
}
```

### Custom Command Recording

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
    
    // Record a custom command
    record := history.CommandRecord{
        Command:   "custom-script.sh",
        Directory: "/home/user/project",
        Timestamp: time.Now(),
        Shell:     history.Bash,
        ExitCode:  0,
        Duration:  time.Second * 2,
    }
    
    if err := store.SaveCommand(record); err != nil {
        log.Fatal(err)
    }
}
```

### Safe Command Execution

```go
package main

import (
    "fmt"
    "log"
    
    "command-history-tracker/internal/executor"
    "github.com/ValGrace/command-history-tracker/pkg/history"
)

func main() {
    exec := executor.NewExecutor()
    
    cmd := &history.CommandRecord{
        Command: "rm -rf temp/",
        // ... other fields
    }
    
    // Validate command safety
    if err := exec.ValidateCommand(cmd); err != nil {
        log.Printf("Unsafe command: %v", err)
        return
    }
    
    // Get user confirmation for destructive commands
    if exec.RequiresConfirmation(cmd.Command) {
        confirmed, err := exec.ConfirmExecution(cmd)
        if err != nil || !confirmed {
            fmt.Println("Execution cancelled")
            return
        }
    }
    
    fmt.Println("Command validated and confirmed")
}
```

### Querying Command History

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
    
    // Get all directories with history
    dirs, err := store.GetDirectoriesWithHistory()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Directories with command history:")
    for _, dir := range dirs {
        fmt.Printf("  - %s\n", dir)
    }
    
    // Get commands for a specific directory
    commands, err := store.GetCommandsByDirectory(dirs[0])
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("\nRecent commands in %s:\n", dirs[0])
    for i, cmd := range commands {
        if i >= 10 {
            break
        }
        fmt.Printf("  %s\n", cmd.Command)
    }
}
```

## API Documentation

### Core Interfaces

#### StorageEngine

The `StorageEngine` interface provides methods for persisting and retrieving command history:

```go
type StorageEngine interface {
    SaveCommand(cmd CommandRecord) error
    GetCommandsByDirectory(dir string) ([]CommandRecord, error)
    GetDirectoriesWithHistory() ([]string, error)
    CleanupOldCommands(retentionDays int) error
    Close() error
}
```

#### CommandInterceptor

The `CommandInterceptor` interface handles command capture from shells:

```go
type CommandInterceptor interface {
    StartRecording() error
    StopRecording() error
    SetupShellIntegration(shell ShellType) error
}
```

#### HistoryBrowser

The `HistoryBrowser` interface provides interactive command browsing:

```go
type HistoryBrowser interface {
    ShowDirectoryHistory(dir string) error
    ShowDirectoryTree() error
    SelectCommand() (*CommandRecord, error)
    FilterCommands(pattern string) error
}
```

### Data Structures

#### CommandRecord

```go
type CommandRecord struct {
    ID          string        // Unique identifier
    Command     string        // Command text
    Directory   string        // Execution directory
    Timestamp   time.Time     // Execution time
    Shell       ShellType     // Shell type (PowerShell, Bash, etc.)
    ExitCode    int           // Command exit code
    Duration    time.Duration // Execution duration
}
```

#### Config

```go
type Config struct {
    StoragePath     string      // Path to storage directory
    RetentionDays   int         // Days to retain history
    MaxCommands     int         // Max commands per directory
    EnabledShells   []ShellType // Enabled shell types
    ExcludePatterns []string    // Command patterns to exclude
    AutoCleanup     bool        // Enable automatic cleanup
}
```

### Shell Integration

The tracker supports multiple shell types:

- **PowerShell** (Windows)
- **Bash** (Linux/macOS/Windows)
- **Zsh** (Linux/macOS)
- **Cmd** (Windows)

Shell integration is automatically configured during setup and uses shell-specific hooks to capture commands without interfering with normal shell operation.

## License

[License information to be added]

## Contributing

[Contributing guidelines to be added]