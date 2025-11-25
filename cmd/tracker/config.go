package main

import (
	"bufio"
	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var configFlags struct {
	get      string
	set      string
	list     bool
	reset    bool
	edit     bool
	showPath bool
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage tracker configuration",
	Long: `View and modify tracker configuration settings.
Use subcommands or flags to get, set, or list configuration values.`,
	RunE: runConfig,
}

func init() {
	configCmd.Flags().StringVar(&configFlags.get, "get", "", "Get configuration value (e.g., 'retention_days')")
	configCmd.Flags().StringVar(&configFlags.set, "set", "", "Set configuration value (format: 'key=value')")
	configCmd.Flags().BoolVarP(&configFlags.list, "list", "l", false, "List all configuration values")
	configCmd.Flags().BoolVar(&configFlags.reset, "reset", false, "Reset configuration to defaults")
	configCmd.Flags().BoolVarP(&configFlags.edit, "edit", "e", false, "Edit configuration interactively")
	configCmd.Flags().BoolVar(&configFlags.showPath, "path", false, "Show configuration file path")

	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	// Show config path
	if configFlags.showPath {
		path := config.GetConfigPath()
		fmt.Printf("Configuration file: %s\n", path)
		return nil
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Reset to defaults
	if configFlags.reset {
		return resetConfig()
	}

	// Edit interactively
	if configFlags.edit {
		return editConfigInteractive(cfg)
	}

	// Get specific value
	if configFlags.get != "" {
		return getConfigValue(cfg, configFlags.get)
	}

	// Set specific value
	if configFlags.set != "" {
		return setConfigValue(cfg, configFlags.set)
	}

	// List all values (default behavior)
	if configFlags.list || (configFlags.get == "" && configFlags.set == "") {
		return listConfig(cfg)
	}

	return nil
}

func listConfig(cfg *config.Config) error {
	fmt.Println("=== Command History Tracker Configuration ===")

	fmt.Printf("Storage Path:       %s\n", cfg.StoragePath)
	fmt.Printf("Retention Days:     %d\n", cfg.RetentionDays)
	fmt.Printf("Max Commands:       %d\n", cfg.MaxCommands)
	fmt.Printf("Auto Cleanup:       %v\n", cfg.AutoCleanup)
	fmt.Printf("Cleanup Interval:   %s\n", cfg.CleanupInterval)
	fmt.Printf("Database Timeout:   %s\n", cfg.DatabaseTimeout)
	fmt.Printf("UI Theme:           %s\n", cfg.UITheme)

	fmt.Printf("\nEnabled Shells:     ")
	for i, shell := range cfg.EnabledShells {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(shellTypeToString(shell))
	}
	fmt.Println()

	fmt.Printf("\nExclude Patterns:   ")
	if len(cfg.ExcludePatterns) == 0 {
		fmt.Println("(none)")
	} else {
		fmt.Println()
		for _, pattern := range cfg.ExcludePatterns {
			fmt.Printf("  - %s\n", pattern)
		}
	}

	fmt.Printf("\nConfiguration file: %s\n", config.GetConfigPath())

	return nil
}

func getConfigValue(cfg *config.Config, key string) error {
	key = strings.ToLower(strings.ReplaceAll(key, "-", "_"))

	switch key {
	case "storage_path", "storagepath":
		fmt.Println(cfg.StoragePath)
	case "retention_days", "retentiondays":
		fmt.Println(cfg.RetentionDays)
	case "max_commands", "maxcommands":
		fmt.Println(cfg.MaxCommands)
	case "auto_cleanup", "autocleanup":
		fmt.Println(cfg.AutoCleanup)
	case "cleanup_interval", "cleanupinterval":
		fmt.Println(cfg.CleanupInterval)
	case "database_timeout", "databasetimeout":
		fmt.Println(cfg.DatabaseTimeout)
	case "ui_theme", "uitheme":
		fmt.Println(cfg.UITheme)
	case "enabled_shells", "enabledshells":
		for i, shell := range cfg.EnabledShells {
			if i > 0 {
				fmt.Print(",")
			}
			fmt.Print(shellTypeToString(shell))
		}
		fmt.Println()
	case "exclude_patterns", "excludepatterns":
		for _, pattern := range cfg.ExcludePatterns {
			fmt.Println(pattern)
		}
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	return nil
}

func setConfigValue(cfg *config.Config, keyValue string) error {
	parts := strings.SplitN(keyValue, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format, use: key=value")
	}

	key := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(parts[0]), "-", "_"))
	value := strings.TrimSpace(parts[1])

	switch key {
	case "storage_path", "storagepath":
		cfg.StoragePath = value
	case "retention_days", "retentiondays":
		days, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid retention_days value: %w", err)
		}
		cfg.RetentionDays = days
	case "max_commands", "maxcommands":
		max, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid max_commands value: %w", err)
		}
		cfg.MaxCommands = max
	case "auto_cleanup", "autocleanup":
		autoCleanup, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid auto_cleanup value: %w", err)
		}
		cfg.AutoCleanup = autoCleanup
	case "ui_theme", "uitheme":
		cfg.UITheme = value
	case "exclude_patterns", "excludepatterns":
		patterns := strings.Split(value, ",")
		for i := range patterns {
			patterns[i] = strings.TrimSpace(patterns[i])
		}
		cfg.ExcludePatterns = patterns
	default:
		return fmt.Errorf("unknown or read-only configuration key: %s", key)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Save configuration
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("✓ Configuration updated: %s = %s\n", key, value)
	return nil
}

func resetConfig() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("⚠ This will reset all configuration to default values.")
	fmt.Print("Are you sure? (y/N): ")
	response, err := reader.ReadString('\n')
	if err != nil {
		response = ""
	}
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		fmt.Println("Reset cancelled.")
		return nil
	}

	// Create default configuration
	cfg := config.DefaultConfig()

	// Save configuration
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println("✓ Configuration reset to defaults")
	return listConfig(cfg)
}

func editConfigInteractive(cfg *config.Config) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Interactive Configuration Editor ===")
	fmt.Println("Press Enter to keep current value, or enter new value.")

	// Edit retention days
	fmt.Printf("Retention days [%d]: ", cfg.RetentionDays)
	if input := readLine(reader); input != "" {
		if days, err := strconv.Atoi(input); err == nil && days > 0 {
			cfg.RetentionDays = days
		} else {
			fmt.Println("  Invalid value, keeping current")
		}
	}

	// Edit max commands
	fmt.Printf("Max commands per directory [%d]: ", cfg.MaxCommands)
	if input := readLine(reader); input != "" {
		if max, err := strconv.Atoi(input); err == nil && max > 0 {
			cfg.MaxCommands = max
		} else {
			fmt.Println("  Invalid value, keeping current")
		}
	}

	// Edit auto cleanup
	fmt.Printf("Auto cleanup [%v] (true/false): ", cfg.AutoCleanup)
	if input := readLine(reader); input != "" {
		if autoCleanup, err := strconv.ParseBool(input); err == nil {
			cfg.AutoCleanup = autoCleanup
		} else {
			fmt.Println("  Invalid value, keeping current")
		}
	}

	// Edit UI theme
	fmt.Printf("UI theme [%s]: ", cfg.UITheme)
	if input := readLine(reader); input != "" {
		cfg.UITheme = input
	}

	// Edit exclude patterns
	fmt.Printf("Exclude patterns (comma-separated) [%s]: ", strings.Join(cfg.ExcludePatterns, ", "))
	if input := readLine(reader); input != "" {
		patterns := strings.Split(input, ",")
		for i := range patterns {
			patterns[i] = strings.TrimSpace(patterns[i])
		}
		cfg.ExcludePatterns = patterns
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Save configuration
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println("\n✓ Configuration saved successfully")
	return nil
}

func readLine(reader *bufio.Reader) string {
	line, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimSpace(line)
}

func shellTypeToString(shell history.ShellType) string {
	switch shell {
	case history.PowerShell:
		return "powershell"
	case history.Bash:
		return "bash"
	case history.Zsh:
		return "zsh"
	case history.Cmd:
		return "cmd"
	default:
		return "unknown"
	}
}
