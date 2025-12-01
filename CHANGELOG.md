# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of Command History Tracker
- Automatic command recording for PowerShell, Bash, Zsh, and Cmd
- Interactive terminal UI for browsing command history
- Cross-directory command navigation
- Command search and filtering capabilities
- Safe command execution with validation
- SQLite-based storage with efficient indexing
- Configurable retention policies and auto-cleanup
- Cross-platform support (Windows, macOS, Linux)
- Performance optimizations with caching and connection pooling
- Comprehensive logging system
- Version management and build automation

### Changed
- N/A

### Deprecated
- N/A

### Removed
- N/A

### Fixed
- Directory path normalization in history and browse commands to ensure consistent cross-platform lookups
- Added shared utility function for path normalization to maintain consistency across CLI commands
- Timestamp parsing in SQLite storage to handle both RFC3339 and SQLite datetime formats correctly

### Security
- Command validation to prevent injection attacks
- Secure file permissions for command history storage

## [0.1.0] - TBD

### Added
- Initial release with core functionality
- Command recording and history management
- Interactive browser interface
- Cross-platform shell integration
- Storage engine with SQLite backend
- Configuration management
- CLI commands for all operations

[Unreleased]: https://github.com/yourusername/command-history-tracker/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/yourusername/command-history-tracker/releases/tag/v0.1.0
