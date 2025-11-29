# Integration Guide

This guide provides detailed examples for integrating Command History Tracker into your projects and workflows.

## Table of Contents

- [Using as a Library](#using-as-a-library)
- [Custom Storage Backends](#custom-storage-backends)
- [Shell Integration](#shell-integration)
- [CI/CD Integration](#cicd-integration)
- [Advanced Use Cases](#advanced-use-cases)

## Using as a Library

### Basic Command Recording

```go
package main

import (
    "log"
    "time"
    
    "command-history-tracker/internal/config"
    "command-history-tracker/internal/storage"
    "github.com/ValGrace/command-history-tracker/pkg/history"
)

func main() {
    // Initialize configuration
    cfg := config.DefaultConfig()
    cfg.StoragePath = "./my-app-history"
    
    // Create storage engine
    store, err := storage.NewSQLiteStorage(cfg.StoragePath)
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Record a command
    record := history.CommandRecord{
        Command:   "go test ./...",
        Directory: "/home/user/myproject",
        Timestamp: time.Now(),
        Shell:     history.Bash,
        ExitCode:  0,
        Duration:  time.Second * 5,
    }
    
    if err := store.SaveCommand(record); err != nil {
        log.Fatal(err)
    }
    
    log.Println("Command recorded successfully")
}
```

### Querying Command History

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "command-history-tracker/internal/storage"
)

func main() {
    store, err := storage.NewSQLiteStorage("~/.command-history-tracker")
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Get commands from the last 7 days
    cutoff := time.Now().AddDate(0, 0, -7)
    commands, err := store.GetCommandsByDirectory("/home/user/project")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Commands from the last 7 days:")
    for _, cmd := range commands {
        if cmd.Timestamp.After(cutoff) {
            fmt.Printf("[%s] %s (exit: %d)\n", 
                cmd.Timestamp.Format("2006-01-02 15:04"), 
                cmd.Command, 
                cmd.ExitCode)
        }
    }
}
```

### Building a Custom Browser

```go
package main

import (
    "fmt"
    "log"
    
    "command-history-tracker/internal/storage"
    "github.com/ValGrace/command-history-tracker/pkg/history"
)

type SimpleBrowser struct {
    storage history.StorageEngine
}

func NewSimpleBrowser(storagePath string) (*SimpleBrowser, error) {
    store, err := storage.NewSQLiteStorage(storagePath)
    if err != nil {
        return nil, err
    }
    
    return &SimpleBrowser{storage: store}, nil
}

func (b *SimpleBrowser) ShowHistory(dir string, limit int) error {
    commands, err := b.storage.GetCommandsByDirectory(dir)
    if err != nil {
        return err
    }
    
    fmt.Printf("Command history for %s:\n\n", dir)
    
    count := 0
    for _, cmd := range commands {
        if count >= limit {
            break
        }
        
        fmt.Printf("%d. %s\n", count+1, cmd.Command)
        fmt.Printf("   Time: %s | Exit: %d | Duration: %s\n\n",
            cmd.Timestamp.Format("2006-01-02 15:04:05"),
            cmd.ExitCode,
            cmd.Duration)
        
        count++
    }
    
    return nil
}

func main() {
    browser, err := NewSimpleBrowser("~/.command-history-tracker")
    if err != nil {
        log.Fatal(err)
    }
    
    if err := browser.ShowHistory("/home/user/project", 10); err != nil {
        log.Fatal(err)
    }
}
```

### Directory Navigation

The browser supports hierarchical directory navigation with tree organization:

```go
package main

import (
    "fmt"
    "log"
    "strings"
    
    "command-history-tracker/internal/storage"
    "github.com/ValGrace/command-history-tracker/pkg/history"
)

type DirectoryNavigator struct {
    storage history.StorageEngine
}

func NewDirectoryNavigator(storagePath string) (*DirectoryNavigator, error) {
    store, err := storage.NewSQLiteStorage(storagePath)
    if err != nil {
        return nil, err
    }
    
    return &DirectoryNavigator{storage: store}, nil
}

// ShowDirectoryTree displays directories in a hierarchical tree structure
func (n *DirectoryNavigator) ShowDirectoryTree() error {
    dirs, err := n.storage.GetDirectoriesWithHistory()
    if err != nil {
        return err
    }
    
    // Organize directories hierarchically
    tree := n.organizeHierarchically(dirs)
    
    fmt.Println("Directory Tree:")
    n.printTree(tree, 0)
    
    return nil
}

// organizeHierarchically organizes directories into a tree structure
func (n *DirectoryNavigator) organizeHierarchically(dirs []string) map[string][]string {
    tree := make(map[string][]string)
    
    for _, dir := range dirs {
        parts := strings.Split(dir, "/")
        if len(parts) > 1 {
            parent := strings.Join(parts[:len(parts)-1], "/")
            if parent == "" {
                parent = "/"
            }
            tree[parent] = append(tree[parent], dir)
        } else {
            tree["/"] = append(tree["/"], dir)
        }
    }
    
    return tree
}

// printTree recursively prints the directory tree
func (n *DirectoryNavigator) printTree(tree map[string][]string, level int) {
    indent := strings.Repeat("  ", level)
    
    for parent, children := range tree {
        // Get command count for this directory
        commands, _ := n.storage.GetCommandsByDirectory(parent)
        fmt.Printf("%s📁 %s (%d commands)\n", indent, parent, len(commands))
        
        for _, child := range children {
            n.printTree(map[string][]string{child: tree[child]}, level+1)
        }
    }
}

// NavigateToParent returns the parent directory path
func (n *DirectoryNavigator) NavigateToParent(currentDir string) string {
    if currentDir == "/" || currentDir == "" {
        return currentDir
    }
    
    parts := strings.Split(currentDir, "/")
    if len(parts) <= 1 {
        return "/"
    }
    
    return strings.Join(parts[:len(parts)-1], "/")
}

// IsParentOf checks if one directory is a parent of another
func (n *DirectoryNavigator) IsParentOf(parent, child string) bool {
    if parent == child {
        return false
    }
    
    return strings.HasPrefix(child, parent+"/")
}

func main() {
    nav, err := NewDirectoryNavigator("~/.command-history-tracker")
    if err != nil {
        log.Fatal(err)
    }
    
    // Show directory tree
    if err := nav.ShowDirectoryTree(); err != nil {
        log.Fatal(err)
    }
    
    // Navigate to parent
    currentDir := "/home/user/project/src"
    parentDir := nav.NavigateToParent(currentDir)
    fmt.Printf("\nParent of %s is %s\n", currentDir, parentDir)
    
    // Check parent relationship
    if nav.IsParentOf("/home/user", "/home/user/project") {
        fmt.Println("/home/user is a parent of /home/user/project")
    }
}
```

**Directory Navigation Features**:

- **Hierarchical Organization**: Directories are automatically organized into a tree structure
- **Parent Navigation**: Navigate up the directory hierarchy with keyboard shortcuts
- **Breadcrumb Support**: Visual breadcrumbs show the current path
- **Cross-Platform Paths**: Supports both Unix (`/home/user`) and Windows (`C:\Users\test`) path formats
- **Visual Indicators**: 
  - Current directory highlighting
  - Inactive directory dimming
  - Command count per directory
  - Indentation for hierarchy levels

## Custom Storage Backends

You can implement custom storage backends by implementing the `StorageEngine` interface:

```go
package customstorage

import (
    "github.com/ValGrace/command-history-tracker/pkg/history"
)

type CustomStorage struct {
    // Your custom storage fields
}

func NewCustomStorage(config map[string]interface{}) (*CustomStorage, error) {
    // Initialize your storage
    return &CustomStorage{}, nil
}

func (s *CustomStorage) SaveCommand(cmd history.CommandRecord) error {
    // Implement command saving logic
    return nil
}

func (s *CustomStorage) GetCommandsByDirectory(dir string) ([]history.CommandRecord, error) {
    // Implement command retrieval logic
    return nil, nil
}

func (s *CustomStorage) GetDirectoriesWithHistory() ([]string, error) {
    // Implement directory listing logic
    return nil, nil
}

func (s *CustomStorage) CleanupOldCommands(retentionDays int) error {
    // Implement cleanup logic
    return nil
}

func (s *CustomStorage) Close() error {
    // Cleanup resources
    return nil
}
```

## Shell Integration

### Manual Shell Integration

If you prefer manual shell integration instead of using `tracker setup`:

#### Bash/Zsh

Add to `~/.bashrc` or `~/.zshrc`:

```bash
# Command History Tracker Integration
export TRACKER_ENABLED=1

# Function to record commands
_tracker_record() {
    local exit_code=$?
    local cmd=$(history 1 | sed 's/^[ ]*[0-9]*[ ]*//')
    local dir=$(pwd)
    
    if [ -n "$cmd" ] && [ "$TRACKER_ENABLED" = "1" ]; then
        tracker record --command "$cmd" --directory "$dir" --exit-code $exit_code &
    fi
    
    return $exit_code
}

# Hook into prompt
PROMPT_COMMAND="_tracker_record${PROMPT_COMMAND:+; $PROMPT_COMMAND}"
```

#### PowerShell

Add to your PowerShell profile (`$PROFILE`):

```powershell
# Command History Tracker Integration
$env:TRACKER_ENABLED = "1"

function Invoke-TrackerRecord {
    param($Command, $ExitCode)
    
    if ($env:TRACKER_ENABLED -eq "1") {
        $dir = Get-Location
        Start-Job -ScriptBlock {
            param($cmd, $dir, $exit)
            tracker record --command $cmd --directory $dir --exit-code $exit
        } -ArgumentList $Command, $dir, $ExitCode | Out-Null
    }
}

# Hook into prompt
$ExecutionContext.InvokeCommand.CommandNotFoundAction = {
    param($CommandName, $CommandLookupEventArgs)
    # Record command execution
}
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Track Build Commands

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.24
      
      - name: Install Tracker
        run: |
          go install github.com/ValGrace/command-history-tracker/cmd/tracker@latest
          tracker setup --ci-mode
      
      - name: Build
        run: make build
      
      - name: Test
        run: make test
      
      - name: Export Command History
        if: always()
        run: |
          tracker export --format json > command-history.json
      
      - name: Upload History
        if: always()
        uses: actions/upload-artifact@v2
        with:
          name: command-history
          path: command-history.json
```

### GitLab CI

```yaml
stages:
  - build
  - test

before_script:
  - go install github.com/ValGrace/command-history-tracker/cmd/tracker@latest
  - tracker setup --ci-mode

build:
  stage: build
  script:
    - make build
  after_script:
    - tracker export --format json > command-history.json
  artifacts:
    paths:
      - command-history.json
    when: always

test:
  stage: test
  script:
    - make test
  after_script:
    - tracker export --format json > command-history.json
  artifacts:
    paths:
      - command-history.json
    when: always
```

## Safe Command Execution

### Using the Command Executor

The Command Executor provides a multi-layer safety system for executing commands from history:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/ValGrace/command-history-tracker/internal/executor"
    "github.com/ValGrace/command-history-tracker/internal/storage"
)

func main() {
    // Initialize storage
    store, err := storage.NewSQLiteStorage("~/.command-history-tracker")
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Get commands from history
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
    
    // Select a command to execute
    cmd := &commands[0]
    
    // Step 1: Validate command safety
    if err := exec.ValidateCommand(cmd); err != nil {
        log.Printf("Command blocked for safety: %v", err)
        return
    }
    
    // Step 2: Show preview
    fmt.Println(exec.PreviewCommand(cmd))
    
    // Step 3: Get confirmation if needed
    confirmed, err := exec.ConfirmExecution(cmd)
    if err != nil {
        log.Printf("Confirmation failed: %v", err)
        return
    }
    
    if !confirmed {
        fmt.Println("Command execution cancelled by user")
        return
    }
    
    fmt.Println("✓ Command validated and confirmed")
    // Proceed with actual execution...
}
```

### Checking Command Safety

```go
package main

import (
    "fmt"
    
    "github.com/ValGrace/command-history-tracker/internal/executor"
)

func main() {
    exec := executor.NewExecutor()
    
    // Check if a command is dangerous
    dangerousCommands := []string{
        "rm -rf /",
        "format C:",
        "dd if=/dev/zero of=/dev/sda",
    }
    
    for _, cmd := range dangerousCommands {
        if exec.IsDangerous(cmd) {
            fmt.Printf("⚠️  DANGEROUS: %s\n", cmd)
        }
    }
    
    // Check if a command requires confirmation
    destructiveCommands := []string{
        "rm -rf temp/",
        "git push --force",
        "docker system prune",
    }
    
    for _, cmd := range destructiveCommands {
        if exec.RequiresConfirmation(cmd) {
            fmt.Printf("⚡ REQUIRES CONFIRMATION: %s\n", cmd)
        }
    }
}
```

### Custom Safety Rules

You can extend the executor with custom safety rules:

```go
package main

import (
    "fmt"
    "regexp"
    
    "github.com/ValGrace/command-history-tracker/internal/executor"
    "github.com/ValGrace/command-history-tracker/pkg/history"
)

type CustomExecutor struct {
    *executor.Executor
    customPatterns []*regexp.Regexp
}

func NewCustomExecutor() *CustomExecutor {
    return &CustomExecutor{
        Executor: executor.NewExecutor(),
        customPatterns: []*regexp.Regexp{
            regexp.MustCompile(`^curl.*\|.*bash`), // Piping to bash
            regexp.MustCompile(`^wget.*\|.*sh`),   // Piping to sh
        },
    }
}

func (e *CustomExecutor) ValidateCommand(cmd *history.CommandRecord) error {
    // First run standard validation
    if err := e.Executor.ValidateCommand(cmd); err != nil {
        return err
    }
    
    // Then check custom patterns
    for _, pattern := range e.customPatterns {
        if pattern.MatchString(cmd.Command) {
            return fmt.Errorf("command blocked by custom rule: %s", cmd.Command)
        }
    }
    
    return nil
}

func main() {
    exec := NewCustomExecutor()
    
    cmd := &history.CommandRecord{
        Command: "curl https://example.com/script.sh | bash",
    }
    
    if err := exec.ValidateCommand(cmd); err != nil {
        fmt.Printf("Validation failed: %v\n", err)
    }
}
```

### Safety Features

The executor provides three levels of protection:

1. **Dangerous Command Blocking**: Automatically blocks commands that could cause system damage
   - Root directory deletion (`rm -rf /`)
   - Disk formatting (`format C:`, `mkfs.*`)
   - Fork bombs
   - Direct disk writes (`dd` to devices)

2. **Confirmation Required**: Prompts user before executing potentially destructive commands
   - Recursive deletion (`rm -rf`, `del /s`)
   - Force operations (`git push --force`, `git reset --hard`)
   - System-wide cleanup (`docker system prune`)
   - Database operations (`DROP`, `TRUNCATE`)

3. **Command Preview**: Shows detailed information before execution
   - Full command text
   - Execution context (directory, shell)
   - Previous exit code
   - Warning indicators for destructive operations

## Advanced Use Cases

### Command Analytics

```go
package main

import (
    "fmt"
    "log"
    "sort"
    
    "github.com/ValGrace/command-history-tracker/internal/storage"
)

func main() {
    store, err := storage.NewSQLiteStorage("~/.command-history-tracker")
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Get all directories
    dirs, err := store.GetDirectoriesWithHistory()
    if err != nil {
        log.Fatal(err)
    }
    
    // Count commands per directory
    type dirStats struct {
        path  string
        count int
    }
    
    var stats []dirStats
    for _, dir := range dirs {
        commands, err := store.GetCommandsByDirectory(dir)
        if err != nil {
            continue
        }
        stats = append(stats, dirStats{dir, len(commands)})
    }
    
    // Sort by command count
    sort.Slice(stats, func(i, j int) bool {
        return stats[i].count > stats[j].count
    })
    
    // Display top 10 directories
    fmt.Println("Top 10 directories by command count:")
    for i, s := range stats {
        if i >= 10 {
            break
        }
        fmt.Printf("%d. %s: %d commands\n", i+1, s.path, s.count)
    }
}
```

### Command Frequency Analysis

```go
package main

import (
    "fmt"
    "log"
    "sort"
    "strings"
    
    "github.com/ValGrace/command-history-tracker/internal/storage"
)

func main() {
    store, err := storage.NewSQLiteStorage("~/.command-history-tracker")
    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()
    
    // Get all directories
    dirs, err := store.GetDirectoriesWithHistory()
    if err != nil {
        log.Fatal(err)
    }
    
    // Count command frequencies
    cmdFreq := make(map[string]int)
    
    for _, dir := range dirs {
        commands, err := store.GetCommandsByDirectory(dir)
        if err != nil {
            continue
        }
        
        for _, cmd := range commands {
            // Extract base command (first word)
            parts := strings.Fields(cmd.Command)
            if len(parts) > 0 {
                baseCmd := parts[0]
                cmdFreq[baseCmd]++
            }
        }
    }
    
    // Sort by frequency
    type cmdCount struct {
        cmd   string
        count int
    }
    
    var counts []cmdCount
    for cmd, count := range cmdFreq {
        counts = append(counts, cmdCount{cmd, count})
    }
    
    sort.Slice(counts, func(i, j int) bool {
        return counts[i].count > counts[j].count
    })
    
    // Display top 20 commands
    fmt.Println("Top 20 most used commands:")
    for i, c := range counts {
        if i >= 20 {
            break
        }
        fmt.Printf("%d. %s: %d times\n", i+1, c.cmd, c.count)
    }
}
```

### Automated Command Suggestions

```go
package main

import (
    "fmt"
    "log"
    "strings"
    
    "github.com/ValGrace/command-history-tracker/internal/storage"
)

func suggestCommands(currentDir string, prefix string) ([]string, error) {
    store, err := storage.NewSQLiteStorage("~/.command-history-tracker")
    if err != nil {
        return nil, err
    }
    defer store.Close()
    
    // Get commands for current directory
    commands, err := store.GetCommandsByDirectory(currentDir)
    if err != nil {
        return nil, err
    }
    
    // Find matching commands
    var suggestions []string
    seen := make(map[string]bool)
    
    for _, cmd := range commands {
        if strings.HasPrefix(cmd.Command, prefix) && !seen[cmd.Command] {
            suggestions = append(suggestions, cmd.Command)
            seen[cmd.Command] = true
            
            if len(suggestions) >= 10 {
                break
            }
        }
    }
    
    return suggestions, nil
}

func main() {
    suggestions, err := suggestCommands("/home/user/project", "git ")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Command suggestions:")
    for i, suggestion := range suggestions {
        fmt.Printf("%d. %s\n", i+1, suggestion)
    }
}
```

## CLI Command Workflows

### Parameter Combinations

The CLI supports flexible parameter combinations for various workflows:

#### History with Multiple Filters

```bash
# View recent commands with multiple filters
tracker history --dir /project --limit 10 --since 6h --shell bash

# Non-interactive mode for scripting
tracker history --no-interactive > commands.txt
```

#### Search Across Directories

```bash
# Search all directories with limit
tracker search "git commit" --all-dirs --limit 20

# Case-sensitive search in current directory
tracker search "MyCommand" --case-sensitive
```

#### Browse with Tree View

```bash
# Display directory tree
tracker browse --tree

# Browse specific directory
tracker browse --dir /home/user/project
```

### Command Chaining Workflows

Execute multiple operations in sequence:

#### Record-Search-Execute Workflow

```bash
# 1. Record a command
tracker record --command "npm test" --directory /project

# 2. Search for it
tracker search "npm test"

# 3. Execute from history
tracker browse --dir /project
# (select command interactively)
```

#### Configuration Management Workflow

```bash
# 1. Set configuration values
tracker config set retention-days 200
tracker config set max-commands 50000

# 2. Verify changes
tracker config get retention-days
tracker config show

# 3. Reset if needed
tracker config reset
```

#### Record-Browse-Cleanup Workflow

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/ValGrace/command-history-tracker/internal/storage"
    "github.com/ValGrace/command-history-tracker/pkg/history"
)

func main() {
    store, _ := storage.NewStorageEngine("sqlite", "commands.db")
    defer store.Close()
    store.Initialize()
    
    // 1. Record commands
    for i := 0; i < 5; i++ {
        record := history.CommandRecord{
            Command:   fmt.Sprintf("command-%d", i),
            Directory: "/project",
            Timestamp: time.Now().Add(-time.Duration(i*24) * time.Hour),
            Shell:     history.Bash,
        }
        store.SaveCommand(record)
    }
    
    // 2. Browse commands
    commands, _ := store.GetCommandsByDirectory("/project")
    fmt.Printf("Found %d commands\n", len(commands))
    
    // 3. Cleanup old commands (older than 2 days)
    store.CleanupOldCommands(2)
    
    // 4. Verify cleanup
    commands, _ = store.GetCommandsByDirectory("/project")
    fmt.Printf("After cleanup: %d commands\n", len(commands))
}
```

## Error Handling Patterns

### Graceful Degradation

Handle errors gracefully with fallback mechanisms:

```go
package main

import (
    "log"
    
    "github.com/ValGrace/command-history-tracker/internal/config"
    "github.com/ValGrace/command-history-tracker/internal/storage"
)

func initializeStorage() storage.StorageEngine {
    // Try to load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Println("Config load failed, using defaults")
        cfg = config.DefaultConfig()
    }
    
    // Try to initialize storage
    store, err := storage.NewStorageEngine("sqlite", cfg.StoragePath)
    if err != nil {
        log.Printf("Storage initialization failed: %v", err)
        // Fallback to in-memory or alternative storage
        return nil
    }
    
    if err := store.Initialize(); err != nil {
        log.Printf("Storage initialization failed: %v", err)
        return nil
    }
    
    return store
}
```

### Configuration Validation

Validate configuration before use:

```go
package main

import (
    "fmt"
    
    "github.com/ValGrace/command-history-tracker/internal/config"
)

func validateAndApplyConfig(cfg *config.Config) error {
    // Validate configuration
    if err := cfg.Validate(); err != nil {
        return fmt.Errorf("invalid configuration: %w", err)
    }
    
    // Check specific constraints
    if cfg.RetentionDays < 7 {
        return fmt.Errorf("retention period too short: %d days", cfg.RetentionDays)
    }
    
    if cfg.MaxCommands < 100 {
        return fmt.Errorf("max commands too low: %d", cfg.MaxCommands)
    }
    
    // Apply configuration
    if err := cfg.Save(); err != nil {
        return fmt.Errorf("failed to save configuration: %w", err)
    }
    
    return nil
}

func main() {
    cfg := config.DefaultConfig()
    cfg.RetentionDays = 365
    cfg.MaxCommands = 50000
    
    if err := validateAndApplyConfig(cfg); err != nil {
        fmt.Printf("Configuration error: %v\n", err)
        return
    }
    
    fmt.Println("Configuration applied successfully")
}
```

### Concurrent Access Handling

Handle concurrent storage access safely:

```go
package main

import (
    "fmt"
    "sync"
    "time"
    
    "github.com/ValGrace/command-history-tracker/internal/storage"
    "github.com/ValGrace/command-history-tracker/pkg/history"
)

func concurrentRecording(store storage.StorageEngine, count int) {
    var wg sync.WaitGroup
    errors := make(chan error, count)
    
    for i := 0; i < count; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            record := history.CommandRecord{
                Command:   fmt.Sprintf("concurrent-cmd-%d", id),
                Directory: "/test",
                Timestamp: time.Now(),
                Shell:     history.Bash,
            }
            
            if err := store.SaveCommand(record); err != nil {
                errors <- fmt.Errorf("goroutine %d: %w", id, err)
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    // Check for errors
    for err := range errors {
        fmt.Printf("Error: %v\n", err)
    }
}

func main() {
    store, _ := storage.NewStorageEngine("sqlite", "concurrent.db")
    defer store.Close()
    store.Initialize()
    
    // Record 10 commands concurrently
    concurrentRecording(store, 10)
    
    // Verify all commands were saved
    commands, _ := store.GetCommandsByDirectory("/test")
    fmt.Printf("Saved %d commands concurrently\n", len(commands))
}
```

### Search Error Handling

Handle search errors and empty results:

```go
package main

import (
    "fmt"
    
    "github.com/ValGrace/command-history-tracker/internal/storage"
)

func searchWithFallback(store storage.StorageEngine, pattern, dir string) {
    // Try directory-specific search
    results, err := store.SearchCommands(pattern, dir)
    if err != nil {
        fmt.Printf("Search error: %v\n", err)
        return
    }
    
    if len(results) == 0 {
        fmt.Printf("No results in %s, searching all directories...\n", dir)
        
        // Fallback to all directories
        dirs, _ := store.GetDirectoriesWithHistory()
        for _, d := range dirs {
            results, _ = store.SearchCommands(pattern, d)
            if len(results) > 0 {
                fmt.Printf("Found %d results in %s\n", len(results), d)
                break
            }
        }
    } else {
        fmt.Printf("Found %d results in %s\n", len(results), dir)
    }
}
```

## Testing Integration

### Unit Testing with Mock Storage

```go
package myapp

import (
    "testing"
    
    "github.com/ValGrace/command-history-tracker/pkg/history"
)

type MockStorage struct {
    commands []history.CommandRecord
}

func (m *MockStorage) SaveCommand(cmd history.CommandRecord) error {
    m.commands = append(m.commands, cmd)
    return nil
}

func (m *MockStorage) GetCommandsByDirectory(dir string) ([]history.CommandRecord, error) {
    var results []history.CommandRecord
    for _, cmd := range m.commands {
        if cmd.Directory == dir {
            results = append(results, cmd)
        }
    }
    return results, nil
}

func TestMyFeature(t *testing.T) {
    mock := &MockStorage{}
    
    // Test your feature with mock storage
    cmd := history.CommandRecord{
        Command:   "test",
        Directory: "/test",
    }
    
    if err := mock.SaveCommand(cmd); err != nil {
        t.Fatalf("SaveCommand failed: %v", err)
    }
    
    results, _ := mock.GetCommandsByDirectory("/test")
    if len(results) != 1 {
        t.Errorf("Expected 1 command, got %d", len(results))
    }
}
```

### Integration Testing

```go
package myapp

import (
    "os"
    "path/filepath"
    "testing"
    
    "github.com/ValGrace/command-history-tracker/internal/storage"
    "github.com/ValGrace/command-history-tracker/pkg/history"
)

func TestIntegration(t *testing.T) {
    // Create temporary database
    tmpDir := t.TempDir()
    dbPath := filepath.Join(tmpDir, "test.db")
    
    store, err := storage.NewStorageEngine("sqlite", dbPath)
    if err != nil {
        t.Fatalf("Failed to create storage: %v", err)
    }
    defer store.Close()
    
    if err := store.Initialize(); err != nil {
        t.Fatalf("Failed to initialize: %v", err)
    }
    
    // Test complete workflow
    cmd := history.CommandRecord{
        Command:   "integration-test",
        Directory: "/test",
    }
    
    // Save
    if err := store.SaveCommand(cmd); err != nil {
        t.Fatalf("SaveCommand failed: %v", err)
    }
    
    // Retrieve
    results, err := store.GetCommandsByDirectory("/test")
    if err != nil {
        t.Fatalf("GetCommandsByDirectory failed: %v", err)
    }
    
    if len(results) != 1 {
        t.Errorf("Expected 1 command, got %d", len(results))
    }
}
```

## Best Practices

1. **Storage Location**: Use a dedicated directory for command history storage
2. **Retention Policies**: Set appropriate retention periods based on your needs (30-365 days)
3. **Exclude Patterns**: Configure patterns to exclude sensitive commands (passwords, tokens)
4. **Performance**: Use batch operations when recording multiple commands
5. **Error Handling**: Always handle errors gracefully with fallback mechanisms
6. **Concurrency**: The storage engine is thread-safe with SQLite WAL mode
7. **Configuration Validation**: Always validate configuration before applying changes
8. **Testing**: Use mock storage for unit tests, temporary databases for integration tests
9. **Command Chaining**: Leverage CLI parameter combinations for efficient workflows
10. **Safety Checks**: Use the executor's validation before executing commands from history

## Troubleshooting

### Commands Not Being Recorded

1. Check if shell integration is properly configured: `tracker status`
2. Verify the tracker is running: `tracker status`
3. Check shell profile has been sourced: `source ~/.bashrc` or restart terminal
4. Verify environment variables: `echo $CHT_TRACKER_PATH`
5. Check exclude patterns in configuration

### Storage Issues

1. Check storage path permissions: `ls -la ~/.command-history-tracker`
2. Verify disk space availability: `df -h`
3. Check SQLite database integrity: `tracker verify`
4. Review database size: `du -h ~/.command-history-tracker/commands.db`
5. Try database optimization: `tracker optimize`

### Performance Issues

1. Run cleanup to remove old commands: `tracker cleanup --days 30`
2. Reduce retention period in configuration: `tracker config set retention-days 30`
3. Set lower max commands per directory: `tracker config set max-commands 5000`
4. Check database size and run VACUUM: `tracker optimize`
5. Review query patterns and add appropriate filters

### Configuration Issues

1. Validate configuration: `tracker config validate`
2. Check configuration file location: `tracker config path`
3. Reset to defaults if corrupted: `tracker config reset`
4. Verify configuration values: `tracker config show`

### Search Not Finding Commands

1. Verify directory path is correct: `tracker history --dir $(pwd)`
2. Try searching all directories: `tracker search "pattern" --all-dirs`
3. Check case sensitivity: use `--case-sensitive` flag if needed
4. Verify commands were recorded: `tracker status`

For more help, run `tracker --help` or visit the project documentation.
