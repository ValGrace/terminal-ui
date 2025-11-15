# Project Structure

## Directory Organization

```
command-history-tracker/
├── cmd/                    # CLI application entry points
│   ├── tracker/           # Main CLI application
│   └── setup/             # Installation and setup utilities
├── internal/              # Private application code
│   ├── interceptor/       # Command capture implementation
│   ├── storage/           # SQLite storage engine
│   ├── browser/           # Terminal UI for history browsing
│   ├── executor/          # Command execution engine
│   └── config/            # Configuration management
├── pkg/                   # Public library interfaces
│   ├── history/           # Core history tracking interfaces
│   └── shell/             # Shell integration utilities
├── .kiro/                 # Kiro workspace configuration
│   ├── specs/             # Feature specifications
│   └── steering/          # AI assistant guidance
├── go.mod                 # Go module definition
├── go.sum                 # Dependency checksums
├── Makefile              # Build automation
└── README.md             # Project documentation
```

## Code Organization Principles

### Interface-First Design
- Define interfaces in `pkg/` for public APIs
- Implement concrete types in `internal/` packages
- Use dependency injection for testability

### Component Separation
- **interceptor**: Shell-specific command capture logic
- **storage**: Database operations and data persistence
- **browser**: Interactive terminal UI components
- **executor**: Safe command execution with validation
- **config**: Configuration loading and management

### Testing Structure
- Unit tests alongside implementation files (`*_test.go`)
- Integration tests in separate files with build tags
- Test fixtures and mocks in `testdata/` directories
- In-memory SQLite for storage testing

### Error Handling
- Custom error types for different failure categories
- Structured error context with relevant metadata
- Graceful degradation when components are unavailable
- User-friendly error messages with actionable suggestions

## File Naming Conventions

- Interface definitions: `interfaces.go`
- Implementation files: descriptive names (e.g., `sqlite_storage.go`)
- Test files: `*_test.go`
- Mock implementations: `mock_*.go`
- Configuration files: `config.go`

## Import Organization

1. Standard library imports
2. Third-party dependencies
3. Internal project imports (grouped by component)