package main

import (
	"github.com/ValGrace/command-history-tracker/internal/browser"
	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/ValGrace/command-history-tracker/internal/storage"
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var searchFlags struct {
	dir           string
	allDirs       bool
	caseSensitive bool
	limit         int
	noInteractive bool
}

var searchCmd = &cobra.Command{
	Use:   "search [pattern]",
	Short: "Search command history",
	Long: `Search for commands matching a pattern in command history.
By default, launches an interactive browser with search results.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearch,
}

func init() {
	searchCmd.Flags().StringVarP(&searchFlags.dir, "dir", "d", "", "Search in specific directory")
	searchCmd.Flags().BoolVarP(&searchFlags.allDirs, "all", "a", false, "Search across all directories")
	searchCmd.Flags().BoolVarP(&searchFlags.caseSensitive, "case-sensitive", "c", false, "Case-sensitive search")
	searchCmd.Flags().IntVarP(&searchFlags.limit, "limit", "n", 50, "Limit number of results")
	searchCmd.Flags().BoolVar(&searchFlags.noInteractive, "no-interactive", false, "Disable interactive mode, print list")

	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	pattern := strings.Join(args, " ")

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
	dir := searchFlags.dir
	if dir == "" && !searchFlags.allDirs {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		dir = cwd
	}

	// If interactive mode, launch browser with search
	if !searchFlags.noInteractive {
		b := browser.NewBrowser(storageEngine)
		if dir != "" {
			if err := b.SetCurrentDirectory(dir); err != nil {
				return fmt.Errorf("failed to set directory: %w", err)
			}
		}
		return b.FilterCommands(pattern)
	}

	// Non-interactive mode: search and print results
	var commands []history.CommandRecord

	if searchFlags.allDirs {
		// Search across all directories
		dirs, err := storageEngine.GetDirectoriesWithHistory()
		if err != nil {
			return fmt.Errorf("failed to get directories: %w", err)
		}

		for _, d := range dirs {
			results, err := storageEngine.SearchCommands(pattern, d)
			if err != nil {
				continue
			}
			commands = append(commands, results...)
		}
	} else {
		// Search in specific directory
		commands, err = storageEngine.SearchCommands(pattern, dir)
		if err != nil {
			return fmt.Errorf("failed to search commands: %w", err)
		}
	}

	// Apply case sensitivity filter if needed
	if !searchFlags.caseSensitive {
		commands = filterCaseInsensitive(commands, pattern)
	}

	// Apply limit
	if searchFlags.limit > 0 && len(commands) > searchFlags.limit {
		commands = commands[:searchFlags.limit]
	}

	// Print results
	if len(commands) == 0 {
		fmt.Printf("No commands found matching: %s\n", pattern)
		return nil
	}

	fmt.Printf("Search results for: %s\n", pattern)
	if searchFlags.allDirs {
		fmt.Println("Searching across all directories")
	} else {
		fmt.Printf("Directory: %s\n", dir)
	}
	fmt.Printf("Found %d command(s)\n\n", len(commands))

	for i, cmd := range commands {
		if searchFlags.allDirs {
			fmt.Printf("%4d  %s  [%s]  %s\n", i+1, cmd.Timestamp.Format("2006-01-02 15:04:05"), cmd.Directory, cmd.Command)
		} else {
			fmt.Printf("%4d  %s  %s\n", i+1, cmd.Timestamp.Format("2006-01-02 15:04:05"), cmd.Command)
		}
	}

	return nil
}

func filterCaseInsensitive(commands []history.CommandRecord, pattern string) []history.CommandRecord {
	lowerPattern := strings.ToLower(pattern)
	var filtered []history.CommandRecord

	for _, cmd := range commands {
		if strings.Contains(strings.ToLower(cmd.Command), lowerPattern) {
			filtered = append(filtered, cmd)
		}
	}

	return filtered
}
