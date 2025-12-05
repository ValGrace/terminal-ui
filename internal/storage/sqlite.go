package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ValGrace/command-history-tracker/pkg/history"
	_ "modernc.org/sqlite"
)

// SQLiteStorage implements the StorageEngine interface using SQLite
type SQLiteStorage struct {
	dbPath string
	db     *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage engine
func NewSQLiteStorage(dbPath string) *SQLiteStorage {
	return &SQLiteStorage{
		dbPath: dbPath,
	}
}

// Initialize opens the database connection and creates tables if needed
func (s *SQLiteStorage) Initialize() error {
	// Ensure directory exists (only if not using current directory)
	if filepath.Dir(s.dbPath) != "." {
		if err := os.MkdirAll(filepath.Dir(s.dbPath), 0755); err != nil {
			return fmt.Errorf("failed to create storage directory: %w", err)
		}
	}

	// Open database connection
	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)           // Maximum number of open connections
	db.SetMaxIdleConns(5)            // Maximum number of idle connections
	db.SetConnMaxLifetime(time.Hour) // Maximum lifetime of a connection

	s.db = db

	// Configure SQLite for better performance
	pragmas := []string{
		"PRAGMA journal_mode=WAL",   // Write-Ahead Logging for better concurrency
		"PRAGMA synchronous=NORMAL", // Balance between safety and performance
		"PRAGMA cache_size=-64000",  // 64MB cache
		"PRAGMA temp_store=MEMORY",  // Store temp tables in memory
		"PRAGMA busy_timeout=5000",  // Wait up to 5 seconds on lock
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			// Log warning but don't fail initialization
			fmt.Printf("Warning: failed to set pragma %s: %v\n", pragma, err)
		}
	}

	// Create tables and indexes
	if err := s.createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Run migrations
	if err := s.runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// createTables creates the necessary database tables and indexes
func (s *SQLiteStorage) createTables() error {
	// Create schema version table
	createVersionTable := `CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY, applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`

	if _, err := s.db.Exec(createVersionTable); err != nil {
		return fmt.Errorf("failed to create schema_version table: %w", err)
	}

	// Create commands table
	createCommandsTable := `
	CREATE TABLE IF NOT EXISTS commands (
		id TEXT PRIMARY KEY,
		command TEXT NOT NULL,
		directory TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		shell INTEGER NOT NULL,
		exit_code INTEGER NOT NULL DEFAULT 0,
		duration INTEGER NOT NULL DEFAULT 0,
		tags TEXT DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := s.db.Exec(createCommandsTable); err != nil {
		return fmt.Errorf("failed to create commands table: %w", err)
	}

	// Create indexes for performance
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_commands_directory ON commands(directory);",
		"CREATE INDEX IF NOT EXISTS idx_commands_timestamp ON commands(timestamp);",
		"CREATE INDEX IF NOT EXISTS idx_commands_dir_timestamp ON commands(directory, timestamp DESC);",
		"CREATE INDEX IF NOT EXISTS idx_commands_shell ON commands(shell);",
		"CREATE INDEX IF NOT EXISTS idx_commands_command ON commands(command);",
	}

	for _, indexSQL := range indexes {
		if _, err := s.db.Exec(indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Create directory_stats table for optimization
	createStatsTable := `
	CREATE TABLE IF NOT EXISTS directory_stats (
		path TEXT PRIMARY KEY,
		command_count INTEGER NOT NULL DEFAULT 0,
		last_used DATETIME NOT NULL,
		is_active BOOLEAN NOT NULL DEFAULT 1,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := s.db.Exec(createStatsTable); err != nil {
		return fmt.Errorf("failed to create directory_stats table: %w", err)
	}

	return nil
}

// runMigrations applies database schema migrations
func (s *SQLiteStorage) runMigrations() error {
	// Get current schema version
	var currentVersion int
	err := s.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to get schema version: %w", err)
	}

	// Define migrations
	migrations := []struct {
		version int
		sql     string
	}{
		{
			version: 1,
			sql: `
			-- Initial schema is already created in createTables
			-- This migration just marks the initial version
			`,
		},
	}

	// Apply migrations
	for _, migration := range migrations {
		if migration.version > currentVersion {
			if migration.sql != "" {
				if _, err := s.db.Exec(migration.sql); err != nil {
					return fmt.Errorf("failed to apply migration %d: %w", migration.version, err)
				}
			}

			// Record migration
			_, err := s.db.Exec("INSERT INTO schema_version (version) VALUES (?)", migration.version)
			if err != nil {
				return fmt.Errorf("failed to record migration %d: %w", migration.version, err)
			}
		}
	}

	return nil
}

// SaveCommand stores a command record
func (s *SQLiteStorage) SaveCommand(cmd history.CommandRecord) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Validate command record
	if err := cmd.Validate(); err != nil {
		return fmt.Errorf("invalid command record: %w", err)
	}

	// Convert tags to comma-separated string
	tagsStr := strings.Join(cmd.Tags, ",")

	// Insert command
	insertSQL := `
	INSERT INTO commands (id, command, directory, timestamp, shell, exit_code, duration, tags)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(insertSQL, cmd.ID, cmd.Command, cmd.Directory, cmd.Timestamp, int(cmd.Shell), cmd.ExitCode, int64(cmd.Duration), tagsStr)
	if err != nil {
		return fmt.Errorf("failed to save command: %w", err)
	}

	// Update directory stats
	if err := s.updateDirectoryStats(cmd.Directory); err != nil {
		// Log error but don't fail the save operation
		fmt.Printf("Warning: failed to update directory stats: %v\n", err)
	}

	return nil
}

// GetCommandsByDirectory retrieves commands for a specific directory
func (s *SQLiteStorage) GetCommandsByDirectory(dir string) ([]history.CommandRecord, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
	SELECT id, command, directory, timestamp, shell, exit_code, duration, tags
	FROM commands
	WHERE directory = ?
	ORDER BY timestamp DESC`

	rows, err := s.db.Query(query, dir)
	if err != nil {
		return nil, fmt.Errorf("failed to query commands: %w", err)
	}
	defer rows.Close()

	return s.scanCommands(rows)
}

// GetDirectoriesWithHistory returns all directories that have command history
func (s *SQLiteStorage) GetDirectoriesWithHistory() ([]string, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `SELECT DISTINCT directory FROM commands ORDER BY directory`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query directories: %w", err)
	}
	defer rows.Close()

	var directories []string
	for rows.Next() {
		var dir string
		if err := rows.Scan(&dir); err != nil {
			return nil, fmt.Errorf("failed to scan directory: %w", err)
		}
		directories = append(directories, dir)
	}

	return directories, nil
}

// CleanupOldCommands removes commands older than specified retention period
func (s *SQLiteStorage) CleanupOldCommands(retentionDays int) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	// Delete old commands
	deleteSQL := `DELETE FROM commands WHERE timestamp < ?`
	result, err := s.db.Exec(deleteSQL, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup old commands: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("Cleaned up %d old commands\n", rowsAffected)

		// Update directory stats after cleanup
		if err := s.refreshDirectoryStats(); err != nil {
			fmt.Printf("Warning: failed to refresh directory stats: %v\n", err)
		}
	}

	return nil
}

// SearchCommands finds commands matching a pattern
func (s *SQLiteStorage) SearchCommands(pattern string, dir string) ([]history.CommandRecord, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var query string
	var args []interface{}

	if dir != "" {
		query = `
		SELECT id, command, directory, timestamp, shell, exit_code, duration, tags
		FROM commands
		WHERE directory = ? AND command LIKE ?
		ORDER BY timestamp DESC`
		args = []interface{}{dir, "%" + pattern + "%"}
	} else {
		query = `
		SELECT id, command, directory, timestamp, shell, exit_code, duration, tags
		FROM commands
		WHERE command LIKE ?
		ORDER BY timestamp DESC`
		args = []interface{}{"%" + pattern + "%"}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search commands: %w", err)
	}
	defer rows.Close()

	return s.scanCommands(rows)
}

// BatchSaveCommands saves multiple commands in a single transaction
func (s *SQLiteStorage) BatchSaveCommands(commands []history.CommandRecord) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	if len(commands) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Ensure rollback is attempted if commit is not reached. Check the
	// returned error and ignore ErrTxDone which indicates the transaction
	// has already been committed or rolled back.
	// defer tx.Rollback()
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			fmt.Printf("Warning: failed to rollback transaction: %v\n", err)
		}
	}()

	insertSQL := `
	INSERT INTO commands (id, command, directory, timestamp, shell, exit_code, duration, tags)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, cmd := range commands {
		if err := cmd.Validate(); err != nil {
			return fmt.Errorf("invalid command record: %w", err)
		}

		tagsStr := strings.Join(cmd.Tags, ",")
		_, err := stmt.Exec(cmd.ID, cmd.Command, cmd.Directory, cmd.Timestamp, int(cmd.Shell), cmd.ExitCode, int64(cmd.Duration), tagsStr)
		if err != nil {
			return fmt.Errorf("failed to save command: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update directory stats for all affected directories
	directories := make(map[string]bool)
	for _, cmd := range commands {
		directories[cmd.Directory] = true
	}

	for dir := range directories {
		if err := s.updateDirectoryStats(dir); err != nil {
			fmt.Printf("Warning: failed to update directory stats for %s: %v\n", dir, err)
		}
	}

	return nil
}

// scanCommands is a helper function to scan command records from SQL rows
func (s *SQLiteStorage) scanCommands(rows *sql.Rows) ([]history.CommandRecord, error) {
	var commands []history.CommandRecord
	for rows.Next() {
		var cmd history.CommandRecord
		var tagsStr string
		var shellInt int
		var durationInt int64

		err := rows.Scan(&cmd.ID, &cmd.Command, &cmd.Directory, &cmd.Timestamp, &shellInt, &cmd.ExitCode, &durationInt, &tagsStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan command: %w", err)
		}

		cmd.Shell = history.ShellType(shellInt)
		cmd.Duration = time.Duration(durationInt)

		// Parse tags
		if tagsStr != "" {
			cmd.Tags = strings.Split(tagsStr, ",")
		} else {
			cmd.Tags = []string{}
		}

		commands = append(commands, cmd)
	}

	return commands, nil
}

// updateDirectoryStats updates statistics for a directory
func (s *SQLiteStorage) updateDirectoryStats(dir string) error {
	// Get current count
	var count int
	countQuery := `SELECT COUNT(*) FROM commands WHERE directory = ?`
	if err := s.db.QueryRow(countQuery, dir).Scan(&count); err != nil {
		return err
	}

	// Update or insert directory stats
	upsertSQL := `
	INSERT INTO directory_stats (path, command_count, last_used, is_active)
	VALUES (?, ?, ?, 1)
	ON CONFLICT(path) DO UPDATE SET
		command_count = ?,
		last_used = ?,
		is_active = 1,
		updated_at = CURRENT_TIMESTAMP`

	now := time.Now()
	_, err := s.db.Exec(upsertSQL, dir, count, now, count, now)
	return err
}

// refreshDirectoryStats recalculates all directory statistics
func (s *SQLiteStorage) refreshDirectoryStats() error {
	// Clear existing stats
	if _, err := s.db.Exec(`DELETE FROM directory_stats`); err != nil {
		return err
	}

	// Recalculate stats
	query := `
	INSERT INTO directory_stats (path, command_count, last_used, is_active)
	SELECT directory, COUNT(*), MAX(timestamp), 1
	FROM commands
	GROUP BY directory`

	_, err := s.db.Exec(query)
	return err
}

// GetDirectoryStats returns directory statistics
func (s *SQLiteStorage) GetDirectoryStats() ([]history.DirectoryIndex, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
	SELECT path, command_count, last_used, is_active
	FROM directory_stats
	ORDER BY last_used DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query directory stats: %w", err)
	}
	defer rows.Close()

	var stats []history.DirectoryIndex
	for rows.Next() {
		var stat history.DirectoryIndex
		err := rows.Scan(&stat.Path, &stat.CommandCount, &stat.LastUsed, &stat.IsActive)
		if err != nil {
			return nil, fmt.Errorf("failed to scan directory stat: %w", err)
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// GetCommandCountByDirectory returns the number of commands for each directory
func (s *SQLiteStorage) GetCommandCountByDirectory() (map[string]int, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
	SELECT directory, COUNT(*) as count
	FROM commands
	GROUP BY directory
	ORDER BY count DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query command counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var dir string
		var count int
		if err := rows.Scan(&dir, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[dir] = count
	}

	return counts, nil
}

// GetRecentDirectories returns directories ordered by most recent activity
func (s *SQLiteStorage) GetRecentDirectories(limit int) ([]string, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
	SELECT directory
	FROM commands
	GROUP BY directory
	ORDER BY MAX(timestamp) DESC`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent directories: %w", err)
	}
	defer rows.Close()

	var directories []string
	for rows.Next() {
		var dir string
		if err := rows.Scan(&dir); err != nil {
			return nil, fmt.Errorf("failed to scan directory: %w", err)
		}
		directories = append(directories, dir)
	}

	return directories, nil
}

// GetCommandsByTimeRange retrieves commands within a specific time range
func (s *SQLiteStorage) GetCommandsByTimeRange(startTime, endTime time.Time, dir string) ([]history.CommandRecord, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var query string
	var args []interface{}

	if dir != "" {
		query = `
		SELECT id, command, directory, timestamp, shell, exit_code, duration, tags
		FROM commands
		WHERE directory = ? AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp DESC`
		args = []interface{}{dir, startTime, endTime}
	} else {
		query = `
		SELECT id, command, directory, timestamp, shell, exit_code, duration, tags
		FROM commands
		WHERE timestamp BETWEEN ? AND ?
		ORDER BY timestamp DESC`
		args = []interface{}{startTime, endTime}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query commands by time range: %w", err)
	}
	defer rows.Close()

	return s.scanCommands(rows)
}

// GetCommandsByShell retrieves commands filtered by shell type
func (s *SQLiteStorage) GetCommandsByShell(shellType history.ShellType, dir string) ([]history.CommandRecord, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var query string
	var args []interface{}

	if dir != "" {
		query = `
		SELECT id, command, directory, timestamp, shell, exit_code, duration, tags
		FROM commands
		WHERE directory = ? AND shell = ?
		ORDER BY timestamp DESC`
		args = []interface{}{dir, int(shellType)}
	} else {
		query = `
		SELECT id, command, directory, timestamp, shell, exit_code, duration, tags
		FROM commands
		WHERE shell = ?
		ORDER BY timestamp DESC`
		args = []interface{}{int(shellType)}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query commands by shell: %w", err)
	}
	defer rows.Close()

	return s.scanCommands(rows)
}

// FilterCommands retrieves commands with multiple filter criteria
func (s *SQLiteStorage) FilterCommands(filters CommandFilters) ([]history.CommandRecord, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Build query dynamically based on filters
	query := `
	SELECT id, command, directory, timestamp, shell, exit_code, duration, tags
	FROM commands
	WHERE 1=1`

	var args []interface{}

	// Directory filter
	if filters.Directory != "" {
		query += ` AND directory = ?`
		args = append(args, filters.Directory)
	}

	// Text pattern filter
	if filters.Pattern != "" {
		query += ` AND command LIKE ?`
		args = append(args, "%"+filters.Pattern+"%")
	}

	// Shell type filter
	if filters.ShellType != history.Unknown {
		query += ` AND shell = ?`
		args = append(args, int(filters.ShellType))
	}

	// Date range filter
	if !filters.StartTime.IsZero() {
		query += ` AND timestamp >= ?`
		args = append(args, filters.StartTime)
	}
	if !filters.EndTime.IsZero() {
		query += ` AND timestamp <= ?`
		args = append(args, filters.EndTime)
	}

	// Exit code filter
	if filters.ExitCode != nil {
		query += ` AND exit_code = ?`
		args = append(args, *filters.ExitCode)
	}

	// Order by timestamp descending (most recent first)
	query += ` ORDER BY timestamp DESC`

	// Apply limit if specified
	if filters.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filters.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to filter commands: %w", err)
	}
	defer rows.Close()

	return s.scanCommands(rows)
}

// CommandFilters defines filter criteria for command queries
type CommandFilters struct {
	Directory string
	Pattern   string
	ShellType history.ShellType
	StartTime time.Time
	EndTime   time.Time
	ExitCode  *int
	Limit     int
}

// OptimizeDatabase performs database optimization operations
func (s *SQLiteStorage) OptimizeDatabase() error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Run VACUUM to reclaim space and optimize database
	if _, err := s.db.Exec("VACUUM"); err != nil {
		return fmt.Errorf("failed to vacuum database: %w", err)
	}

	// Analyze tables for query optimization
	if _, err := s.db.Exec("ANALYZE"); err != nil {
		return fmt.Errorf("failed to analyze database: %w", err)
	}

	// Refresh directory stats
	if err := s.refreshDirectoryStats(); err != nil {
		return fmt.Errorf("failed to refresh directory stats: %w", err)
	}

	return nil
}

// GetDatabaseSize returns the size of the database file in bytes
func (s *SQLiteStorage) GetDatabaseSize() (int64, error) {
	if s.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	var pageCount, pageSize int64

	if err := s.db.QueryRow("PRAGMA page_count").Scan(&pageCount); err != nil {
		return 0, fmt.Errorf("failed to get page count: %w", err)
	}

	if err := s.db.QueryRow("PRAGMA page_size").Scan(&pageSize); err != nil {
		return 0, fmt.Errorf("failed to get page size: %w", err)
	}

	return pageCount * pageSize, nil
}

// Close closes the storage connection
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
