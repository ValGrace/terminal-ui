package main

import (
	"github.com/ValGrace/command-history-tracker/internal/interceptor"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show command recording status",
	Long: `Display the current status of command recording including whether tracking 
is enabled, current shell, integration status, and statistics about recorded commands.`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	return interceptor.PrintStatus()
}
