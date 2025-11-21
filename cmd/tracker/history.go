package main

import (
	"github.com/ValGrace/command-history-tracker/internal/browser"
	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/ValGrace/command-history-tracker/internal/storage"
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var historyFlags struct {
	dir           string
	limit         int
	since         string
	shell         string
	noInteractive bool
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show command history",
	Long: `Display command history for the current or specified directory. 
By default, launches an interactive browser. Use --no-interactive for simple list output.`,
	RunE: runHistory,
}

func init() {
	historyCmd.Flags().StringVarP(&historyFlags.dir, "dir", "d", "", "Show history for specific directory")
	historyCmd.Flags().IntVarP(&historyFlags.limit, "limit", "n", 50, "Limit number of commands to show")
	historyCmd.Flags().StringVar(&historyFlags.since, "since", "", "Show commands since time (e.g., '24h', '7d')")
	historyCmd.Flags().StringVar(&historyFlags.shell, "shell", "", "Filter by shell type (powershell, bash, zsh, cmd)")
	historyCmd.Flags().BoolVar(&historyFlags.noInteractive, "no-interactive", false, "Disable interactive mode, print list")

	rootCmd.AddCommand(historyCmd)
}

func runHistory(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg := config.Global()

	// Initialize storage
	storageEngine, err := storage.NewStorageEngine("sqlite", cfg.StoragePath)
	if err != nil {
		return fmt.Errorf("failed to create storage engine: %w", err)
	}

	if err := storageEngine.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer storageEngine.Close()

	// Determine directory
	dir := historyFlags.dir
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		dir = cwd
	}

	// If interactive mode, launch browser
	if !historyFlags.noInteractive {
		b := browser.NewBrowser(storageEngine)
		if err := b.SetCurrentDirectory(dir); err != nil {
			return fmt.Errorf("failed to set directory: %w", err)
		}
		return b.ShowDirectoryHistory(dir)
	}

	// Non-interactive mode: print list
	commands, err := storageEngine.GetCommandsByDirectory(dir)
	if err != nil {
		return fmt.Errorf("failed to get commands: %w", err)
	}

	// Apply filters
	commands = applyHistoryFilters(commands)

	// Apply limit
	if historyFlags.limit > 0 && len(commands) > historyFlags.limit {
		commands = commands[:historyFlags.limit]
	}

	// Print commands
	if len(commands) == 0 {
		fmt.Println("No commands found in history.")
		return nil
	}

	fmt.Printf("Command history for: %s\n", dir)
	fmt.Printf("Found %d command(s)\n\n", len(commands))

	for i, cmd := range commands {
		fmt.Printf("%4d  %s  %s\n", i+1, cmd.Timestamp.Format("2006-01-02 15:04:05"), cmd.Command)
	}

	return nil
}

func applyHistoryFilters(commands []history.CommandRecord) []history.CommandRecord {
	var filtered []history.CommandRecord

	// Parse since duration if provided
	var sinceTime time.Time
	if historyFlags.since != "" {
		duration, err := parseDuration(historyFlags.since)
		if err == nil {
			sinceTime = time.Now().Add(-duration)
		}
	}

	for _, cmd := range commands {
		// Filter by time
		if !sinceTime.IsZero() && cmd.Timestamp.Before(sinceTime) {
			continue
		}

		// Filter by shell
		if historyFlags.shell != "" {
			shellType := parseShellType(historyFlags.shell)
			if cmd.Shell != shellType {
				continue
			}
		}

		filtered = append(filtered, cmd)
	}

	return filtered
}

func parseDuration(s string) (time.Duration, error) {
	// Support formats like "24h", "7d", "30m"
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	unit := s[len(s)-1]
	value := s[:len(s)-1]

	var multiplier time.Duration
	switch unit {
	case 'm':
		multiplier = time.Minute
	case 'h':
		multiplier = time.Hour
	case 'd':
		multiplier = 24 * time.Hour
	case 'w':
		multiplier = 7 * 24 * time.Hour
	default:
		return time.ParseDuration(s)
	}

	var num int
	if _, err := fmt.Sscanf(value, "%d", &num); err != nil {
		return 0, err
	}

	return time.Duration(num) * multiplier, nil
}

func parseShellType(s string) history.ShellType {
	switch s {
	case "powershell", "pwsh":
		return history.PowerShell
	case "bash":
		return history.Bash
	case "zsh":
		return history.Zsh
	case "cmd":
		return history.Cmd
	default:
		return history.Unknown
	}
}
