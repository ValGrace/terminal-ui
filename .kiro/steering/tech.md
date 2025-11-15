# Technology Stack

## Core Technologies

- **Language**: Go (Golang)
- **Database**: SQLite for command history storage
- **CLI Framework**: Cobra for command-line interface
- **Terminal UI**: Bubbletea for interactive terminal interfaces
- **Module System**: Standard Go modules with go.mod

## Architecture Components

- **Command Interceptor**: Shell-specific hooks for capturing terminal commands
- **Storage Engine**: SQLite-based persistence with directory indexing
- **History Browser**: Interactive terminal UI for command navigation
- **Command Executor**: Safe command execution with validation
- **Configuration Manager**: JSON-based configuration system

## Project Structure

```
cmd/           # CLI application entry points
internal/      # Private application code
pkg/           # Public library interfaces
```

## Common Commands

### Development
```bash
# Initialize module
go mod init command-history-tracker

# Build application
go build ./cmd/...

# Run tests
go test ./...

# Install locally
go install ./cmd/...
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...
```

## Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/charmbracelet/bubbletea` - Terminal UI
- `modernc.org/sqlite` - SQLite driver
- Standard library packages for cross-platform support

## Cross-Platform Considerations

- Shell detection for PowerShell, Bash, Zsh, Cmd
- Platform-specific command interception mechanisms
- File system path handling across Windows/Unix systems