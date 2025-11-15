package errors

import (
	"fmt"
)

// ErrorType represents different categories of errors
type ErrorType int

const (
	StorageError ErrorType = iota
	ShellError
	ExecutionError
	ConfigError
	ValidationError
	NetworkError
)

// String returns the string representation of ErrorType
func (e ErrorType) String() string {
	switch e {
	case StorageError:
		return "storage"
	case ShellError:
		return "shell"
	case ExecutionError:
		return "execution"
	case ConfigError:
		return "config"
	case ValidationError:
		return "validation"
	case NetworkError:
		return "network"
	default:
		return "unknown"
	}
}

// HistoryError represents a structured error with context
type HistoryError struct {
	Type    ErrorType              `json:"type"`
	Message string                 `json:"message"`
	Cause   error                  `json:"-"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface
func (e *HistoryError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s error: %s (caused by: %v)", e.Type.String(), e.Message, e.Cause)
	}
	return fmt.Sprintf("%s error: %s", e.Type.String(), e.Message)
}

// Unwrap returns the underlying error
func (e *HistoryError) Unwrap() error {
	return e.Cause
}

// WithContext adds context information to the error
func (e *HistoryError) WithContext(key string, value interface{}) *HistoryError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// NewStorageError creates a new storage-related error
func NewStorageError(message string, cause error) *HistoryError {
	return &HistoryError{
		Type:    StorageError,
		Message: message,
		Cause:   cause,
	}
}

// NewShellError creates a new shell-related error
func NewShellError(message string, cause error) *HistoryError {
	return &HistoryError{
		Type:    ShellError,
		Message: message,
		Cause:   cause,
	}
}

// NewExecutionError creates a new execution-related error
func NewExecutionError(message string, cause error) *HistoryError {
	return &HistoryError{
		Type:    ExecutionError,
		Message: message,
		Cause:   cause,
	}
}

// NewConfigError creates a new configuration-related error
func NewConfigError(message string, cause error) *HistoryError {
	return &HistoryError{
		Type:    ConfigError,
		Message: message,
		Cause:   cause,
	}
}

// NewValidationError creates a new validation-related error
func NewValidationError(message string, cause error) *HistoryError {
	return &HistoryError{
		Type:    ValidationError,
		Message: message,
		Cause:   cause,
	}
}

// IsType checks if an error is of a specific type
func IsType(err error, errorType ErrorType) bool {
	if histErr, ok := err.(*HistoryError); ok {
		return histErr.Type == errorType
	}
	return false
}
