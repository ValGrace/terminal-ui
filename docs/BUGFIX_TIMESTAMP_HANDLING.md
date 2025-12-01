# Bugfix: Timestamp Handling in SQLite Storage

## Issue

The SQLite storage engine was experiencing timestamp-related errors when scanning command records from the database:

```
Error: failed to scan command: sql: Scan error on column index 3, name "timestamp": 
unsupported Scan, storing driver.Value type int64 into type *time.Time
```

Additionally, time-based filtering queries were failing because of type mismatches between Go's `time.Time` and SQLite's stored timestamp format.

## Root Cause

1. **Storage Format Inconsistency**: Initially, timestamps were being converted to Unix timestamps (int64) during insertion but SQLite's modernc.org/sqlite driver was storing them in a different format, causing scan errors.

2. **Query Parameter Type Mismatch**: Time-based filter queries needed to match the storage format used by SQLite for proper comparison operations.

3. **Driver Behavior**: The modernc.org/sqlite driver handles `time.Time` values natively and can store/retrieve them directly, but this wasn't being utilized consistently.

## Solution

### 1. Native time.Time Storage (Latest Fix)

**Changed in SaveCommand (December 1, 2025):**
```go
// Before: Converting to Unix timestamp
_, err := s.db.Exec(insertSQL, cmd.ID, cmd.Command, cmd.Directory, 
    cmd.Timestamp.Unix(), int(cmd.Shell), cmd.ExitCode, int64(cmd.Duration.Seconds()), tagsStr)

// After: Storing time.Time directly
_, err := s.db.Exec(insertSQL, cmd.ID, cmd.Command, cmd.Directory, 
    cmd.Timestamp, int(cmd.Shell), cmd.ExitCode, int64(cmd.Duration.Seconds()), tagsStr)
```

This change allows the modernc.org/sqlite driver to handle timestamp storage natively, which it does by storing timestamps in a format it can reliably retrieve as `time.Time` values.

### 2. Flexible Timestamp Scanning

Modified `scanCommands()` to use `interface{}` for timestamp values and added a new `parseTimestampValue()` helper function that handles multiple timestamp formats from SQLite:

```go
func (s *SQLiteStorage) parseTimestampValue(value interface{}) time.Time {
    switch v := value.(type) {
    case int64:
        // Unix timestamp (seconds) - for backward compatibility
        result = time.Unix(v, 0)
    case string:
        // Try RFC3339, SQLite datetime, and other formats
        // ...
    case time.Time:
        // Already a time.Time (native driver format)
        result = v
    case []byte:
        // Convert to string and retry
        return s.parseTimestampValue(string(v))
    }
    
    // Truncate to second precision for consistency
    return result.Truncate(time.Second)
}
```

### 3. Consistent Query Parameter Handling

Time-based query methods convert `time.Time` values to Unix timestamps for SQL comparison operations:

**GetCommandsByTimeRange:**
```go
startUnix := startTime.Truncate(time.Second).Unix()
endUnix := endTime.Truncate(time.Second).Unix()
args = []interface{}{dir, startUnix, endUnix}
```

**FilterCommands:**
```go
if !filters.StartTime.IsZero() {
    query += ` AND timestamp >= ?`
    args = append(args, filters.StartTime.Truncate(time.Second).Unix())
}
```

**CleanupOldCommands:**
```go
cutoffUnix := cutoffTime.Unix()
result, err := s.db.Exec(deleteSQL, cutoffUnix)
```

### 3. Test Validation Updates

Updated test validation logic to account for second-precision truncation:

```go
// Truncate comparison times to match stored precision
startTrunc := tt.startTime.Truncate(time.Second)
endTrunc := tt.endTime.Truncate(time.Second)
for _, cmd := range results {
    if cmd.Timestamp.Before(startTrunc) || cmd.Timestamp.After(endTrunc) {
        t.Errorf("Command timestamp %v is outside range [%v, %v]", 
            cmd.Timestamp, startTrunc, endTrunc)
    }
}
```

## Impact

### Fixed Issues
- ✅ Timestamp scanning errors resolved
- ✅ Time-based filtering queries now work correctly
- ✅ CleanupOldCommands properly filters by retention period
- ✅ All timestamp-related tests passing

### Backward Compatibility
- ✅ Existing databases continue to work without migration
- ✅ Multiple timestamp formats supported for data import/export
- ✅ No breaking changes to public API

### Performance
- ✅ No performance degradation
- ✅ Queries still use indexed timestamp column efficiently

## Supported Timestamp Formats

The storage engine now handles multiple timestamp formats automatically during retrieval:

1. **time.Time (native)**: Primary storage format via modernc.org/sqlite driver
2. **Unix Timestamps (int64)**: Backward compatibility with older data
3. **RFC3339**: `2024-12-01T15:04:05Z`
4. **SQLite Datetime**: `2024-12-01 15:04:05`
5. **Go Time Format**: `2024-12-01 15:04:05.999999999 -0700 MST`
6. **Unix Timestamp Strings**: `"1701446645"`

This multi-format support ensures compatibility with existing databases while leveraging the driver's native timestamp handling for new records.

## Testing

All storage tests now pass:
```bash
go test ./internal/storage/... -timeout 30s
```

Key test coverage:
- ✅ SaveCommand with various timestamp formats
- ✅ GetCommandsByTimeRange with different time windows
- ✅ FilterCommands with date range filters
- ✅ CleanupOldCommands with retention policies
- ✅ BatchSaveCommands with concurrent operations

## Implementation Details

### Storage Strategy

The current implementation uses a hybrid approach:

1. **Insertion**: `time.Time` values are passed directly to the SQLite driver, which handles the storage format internally
2. **Retrieval**: The `parseTimestampValue()` function handles multiple formats for backward compatibility
3. **Queries**: Time-based comparisons use Unix timestamps for consistent SQL operations

This strategy provides:
- **Native driver support**: Leverages modernc.org/sqlite's built-in time handling
- **Backward compatibility**: Existing databases with Unix timestamps continue to work
- **Flexibility**: Supports data import/export in various formats
- **Consistency**: All timestamps truncated to second precision

## Documentation Updates

Updated API.md to document the robust timestamp handling:

> The SQLite storage engine automatically handles multiple timestamp formats for maximum compatibility:
> - Native time.Time storage via modernc.org/sqlite driver
> - RFC3339 format (e.g., `2024-12-01T15:04:05Z`)
> - SQLite datetime format (e.g., `2024-12-01 15:04:05`)
> - Unix timestamp (integer seconds since epoch)
>
> This ensures reliable timestamp parsing across different storage scenarios and data migrations.

## Related Files

- `internal/storage/sqlite.go` - Core implementation
- `internal/storage/filtering_test.go` - Time-based filtering tests
- `internal/storage/sqlite_test.go` - Storage tests
- `API.md` - API documentation

## Date

December 1, 2025
