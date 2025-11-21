package app

import (
	"github.com/ValGrace/command-history-tracker/internal/browser"
	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/ValGrace/command-history-tracker/internal/errors"
	"github.com/ValGrace/command-history-tracker/internal/executor"
	"github.com/ValGrace/command-history-tracker/internal/interceptor"
	"github.com/ValGrace/command-history-tracker/internal/logging"
	"github.com/ValGrace/command-history-tracker/internal/storage"
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Application represents the main application instance
type Application struct {
	config      *config.Config
	storage     history.StorageEngine
	interceptor *interceptor.CommandRecorder
	browser     *browser.Browser
	executor    *executor.Executor
	logger      *logging.Logger
	mu          sync.RWMutex
	running     bool
	cleanupDone chan struct{}
}

// New creates a new application instance
func New() (*Application, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, errors.NewConfigError("failed to load configuration", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, errors.NewConfigError("invalid configuration", err)
	}

	// Set global configuration
	config.SetGlobal(cfg)

	// Create logger
	logPath := getLogPath()
	logger, err := logging.NewFileLogger(logPath, logging.InfoLevel)
	if err != nil {
		// Fallback to stderr if file logging fails
		logger = logging.New(os.Stderr, logging.InfoLevel)
	}

	// Set as default logger
	logging.SetDefault(logger)

	app := &Application{
		config:      cfg,
		logger:      logger,
		cleanupDone: make(chan struct{}),
	}

	return app, nil
}

// getLogPath returns the path to the log file
func getLogPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./tracker.log"
	}

	logDir := filepath.Join(homeDir, ".command-history-tracker", "logs")
	return filepath.Join(logDir, "tracker.log")
}

// Initialize initializes all application components
func (a *Application) Initialize() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logger.Info("Initializing Command History Tracker...")

	// Initialize storage
	if err := a.initializeStorage(); err != nil {
		return errors.NewStorageError("failed to initialize storage", err)
	}

	// Initialize interceptor
	if err := a.initializeInterceptor(); err != nil {
		return errors.NewShellError("failed to initialize interceptor", err)
	}

	// Initialize browser
	if err := a.initializeBrowser(); err != nil {
		return fmt.Errorf("failed to initialize browser: %w", err)
	}

	// Initialize executor
	if err := a.initializeExecutor(); err != nil {
		return errors.NewExecutionError("failed to initialize executor", err)
	}

	a.logger.Info("✓ All components initialized successfully")
	return nil
}

// initializeStorage initializes the storage engine with caching
func (a *Application) initializeStorage() error {
	a.logger.Info("Initializing storage at: %s", a.config.StoragePath)

	// Create base storage engine (SQLite)
	sqliteStorage := storage.NewSQLiteStorage(a.config.StoragePath)
	
	if err := sqliteStorage.Initialize(); err != nil {
		a.logger.Error("Failed to initialize storage: %v", err)
		return err
	}

	// Wrap with caching layer
	cachedStorage := storage.NewCachedStorage(sqliteStorage, 100, 5*time.Minute)
	a.storage = cachedStorage

	a.logger.Info("✓ Storage initialized with caching (max 100 entries, 5min TTL)")
	return nil
}

// initializeInterceptor initializes the command interceptor
func (a *Application) initializeInterceptor() error {
	a.logger.Info("Initializing command interceptor...")

	recorder, err := interceptor.NewCommandRecorder()
	if err != nil {
		a.logger.Error("Failed to create interceptor: %v", err)
		return err
	}

	a.interceptor = recorder
	a.logger.Info("✓ Interceptor initialized")
	return nil
}

// initializeBrowser initializes the history browser
func (a *Application) initializeBrowser() error {
	a.logger.Info("Initializing history browser...")

	b := browser.NewBrowser(a.storage)
	a.browser = b
	a.logger.Info("✓ Browser initialized")
	return nil
}

// initializeExecutor initializes the command executor
func (a *Application) initializeExecutor() error {
	a.logger.Info("Initializing command executor...")

	exec := executor.NewExecutor()
	a.executor = exec
	a.logger.Info("✓ Executor initialized")
	return nil
}

// Start starts the application and background services
func (a *Application) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return fmt.Errorf("application is already running")
	}

	a.logger.Info("Starting Command History Tracker...")

	// Start automatic cleanup if enabled
	if a.config.AutoCleanup {
		a.logger.Info("Auto-cleanup enabled (interval: %v, retention: %d days)",
			a.config.CleanupInterval, a.config.RetentionDays)
		go a.runAutoCleanup()
	}

	a.running = true
	a.logger.Info("✓ Application started")
	return nil
}

// Stop stops the application and cleans up resources
func (a *Application) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	a.logger.Info("Stopping Command History Tracker...")

	// Signal cleanup to stop
	close(a.cleanupDone)

	// Wait a moment for cleanup to finish
	time.Sleep(100 * time.Millisecond)

	a.running = false
	a.logger.Info("✓ Application stopped")
	return nil
}

// Shutdown performs graceful shutdown of all components
func (a *Application) Shutdown() error {
	a.logger.Info("Shutting down Command History Tracker...")

	// Stop the application
	if err := a.Stop(); err != nil {
		a.logger.Error("Warning: error stopping application: %v", err)
	}

	// Close all components
	var shutdownErrors []error

	// Close interceptor
	if a.interceptor != nil {
		a.logger.Debug("Closing interceptor...")
		if err := a.interceptor.Close(); err != nil {
			a.logger.Error("Failed to close interceptor: %v", err)
			shutdownErrors = append(shutdownErrors, fmt.Errorf("interceptor: %w", err))
		}
	}

	// Close storage
	if a.storage != nil {
		a.logger.Debug("Closing storage...")
		if err := a.storage.Close(); err != nil {
			a.logger.Error("Failed to close storage: %v", err)
			shutdownErrors = append(shutdownErrors, fmt.Errorf("storage: %w", err))
		}
	}

	if len(shutdownErrors) > 0 {
		a.logger.Error("Shutdown completed with %d error(s)", len(shutdownErrors))
		return fmt.Errorf("shutdown errors: %v", shutdownErrors)
	}

	a.logger.Info("✓ Shutdown complete")
	return nil
}

// runAutoCleanup runs automatic cleanup in the background
func (a *Application) runAutoCleanup() {
	ticker := time.NewTicker(a.config.CleanupInterval)
	defer ticker.Stop()

	a.logger.Debug("Auto-cleanup goroutine started")

	for {
		select {
		case <-ticker.C:
			a.performCleanup()
		case <-a.cleanupDone:
			a.logger.Debug("Auto-cleanup goroutine stopped")
			return
		}
	}
}

// performCleanup performs cleanup of old commands
func (a *Application) performCleanup() {
	a.logger.Info("Running automatic cleanup...")

	if err := a.storage.CleanupOldCommands(a.config.RetentionDays); err != nil {
		a.logger.Error("Cleanup error: %v", err)
		return
	}

	a.logger.Info("✓ Cleanup completed")
}

// GetStorage returns the storage engine
func (a *Application) GetStorage() history.StorageEngine {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.storage
}

// GetBrowser returns the history browser
func (a *Application) GetBrowser() *browser.Browser {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.browser
}

// GetExecutor returns the command executor
func (a *Application) GetExecutor() *executor.Executor {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.executor
}

// GetInterceptor returns the command interceptor
func (a *Application) GetInterceptor() *interceptor.CommandRecorder {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.interceptor
}

// GetConfig returns the application configuration
func (a *Application) GetConfig() *config.Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

// IsRunning returns whether the application is running
func (a *Application) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}

// RecordCommand records a command to history
func (a *Application) RecordCommand() error {
	if a.interceptor == nil {
		return fmt.Errorf("interceptor not initialized")
	}

	return a.interceptor.RecordFromEnvironment()
}

// BrowseHistory launches the interactive history browser
func (a *Application) BrowseHistory(directory string) error {
	if a.browser == nil {
		return fmt.Errorf("browser not initialized")
	}

	if err := a.browser.SetCurrentDirectory(directory); err != nil {
		return err
	}

	return a.browser.ShowDirectoryHistory(directory)
}

// ExecuteCommand executes a command from history
func (a *Application) ExecuteCommand(cmd *history.CommandRecord) error {
	if a.executor == nil {
		return fmt.Errorf("executor not initialized")
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	return a.executor.ExecuteCommand(cmd, currentDir)
}

// GetStatus returns the current application status
func (a *Application) GetStatus() (*Status, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	status := &Status{
		Running: a.running,
		Config:  a.config,
	}

	// Get interceptor status
	if a.interceptor != nil {
		interceptorStatus, err := a.interceptor.GetStatus()
		if err == nil {
			status.InterceptorStatus = interceptorStatus
		}
	}

	// Get storage statistics
	if a.storage != nil {
		dirs, err := a.storage.GetDirectoriesWithHistory()
		if err == nil {
			status.TotalDirectories = len(dirs)
		}

		// Count total commands (approximate)
		for _, dir := range dirs {
			commands, err := a.storage.GetCommandsByDirectory(dir)
			if err == nil {
				status.TotalCommands += len(commands)
			}
		}
	}

	return status, nil
}

// Status represents the current application status
type Status struct {
	Running           bool                         `json:"running"`
	Config            *config.Config               `json:"config"`
	InterceptorStatus *interceptor.ProcessorStatus `json:"interceptor_status,omitempty"`
	TotalDirectories  int                          `json:"total_directories"`
	TotalCommands     int                          `json:"total_commands"`
}
