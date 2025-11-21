package main

import (
	"github.com/ValGrace/command-history-tracker/internal/interceptor"
	"fmt"

	"github.com/spf13/cobra"
)

var startFlags struct {
	background bool
	force      bool
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start command recording",
	Long: `Start recording terminal commands in the background. This sets up shell 
hooks to automatically capture all executed commands.`,
	RunE: runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop command recording",
	Long: `Stop recording terminal commands and remove shell hooks. This will 
disable automatic command capture but preserve existing history.`,
	RunE: runStop,
}

func init() {
	startCmd.Flags().BoolVarP(&startFlags.background, "background", "b", false, "Run in background mode")
	startCmd.Flags().BoolVarP(&startFlags.force, "force", "f", false, "Force start even if already running")

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	fmt.Println("Starting command recording...")

	// Setup recording (installs shell hooks)
	if err := interceptor.SetupRecording(); err != nil {
		return fmt.Errorf("failed to start recording: %w", err)
	}

	fmt.Println("✓ Command recording started successfully!")
	fmt.Println("\nCommands will now be automatically recorded.")
	fmt.Println("Restart your shell or source your shell configuration to activate.")
	fmt.Println("\nUseful commands:")
	fmt.Println("  tracker browse    - Browse command history")
	fmt.Println("  tracker status    - Check recording status")
	fmt.Println("  tracker stop      - Stop recording")

	return nil
}

func runStop(cmd *cobra.Command, args []string) error {
	fmt.Println("Stopping command recording...")

	// Remove recording hooks
	if err := interceptor.RemoveRecording(); err != nil {
		return fmt.Errorf("failed to stop recording: %w", err)
	}

	fmt.Println("✓ Command recording stopped successfully!")
	fmt.Println("\nShell hooks have been removed.")
	fmt.Println("Restart your shell or source your shell configuration to deactivate.")
	fmt.Println("\nTo start recording again, run: tracker start")

	return nil
}
