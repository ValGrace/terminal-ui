package main

import (
	"github.com/ValGrace/command-history-tracker/internal/browser"
	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/ValGrace/command-history-tracker/internal/storage"
	"fmt"
	"os"
	"github.com/spf13/cobra"
)

var browseFlags struct {
	dir    string
	search string
	tree   bool
}

var browseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Browse command history interactively",
	Long: `Launch an interactive terminal UI to browse command history. 
Navigate through commands, search, and execute selected commands.`,
	RunE: runBrowse,
}

func init() {
	browseCmd.Flags().StringVarP(&browseFlags.dir, "dir", "d", "", "Browse history for specific directory")
	browseCmd.Flags().StringVarP(&browseFlags.search, "search", "s", "", "Start with search filter")
	browseCmd.Flags().BoolVarP(&browseFlags.tree, "tree", "t", false, "Show directory tree view")

	rootCmd.AddCommand(browseCmd)
}

func runBrowse(cmd *cobra.Command, args []string) error {
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

	// Create browser
	b := browser.NewBrowser(storageEngine)

	// Determine directory to browse
	dir := browseFlags.dir
	if dir == "" {
		// Use current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		dir = cwd
	}

	// Set current directory
	if err := b.SetCurrentDirectory(dir); err != nil {
		return fmt.Errorf("failed to set directory: %w", err)
	}

	// Launch appropriate view
	if browseFlags.tree {
		return b.ShowDirectoryTree()
	} else if browseFlags.search != "" {
		return b.FilterCommands(browseFlags.search)
	} else {
		return b.ShowDirectoryHistory(dir)
	}
}
