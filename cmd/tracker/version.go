package main

import (
	"github.com/ValGrace/command-history-tracker/internal/version"
	"fmt"

	"github.com/spf13/cobra"
)

var versionFlags struct {
	short bool
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version information for the Command History Tracker.`,
	Run:   runVersion,
}

func init() {
	versionCmd.Flags().BoolVarP(&versionFlags.short, "short", "s", false, "Show short version only")
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) {
	if versionFlags.short {
		fmt.Println(version.GetVersion())
	} else {
		fmt.Println(version.FullInfo())
	}
}
