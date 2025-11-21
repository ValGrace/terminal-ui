package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/ValGrace/command-history-tracker/pkg/history"
)

// Config represents the application configuration
type Config struct {
	StoragePath     string              `json:"storage_path"`
	RetentionDays   int                 `json:"retention_days"`
	MaxCommands     int                 `json:"max_commands"`
	EnabledShells   []history.ShellType `json:"enabled_shells"`
	ExcludePatterns []string            `json:"exclude_patterns"`
	AutoCleanup     bool                `json:"auto_cleanup"`
	CleanupInterval time.Duration       `json:"cleanup_interval"`
	DatabaseTimeout time.Duration       `json:"database_timeout"`
	UITheme         string              `json:"ui_theme"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	// Use current directory for database file
	storagePath := "./commands.db"

	return &Config{
		StoragePath:     storagePath,
		RetentionDays:   90,
		MaxCommands:     10000,
		EnabledShells:   []history.ShellType{history.PowerShell, history.Bash, history.Zsh, history.Cmd},
		ExcludePatterns: []string{"cd", "ls", "dir", "pwd", "clear", "exit"},
		AutoCleanup:     true,
		CleanupInterval: 24 * time.Hour,
		DatabaseTimeout: 30 * time.Second,
		UITheme:         "default",
	}
}

// ConfigPath returns the path to the configuration file
func ConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".command-history-tracker")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

// GetConfigPath returns the path to the configuration file (ignoring errors)
func GetConfigPath() string {
	path, _ := ConfigPath()
	return path
}

// LoadConfig loads configuration from file or returns default if file doesn't exist
func LoadConfig() (*Config, error) {
	return Load()
}

// Load loads configuration from file or returns default if file doesn't exist
func Load() (*Config, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return DefaultConfig(), nil
	}

	// If config file doesn't exist, create it with defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultConfig()
		if saveErr := config.Save(); saveErr != nil {
			// If we can't save, just return defaults
			return config, nil
		}
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return DefaultConfig(), nil
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return DefaultConfig(), nil
	}

	// Validate and set defaults for missing fields
	config.validateAndSetDefaults()

	return &config, nil
}

// Save saves the configuration to file
func (c *Config) Save() error {
	configPath, err := ConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// SaveConfig is an alias for Save for consistency
func (c *Config) SaveConfig() error {
	return c.Save()
}

// validateAndSetDefaults ensures all configuration fields have valid values
func (c *Config) validateAndSetDefaults() {
	defaults := DefaultConfig()

	if c.StoragePath == "" {
		c.StoragePath = defaults.StoragePath
	}
	if c.RetentionDays <= 0 {
		c.RetentionDays = defaults.RetentionDays
	}
	if c.MaxCommands <= 0 {
		c.MaxCommands = defaults.MaxCommands
	}
	if len(c.EnabledShells) == 0 {
		c.EnabledShells = defaults.EnabledShells
	}
	// Fix any Unknown shell types
	for i, shell := range c.EnabledShells {
		if shell == history.Unknown {
			c.EnabledShells[i] = history.PowerShell // Default to PowerShell
		}
	}
	if c.CleanupInterval == 0 {
		c.CleanupInterval = defaults.CleanupInterval
	}
	if c.DatabaseTimeout == 0 {
		c.DatabaseTimeout = defaults.DatabaseTimeout
	}
	if c.UITheme == "" {
		c.UITheme = defaults.UITheme
	}
}

// IsShellEnabled checks if a shell type is enabled in configuration
func (c *Config) IsShellEnabled(shell history.ShellType) bool {
	for _, enabled := range c.EnabledShells {
		if enabled == shell {
			return true
		}
	}
	return false
}

// ShouldExcludeCommand checks if a command should be excluded from recording
func (c *Config) ShouldExcludeCommand(command string) bool {
	for _, pattern := range c.ExcludePatterns {
		if command == pattern {
			return true
		}
	}
	return false
}

// Global configuration instance
var globalConfig *Config

// SetGlobal sets the global configuration instance
func SetGlobal(config *Config) {
	globalConfig = config
}

// Global returns the global configuration instance
func Global() *Config {
	if globalConfig == nil {
		globalConfig = DefaultConfig()
	}
	return globalConfig
}

// Validate checks if the configuration has valid values
func (c *Config) Validate() error {
	if c.StoragePath == "" {
		return &ConfigValidationError{Field: "StoragePath", Message: "Storage path cannot be empty"}
	}
	if c.RetentionDays < 0 {
		return &ConfigValidationError{Field: "RetentionDays", Message: "Retention days cannot be negative"}
	}
	if c.MaxCommands < 0 {
		return &ConfigValidationError{Field: "MaxCommands", Message: "Max commands cannot be negative"}
	}
	if c.CleanupInterval < 0 {
		return &ConfigValidationError{Field: "CleanupInterval", Message: "Cleanup interval cannot be negative"}
	}
	if c.DatabaseTimeout < 0 {
		return &ConfigValidationError{Field: "DatabaseTimeout", Message: "Database timeout cannot be negative"}
	}

	// Validate shell types
	for _, shell := range c.EnabledShells {
		if shell <= history.Unknown || shell > history.Cmd {
			return &ConfigValidationError{Field: "EnabledShells", Message: "Invalid shell type in enabled shells"}
		}
	}

	return nil
}

// IsValid checks if the configuration is valid
func (c *Config) IsValid() bool {
	return c.Validate() == nil
}

// ConfigValidationError represents a configuration validation error
type ConfigValidationError struct {
	Field   string
	Message string
}

func (e *ConfigValidationError) Error() string {
	return "config." + e.Field + ": " + e.Message
}
