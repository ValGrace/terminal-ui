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

## Best Practices

1. **Storage Location**: Use a dedicated directory for command history storage
2. **Retention Policies**: Set appropriate retention periods based on your needs
3. **Exclude Patterns**: Configure patterns to exclude sensitive commands (passwords, tokens)
4. **Performance**: Use batch operations when recording multiple commands
5. **Error Handling**: Always handle errors gracefully and provide fallback mechanisms
6. **Concurrency**: The storage engine is thread-safe, but consider connection pooling for high-concurrency scenarios

## Troubleshooting

### Commands Not Being Recorded

1. Check if shell integration is properly configured: `tracker status`
2. Verify the tracker is running: `tracker status`
3. Check shell profile has been sourced: `source ~/.bashrc` or restart terminal

### Storage Issues

1. Check storage path permissions
2. Verify disk space availability
3. Check SQLite database integrity: `tracker check`

### Performance Issues

1. Run cleanup to remove old commands: `tracker cleanup`
2. Reduce retention period in configuration
3. Set lower max commands per directory

For more help, run `tracker --help` or visit the project documentation.
