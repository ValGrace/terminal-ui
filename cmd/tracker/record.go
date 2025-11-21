package main

import (
	"github.com/ValGrace/command-history-tracker/internal/interceptor"
	"fmt"

	"github.com/spf13/cobra"
)

var recordCmd = &cobra.Command{
	Use:   "record",
	Short: "Record a command to history",
	Long: `Record a command to the command history database. This command is typically 
called automatically by shell hooks to capture executed commands.`,
	RunE: runRecord,
}

var recordFlags struct {
	fromArgs bool
	test     bool
}

func init() {
	recordCmd.Flags().BoolVar(&recordFlags.fromArgs, "from-args", false, "Record command from command line arguments")
	recordCmd.Flags().BoolVar(&recordFlags.test, "test", false, "Test command recording functionality")

	rootCmd.AddCommand(recordCmd)
}

func runRecord(cmd *cobra.Command, args []string) error {
	// Handle test mode
	if recordFlags.test {
		return interceptor.TestRecording()
	}

	// Handle recording from arguments
	if recordFlags.fromArgs {
		if len(args) == 0 {
			return fmt.Errorf("no arguments provided for recording")
		}
		return interceptor.RecordCommandWithArgs(args)
	}

	// Default: record from environment variables
	if err := interceptor.RecordCommand(); err != nil {
		// Don't print error to stderr as it might interfere with shell output
		// Instead, log to a file or silently fail
		return nil
	}

	return nil
}
