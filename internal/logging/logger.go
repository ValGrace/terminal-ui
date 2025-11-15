package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Logger provides structured logging capabilities
type Logger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	mu          sync.RWMutex
	level       LogLevel
}

// LogLevel represents the logging level
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	ErrorLevel
)

// String returns the string representation of LogLevel
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// New creates a new logger instance
func New(output io.Writer, level LogLevel) *Logger {
	return &Logger{
		infoLogger:  log.New(output, "[INFO] ", log.LstdFlags),
		errorLogger: log.New(output, "[ERROR] ", log.LstdFlags),
		debugLogger: log.New(output, "[DEBUG] ", log.LstdFlags),
		level:       level,
	}
}

// NewFileLogger creates a logger that writes to a file
func NewFileLogger(logPath string, level LogLevel) (*Logger, error) {
	// Ensure log directory exists
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return New(file, level), nil
}

// Default returns the default logger instance
func Default() *Logger {
	once.Do(func() {
		defaultLogger = New(os.Stderr, InfoLevel)
	})
	return defaultLogger
}

// SetDefault sets the default logger
func SetDefault(logger *Logger) {
	defaultLogger = logger
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current logging level
func (l *Logger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.level <= DebugLevel {
		l.debugLogger.Printf(format, v...)
	}
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.level <= InfoLevel {
		l.infoLogger.Printf(format, v...)
	}
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.level <= ErrorLevel {
		l.errorLogger.Printf(format, v...)
	}
}

// Debugf is an alias for Debug
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.Debug(format, v...)
}

// Infof is an alias for Info
func (l *Logger) Infof(format string, v ...interface{}) {
	l.Info(format, v...)
}

// Errorf is an alias for Error
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.Error(format, v...)
}

// WithField returns a logger with a field added to all messages
func (l *Logger) WithField(key, value string) *Logger {
	prefix := fmt.Sprintf("[%s=%s] ", key, value)
	return &Logger{
		infoLogger:  log.New(l.infoLogger.Writer(), l.infoLogger.Prefix()+prefix, l.infoLogger.Flags()),
		errorLogger: log.New(l.errorLogger.Writer(), l.errorLogger.Prefix()+prefix, l.errorLogger.Flags()),
		debugLogger: log.New(l.debugLogger.Writer(), l.debugLogger.Prefix()+prefix, l.debugLogger.Flags()),
		level:       l.level,
	}
}

// Package-level convenience functions

// Debug logs a debug message using the default logger
func Debug(format string, v ...interface{}) {
	Default().Debug(format, v...)
}

// Info logs an info message using the default logger
func Info(format string, v ...interface{}) {
	Default().Info(format, v...)
}

// Error logs an error message using the default logger
func Error(format string, v ...interface{}) {
	Default().Error(format, v...)
}

// SetLevel sets the logging level for the default logger
func SetLevel(level LogLevel) {
	Default().SetLevel(level)
}
