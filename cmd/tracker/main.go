package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ValGrace/command-history-tracker/internal/app"
	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/spf13/cobra"
)

var (
	globalApp *app.Application
	rootCmd   = &cobra.Command{
		Use:   "tracker",
		Short: "Command History Tracker - Track and manage terminal command history",
		Long: `A Go package that provides comprehensive terminal command history tracking 
and management capabilities. Automatically records commands and provides an 
interactive interface for browsing and re-executing them.`,
		PersistentPreRunE: initializeApp,
	}
)

func main() {
	// Setup signal handling for graceful shutdown
	setupSignalHandling()

	// Check for first run and prompt for auto-setup if no command specified
	if len(os.Args) == 1 {
		// No command specified, check if this is first run
		if err := promptForAutoSetup(); err != nil {
			fmt.Fprintf(os.Stderr, "Setup error: %v\n", err)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		cleanup()
		os.Exit(1)
	}

	cleanup()
}

// initializeApp initializes the application for commands that need it
func initializeApp(cmd *cobra.Command, args []string) error {
	// Skip initialization for commands that don't need it
	skipCommands := map[string]bool{
		"help":    true,
		"version": true,
	}

	if skipCommands[cmd.Name()] {
		return nil
	}

	// Create and initialize application
	application, err := app.New()
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}

	if err := application.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	globalApp = application

	// Set global configuration for backward compatibility
	config.SetGlobal(application.GetConfig())

	return nil
}

// setupSignalHandling sets up graceful shutdown on interrupt signals
func setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cleanup()
		os.Exit(0)
	}()
}

// cleanup performs cleanup operations
func cleanup() {
	if globalApp != nil {
		if err := globalApp.Shutdown(); err != nil {
			fmt.Fprintf(os.Stderr, "Shutdown error: %v\n", err)
		}
	}
}
