# Command History Filtering Guide

This document describes the search and filtering capabilities of the Command History Tracker.

## Overview

The Command History Tracker provides powerful filtering capabilities to help you quickly find the commands you need. You can filter by:

- **Text pattern** - Search for commands containing specific text
- **Date range** - Filter commands by when they were executed
- **Shell type** - Filter by the shell that executed the command (Bash, PowerShell, Zsh, Cmd)
- **Combined filters** - Use multiple filters together for precise results

## Using Filters

### Text Filtering

Press `/` to enter search mode and type your search query. The command list will update in real-time as you type.

**Example:**
- Type `git` to see all git commands
- Type `npm install` to find npm installation commands
- Press `Escape` to cancel search
- Press `Enter` to apply the filter

### Date Range Filtering

Press `f` to show the filter panel, then press `d` to enable date filtering.

**Available date presets:**
1. **Today** - Commands from the last 24 hours
2. **Yesterday** - Commands from yesterday
3. **This Week** - Commands from the current week (Monday to Sunday)
4. **Last Week** - Commands from the previous week
5. **This Month** - Commands from the current month
6. **Last Month** - Commands from the previous month

**Usage:**
- Press `f` to show filters
- Press `d` to enable date filtering (defaults to "Today")
- Press `1-6` to select a date preset
- Press `d` again to disable date filtering

### Shell Type Filtering

Press `f` to show the filter panel, then press `s` to cycle through shell types.

**Available shell filters:**
- PowerShell
- Bash
- Zsh
- Cmd
- None (show all shells)

**Usage:**
- Press `f` to show filters
- Press `s` repeatedly to cycle through shell types
- The current shell filter is displayed in the filter panel

### Combined Filtering

You can use multiple filters simultaneously for precise results.

**Example combinations:**
- Text + Shell: Find all `git` commands executed in Bash
- Text + Date: Find all `npm` commands from today
- Date + Shell: Find all PowerShell commands from this week
- Text + Date + Shell: Find all `docker` commands executed in Bash today

### Clearing Filters

Press `c` (while the filter panel is visible) to clear all active filters and show all commands.

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `/` | Enter search mode |
| `f` | Toggle filter panel |
| `d` | Toggle date filter (when filter panel is visible) |
| `s` | Cycle shell filter (when filter panel is visible) |
| `1-6` | Select date preset (when date filter is enabled) |
| `c` | Clear all filters (when filter panel is visible) |
| `Escape` | Cancel search mode |
| `Enter` | Apply search filter |

## Filter Panel

When you press `f`, the filter panel appears at the top of the screen showing:

- Active text filter (if any)
- Active date filter preset (if enabled)
- Active shell filter (if any)
- Available keyboard shortcuts

**Example filter panel:**
```
Filters: Text: git  Date: Today  Shell: Bash
d: date filter • s: shell filter • c: clear all • f: hide filters
```

## Implementation Details

### UI-Level Filtering

The browser UI applies filters in memory for fast, real-time filtering:

- Text filters use case-insensitive substring matching
- Date filters calculate time ranges based on presets
- Shell filters match exact shell types
- All filters are applied together (AND logic)

### Storage-Level Filtering

The SQLite storage engine provides optimized filtering methods:

- `SearchCommands()` - Text pattern matching with SQL LIKE
- `GetCommandsByTimeRange()` - Date range queries with indexes
- `GetCommandsByShell()` - Shell type filtering
- `FilterCommands()` - Combined filtering with multiple criteria

### Performance

- Filters are applied efficiently using indexed database queries
- UI filtering happens in real-time without database queries
- Large command histories are handled with pagination
- Filter state is preserved when switching views

## Examples

### Find all git commands from today
1. Press `/` and type `git`
2. Press `Enter` to apply
3. Press `f` to show filters
4. Press `d` to enable date filter (defaults to Today)

### Find all PowerShell commands from this week
1. Press `f` to show filters
2. Press `d` to enable date filter
3. Press `3` to select "This Week"
4. Press `s` until "PowerShell" is selected

### Find all docker commands executed in Bash
1. Press `/` and type `docker`
2. Press `Enter`
3. Press `f` to show filters
4. Press `s` until "Bash" is selected

## Tips

- Use text filtering for quick searches
- Combine filters to narrow down results
- Date presets make it easy to find recent commands
- Shell filtering helps when working with multiple shells
- Clear filters with `c` to start fresh
- The filter panel shows what filters are active

## Testing

The filtering functionality is thoroughly tested with comprehensive test coverage:

### Unit Tests
- Individual filter type tests (text, date, shell)
- Filter state management tests
- Filter cycling and preset selection tests

### Integration Tests
- Complete filtering workflow tests
- Combined filter scenarios (text + date + shell)
- Filter clearing and reset functionality
- Storage backend integration tests
- Real-time filter application tests

### Test Coverage Areas
1. **Text Filtering**: Pattern matching, case sensitivity, substring search
2. **Shell Type Filtering**: All shell types (PowerShell, Bash, Zsh, Cmd), filter cycling
3. **Date Range Filtering**: All presets (Today, Yesterday, This Week, Last Week, This Month, Last Month)
4. **Combined Filtering**: Multiple simultaneous filters with AND logic
5. **Storage Integration**: Database-level filtering with SQL queries
6. **Filter State**: Preservation, clearing, and cycling

### Running Tests

Run all filtering tests:
```bash
go test ./internal/browser -v -run Filter
go test ./internal/storage -v -run Filter
```

Run specific test suites:
```bash
# Integration tests
go test ./internal/browser -v -run TestFilteringIntegration

# Storage backend tests
go test ./internal/browser -v -run TestFilteringWithStorageBackend

# Unit tests
go test ./internal/browser -v -run TestFiltering
```

Run with coverage:
```bash
go test ./internal/browser -cover -run Filter
```

### Test Scenarios Covered

The integration tests verify:
- ✓ Text filtering with various patterns
- ✓ Shell type filtering for all supported shells
- ✓ Date range filtering with all presets
- ✓ Combined text + shell filtering
- ✓ Combined text + date + shell filtering
- ✓ Filter clearing and reset
- ✓ Shell filter cycling through all types
- ✓ Date preset cycling through all options
- ✓ Storage-level text search
- ✓ Storage-level shell filtering
- ✓ Storage-level date range filtering
- ✓ Storage-level combined filtering with CommandFilters struct
