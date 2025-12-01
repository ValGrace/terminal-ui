# Bug Fix: Path Normalization Issue

## Problem

Users were seeing "No directories with command history found" even after executing commands, despite the tracker showing commands were recorded in the status output.

## Root Cause

**Path normalization mismatch between storage and retrieval:**

1. **During command capture** (`internal/interceptor/capture.go`):
   - Paths were normalized using `filepath.ToSlash()` 
   - Windows paths like `F:\loso` were stored as `F:/loso` (forward slashes)

2. **During command retrieval** (`cmd/tracker/history.go` and `cmd/tracker/browse.go`):
   - Used `os.Getwd()` which returns native path format
   - On Windows: `F:\loso` (backslashes)
   - Database query for `F:\loso` didn't match stored `F:/loso`

3. **Additional issue - Timestamp parsing**:
   - SQLite stored timestamps in multiple formats (Unix timestamps, RFC3339, datetime strings)
   - Scanner only handled `time.Time` type, causing scan errors

## Solution

### 1. Path Normalization Fix

**Created shared utility function** (`cmd/tracker/utils.go`):
```go
func normalizeDirectoryPath(dir string) string {
    return filepath.ToSlash(filepath.Clean(dir))
}
```

**Updated command files**:
- `cmd/tracker/history.go`: Normalize directory before querying
- `cmd/tracker/browse.go`: Normalize directory before browsing

### 2. Timestamp Parsing Fix

**Updated `internal/storage/sqlite.go`**:
- Changed `scanCommands()` to scan timestamp as string first
- Added multi-format timestamp parsing:
  1. Try RFC3339 format
  2. Try SQLite datetime format (`2006-01-02 15:04:05`)
  3. Try Unix timestamp (integer)

## Files Modified

1. `cmd/tracker/utils.go` - **NEW**: Shared path normalization utility
2. `cmd/tracker/history.go` - Added path normalization before query
3. `cmd/tracker/browse.go` - Added path normalization before browse
4. `internal/storage/sqlite.go` - Fixed timestamp parsing in `scanCommands()`

## Testing

### Before Fix
```bash
PS F:\loso> .\tracker.exe history
No directories with command history found.
```

### After Fix
```bash
PS F:\loso> .\tracker.exe history --no-interactive
Command history for: F:/loso
Found 4 command(s)

   1  2025-11-14 09:20:38  go build
   2  2025-11-14 09:20:11  git add .
   3  2025-11-14 09:19:15  echo test
   4  2025-12-01 10:02:39  test-new-command
```

## Impact

- **Cross-platform compatibility**: Ensures consistent path handling on Windows, macOS, and Linux
- **Data integrity**: All existing commands remain accessible
- **User experience**: Commands now display correctly in browse and history views

## Prevention

To prevent similar issues in the future:

1. **Always normalize paths** when comparing or querying directories
2. **Use consistent path format** throughout the application (forward slashes)
3. **Test on multiple platforms** to catch path separator issues
4. **Handle multiple data formats** when reading from storage (timestamps, paths, etc.)

## Related Code

The normalization logic matches the storage format defined in:
- `internal/interceptor/capture.go` - Line 295: `filepath.ToSlash(cleanPath)`
- `internal/interceptor/processor.go` - Similar normalization in processor

## Date

December 1, 2025
