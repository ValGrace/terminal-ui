package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/ValGrace/command-history-tracker/internal/interceptor"
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"github.com/ValGrace/command-history-tracker/pkg/shell"

	"github.com/spf13/cobra"
)

var (
	setupInteractive bool
	setupForce       bool
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup command recording for current shell",
	Long: `Setup command recording by installing shell hooks for the current shell. 
This will modify your shell configuration files to automatically capture commands.`,
	RunE: runSetup,
}

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove command recording hooks",
	Long: `Remove command recording hooks from your shell configuration. This will 
stop automatic command capture but preserve existing command history.`,
	RunE: runRemove,
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Cleanup old command history",
	Long: `Remove old command history entries based on the configured retention policy. 
This helps keep the command history database size manageable.`,
	RunE: runCleanup,
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall command history tracker",
	Long: `Completely uninstall the command history tracker, including:
- Removing shell hooks from configuration files
- Optionally deleting command history database
- Cleaning up all tracker-related files`,
	RunE: runUninstall,
}

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify tracker installation and configuration",
	Long: `Verify that the command history tracker is properly installed and configured.
This checks shell integration, configuration validity, and storage accessibility.`,
	RunE: runVerify,
}

var manualCmd = &cobra.Command{
	Use:   "manual",
	Short: "Show manual installation instructions",
	Long: `Display manual installation instructions for setting up shell integration.
Use this if automatic setup fails or if you prefer to configure manually.`,
	RunE: runManual,
}

func init() {
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(cleanupCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(verifyCmd)
	setupCmd.AddCommand(manualCmd)

	setupCmd.Flags().BoolVarP(&setupInteractive, "interactive", "i", false, "Run interactive setup wizard")
	setupCmd.Flags().BoolVarP(&setupForce, "force", "f", false, "Force setup even if already configured")
}

func runManual(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Manual Installation Instructions ===")
	fmt.Println()

	// Detect shell
	detector := shell.NewDetector()
	shellType, err := detector.DetectShell()
	if err != nil {
		fmt.Printf("Could not detect shell: %v\n", err)
		fmt.Println("Showing instructions for all supported shells...")
		shellType = history.Unknown
	} else {
		fmt.Printf("Detected shell: %s\n\n", shellType)
	}

	// Get integration script
	integrator := shell.NewIntegrator()

	if shellType != history.Unknown {
		// Show instructions for detected shell
		showManualInstructionsForShell(shellType, integrator)
	} else {
		// Show instructions for all shells
		platform := shell.NewPlatformAbstraction()
		for _, shell := range platform.GetSupportedShells() {
			showManualInstructionsForShell(shell, integrator)
			fmt.Println()
		}
	}

	return nil
}

func showManualInstructionsForShell(shellType history.ShellType, integrator *shell.Integrator) {
	platform := shell.NewPlatformAbstraction()
	configPath, err := platform.GetShellConfigPath(shellType)
	if err != nil {
		fmt.Printf("Could not determine config path for %s: %v\n", shellType, err)
		return
	}

	script, err := integrator.GetIntegrationScript(shellType)
	if err != nil {
		fmt.Printf("Could not get integration script for %s: %v\n", shellType, err)
		return
	}

	fmt.Printf("--- %s ---\n", shellType)
	fmt.Printf("Configuration file: %s\n\n", configPath)
	fmt.Println("Add the following to your shell configuration file:")
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println(script)
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()
	fmt.Println("After adding the script, restart your shell or run:")
	printShellRestartInstructions(shellType)
}

// autoSetup performs automatic setup without user interaction
func autoSetup() error {
	fmt.Println("Performing automatic setup...")

	// Detect current shell
	detector := shell.NewDetector()
	shellType, err := detector.DetectShell()
	if err != nil {
		return fmt.Errorf("failed to detect shell: %w", err)
	}

	fmt.Printf("Detected shell: %s\n", shellType)

	// Check if shell is supported
	if !detector.IsShellSupported(shellType) {
		fmt.Printf("\nShell %s is not fully supported on this platform.\n", shellType)
		fmt.Println("Supported shells on your platform:")
		platform := shell.NewPlatformAbstraction()
		for _, s := range platform.GetSupportedShells() {
			fmt.Printf("  - %s\n", s)
		}
		return fmt.Errorf("unsupported shell")
	}

	// Check if already configured
	integrator := shell.NewIntegrator()
	isInstalled, err := integrator.IsInstalled(shellType)
	if err != nil {
		return fmt.Errorf("failed to check installation status: %w", err)
	}

	if isInstalled {
		fmt.Println("✓ Command recording is already configured.")
		fmt.Println("\nYou can:")
		fmt.Println("  - Check status with: tracker status")
		fmt.Println("  - Browse history with: tracker browse")
		fmt.Println("  - Reconfigure with: tracker setup --force")
		return nil
	}

	// Create default configuration
	fmt.Println("\nCreating configuration...")
	cfg := config.DefaultConfig()
	if err := cfg.Save(); err != nil {
		fmt.Printf("Warning: Failed to save configuration: %v\n", err)
		fmt.Println("Continuing with in-memory configuration...")
	} else {
		fmt.Println("✓ Configuration created")
		fmt.Printf("  - Storage: %s\n", cfg.StoragePath)
		fmt.Printf("  - Retention: %d days\n", cfg.RetentionDays)
		fmt.Printf("  - Max commands: %d per directory\n", cfg.MaxCommands)
	}

	// Get shell config path for user information
	platform := shell.NewPlatformAbstraction()
	configPath, err := platform.GetShellConfigPath(shellType)
	if err != nil {
		fmt.Printf("Warning: Could not determine shell config path: %v\n", err)
	} else {
		fmt.Printf("\nShell configuration file: %s\n", configPath)

		// Ensure shell config file exists
		if err := ensureShellConfigExists(shellType, configPath); err != nil {
			fmt.Printf("Warning: Could not create shell config file: %v\n", err)
		}
	}

	// Create backup of shell config before modification
	fmt.Println("\nBacking up shell configuration...")
	if err := backupShellConfig(shellType); err != nil {
		fmt.Printf("Note: No existing shell config to backup (this is normal for new shells)\n")
	} else {
		fmt.Println("✓ Backup created (.backup extension)")
	}

	// Setup recording
	fmt.Println("\nInstalling shell hooks...")
	if err := interceptor.SetupRecording(); err != nil {
		fmt.Printf("\n✗ Failed to setup recording: %v\n", err)
		fmt.Println("\nTroubleshooting:")
		fmt.Println("  1. Check that you have write permissions for your shell config file")
		fmt.Println("  2. Try running 'tracker setup --interactive' for more options")
		fmt.Println("  3. Manually add hooks by running 'tracker setup --help'")
		return fmt.Errorf("setup failed")
	}
	fmt.Println("✓ Shell hooks installed")

	// Verify installation
	fmt.Println("\nVerifying installation...")
	if err := verifySetup(shellType); err != nil {
		fmt.Printf("⚠ Setup verification failed: %v\n", err)
		fmt.Println("\nSetup may not be complete. You can:")
		fmt.Println("  - Run 'tracker verify' to check installation")
		fmt.Println("  - Run 'tracker setup --interactive' for troubleshooting")
		fmt.Println("  - Check the documentation for manual setup")
	} else {
		fmt.Println("✓ Installation verified")
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Setup Complete!")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("\nNext steps:")
	printShellRestartInstructions(shellType)
	fmt.Println("  2. Start using your terminal - commands will be recorded automatically")
	fmt.Println("  3. Browse history with: tracker browse")
	fmt.Println("  4. Check status with: tracker status")
	fmt.Println("\nTips:")
	fmt.Println("  - Run 'tracker setup --interactive' for custom configuration")
	fmt.Println("  - Run 'tracker verify' to check installation at any time")
	fmt.Println("  - Run 'tracker uninstall' to remove the tracker")

	return nil
}

// ensureShellConfigExists creates the shell configuration file if it doesn't exist
func ensureShellConfigExists(shellType history.ShellType, configPath string) error {
	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return nil // File exists
	} else if !os.IsNotExist(err) {
		return err // Some other error
	}

	// Create directory if needed
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create empty config file with appropriate header comment
	var header string
	switch shellType {
	case history.PowerShell:
		header = "# PowerShell Profile\n"
	case history.Bash:
		header = "# Bash Configuration\n"
	case history.Zsh:
		header = "# Zsh Configuration\n"
	case history.Cmd:
		header = "REM Command Prompt Configuration\n"
	default:
		header = "# Shell Configuration\n"
	}

	if err := os.WriteFile(configPath, []byte(header), 0644); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	fmt.Printf("✓ Created shell config file: %s\n", configPath)
	return nil
}

// verifySetup checks if the setup was successful
func verifySetup(shellType history.ShellType) error {
	// Check if integration is installed
	integrator := shell.NewIntegrator()
	isInstalled, err := integrator.IsInstalled(shellType)
	if err != nil {
		return fmt.Errorf("failed to verify installation: %w", err)
	}

	if !isInstalled {
		return fmt.Errorf("shell hooks not found in configuration file")
	}

	// Check if config file exists and is valid
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Check if storage path is accessible
	storageDir := filepath.Dir(cfg.StoragePath)
	if storageDir != "" && storageDir != "." {
		if err := os.MkdirAll(storageDir, 0755); err != nil {
			return fmt.Errorf("storage directory not accessible: %w", err)
		}
	}

	return nil
}

func runVerify(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Verifying Command History Tracker Installation ===")

	hasErrors := false

	// Check shell detection
	fmt.Println("1. Checking shell detection...")
	detector := shell.NewDetector()
	shellType, err := detector.DetectShell()
	if err != nil {
		fmt.Printf("   ✗ Failed to detect shell: %v\n", err)
		hasErrors = true
	} else {
		fmt.Printf("   ✓ Detected shell: %s\n", shellType)

		// Check if shell is supported
		if !detector.IsShellSupported(shellType) {
			fmt.Printf("   ✗ Shell %s is not supported on this platform\n", shellType)
			hasErrors = true
		} else {
			fmt.Printf("   ✓ Shell is supported\n")
		}
	}

	// Check shell integration
	fmt.Println("\n2. Checking shell integration...")
	integrator := shell.NewIntegrator()
	isInstalled, err := integrator.IsInstalled(shellType)
	if err != nil {
		fmt.Printf("   ✗ Failed to check integration: %v\n", err)
		hasErrors = true
	} else if !isInstalled {
		fmt.Printf("   ✗ Shell hooks not installed\n")
		fmt.Printf("   → Run 'tracker setup' to install\n")
		hasErrors = true
	} else {
		fmt.Printf("   ✓ Shell hooks installed\n")

		// Check shell config file
		platform := shell.NewPlatformAbstraction()
		configPath, err := platform.GetShellConfigPath(shellType)
		if err != nil {
			fmt.Printf("   ✗ Failed to get shell config path: %v\n", err)
			hasErrors = true
		} else {
			fmt.Printf("   ✓ Shell config: %s\n", configPath)
		}
	}

	// Check configuration
	fmt.Println("\n3. Checking configuration...")
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("   ✗ Failed to load configuration: %v\n", err)
		hasErrors = true
	} else {
		fmt.Printf("   ✓ Configuration loaded\n")

		// Validate configuration
		if err := cfg.Validate(); err != nil {
			fmt.Printf("   ✗ Configuration validation failed: %v\n", err)
			hasErrors = true
		} else {
			fmt.Printf("   ✓ Configuration is valid\n")
		}

		// Display key settings
		fmt.Printf("   - Storage path: %s\n", cfg.StoragePath)
		fmt.Printf("   - Retention: %d days\n", cfg.RetentionDays)
		fmt.Printf("   - Max commands: %d\n", cfg.MaxCommands)
		fmt.Printf("   - Auto cleanup: %v\n", cfg.AutoCleanup)
	}

	// Check storage accessibility
	fmt.Println("\n4. Checking storage...")
	if cfg != nil {
		storageDir := filepath.Dir(cfg.StoragePath)
		if storageDir == "" || storageDir == "." {
			storageDir, _ = os.Getwd()
		}

		// Check if directory exists or can be created
		if err := os.MkdirAll(storageDir, 0755); err != nil {
			fmt.Printf("   ✗ Storage directory not accessible: %v\n", err)
			hasErrors = true
		} else {
			fmt.Printf("   ✓ Storage directory accessible: %s\n", storageDir)
		}

		// Check if database file exists
		if _, err := os.Stat(cfg.StoragePath); err == nil {
			fmt.Printf("   ✓ Database file exists: %s\n", cfg.StoragePath)
		} else if os.IsNotExist(err) {
			fmt.Printf("   - Database file will be created on first command\n")
		} else {
			fmt.Printf("   ✗ Error checking database: %v\n", err)
			hasErrors = true
		}
	}

	// Check tracker executable
	fmt.Println("\n5. Checking tracker executable...")
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("   ✗ Failed to get executable path: %v\n", err)
		hasErrors = true
	} else {
		fmt.Printf("   ✓ Tracker executable: %s\n", execPath)
	}

	// Summary
	fmt.Println("\n=== Verification Summary ===")
	if hasErrors {
		fmt.Println("✗ Some checks failed. Please review the errors above.")
		fmt.Println("\nTroubleshooting:")
		fmt.Println("  - Run 'tracker setup --interactive' to reconfigure")
		fmt.Println("  - Check that your shell is supported")
		fmt.Println("  - Ensure you have write permissions for config and storage directories")
		return fmt.Errorf("verification failed")
	}

	fmt.Println("✓ All checks passed! The tracker is properly configured.")
	fmt.Println("\nYou can now:")
	fmt.Println("  - Use your terminal normally (commands will be recorded)")
	fmt.Println("  - Browse history with: tracker browse")
	fmt.Println("  - Check status with: tracker status")

	return nil
}

// isFirstRun checks if this is the first time the tracker is being run
func isFirstRun() bool {
	// Check if config file exists
	configPath := config.GetConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		// Config exists, check if it has the first_run_complete flag
		cfg, err := config.Load()
		if err == nil && cfg != nil {
			// If config loads successfully, consider it not first run
			return false
		}
	}

	// Check if any shell integration is installed
	detector := shell.NewDetector()
	integrator := shell.NewIntegrator()

	shellType, err := detector.DetectShell()
	if err != nil {
		return true
	}

	isInstalled, err := integrator.IsInstalled(shellType)
	if err != nil {
		return true
	}

	return !isInstalled
}

// promptForAutoSetup prompts the user to run setup on first run
func promptForAutoSetup() error {
	if !isFirstRun() {
		return nil
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Welcome to Command History Tracker ===")
	fmt.Println("It looks like this is your first time running the tracker.")
	fmt.Println("The tracker can automatically record your terminal commands for easy browsing and re-execution.")
	fmt.Println("Setup options:")
	fmt.Println("  1. Quick setup (automatic with defaults)")
	fmt.Println("  2. Custom setup (interactive wizard)")
	fmt.Println("  3. Skip setup (configure later)")
	fmt.Print("Choose an option (1/2/3): ")

	response, err := reader.ReadString('\n')
	if err != nil {
		// treat read errors (including EOF) as empty response -> default
		response = ""
	}
	response = strings.TrimSpace(response)

	switch response {
	case "1", "":
		// Quick automatic setup
		fmt.Println()
		return autoSetup()
	case "2":
		// Interactive setup wizard
		fmt.Println()
		return runInteractiveSetup()
	case "3":
		fmt.Println("\nSetup skipped. You can run setup later with:")
		fmt.Println("  tracker setup              # Quick setup")
		fmt.Println("  tracker setup --interactive # Custom setup")
		return nil
	default:
		fmt.Println("\nInvalid option. Setup skipped.")
		fmt.Println("Run 'tracker setup' when you're ready to configure.")
		return nil
	}
}

func runSetup(cmd *cobra.Command, args []string) error {
	if setupInteractive {
		return runInteractiveSetup()
	}

	fmt.Println("Setting up command recording...")

	// Detect current shell
	detector := shell.NewDetector()
	shellType, err := detector.DetectShell()
	if err != nil {
		return fmt.Errorf("failed to detect shell: %w", err)
	}

	fmt.Printf("Detected shell: %s\n", shellType)

	// Check if already configured
	integrator := shell.NewIntegrator()
	isInstalled, err := integrator.IsInstalled(shellType)
	if err != nil {
		return fmt.Errorf("failed to check installation status: %w", err)
	}

	if isInstalled && !setupForce {
		fmt.Println("Command recording is already configured for this shell.")
		fmt.Println("Use --force to reconfigure or run 'tracker status' to check status.")
		return nil
	}

	// Create backup of shell config before modification
	if err := backupShellConfig(shellType); err != nil {
		fmt.Printf("Warning: Failed to create backup of shell config: %v\n", err)
	}

	// Setup recording
	if err := interceptor.SetupRecording(); err != nil {
		return fmt.Errorf("failed to setup recording: %w", err)
	}

	fmt.Println("\n✓ Command recording setup completed successfully!")
	fmt.Println("\nNext steps:")
	printShellRestartInstructions(shellType)
	fmt.Println("  2. Verify setup with: tracker status")
	fmt.Println("  3. Start using your terminal normally - commands will be recorded automatically")

	return nil
}

// backupShellConfig creates a backup of the shell configuration file
func backupShellConfig(shellType history.ShellType) error {
	platform := shell.NewPlatformAbstraction()
	configPath, err := platform.GetShellConfigPath(shellType)
	if err != nil {
		return err
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil // No file to backup
	}

	// Check write permissions before attempting backup
	if err := checkWritePermissions(configPath); err != nil {
		return fmt.Errorf("insufficient permissions to backup config file: %w", err)
	}

	// Create backup with timestamp
	backupPath := configPath + ".backup"

	// Read original file
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	// Write backup
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return err
	}

	fmt.Printf("Created backup: %s\n", backupPath)
	return nil
}

// checkWritePermissions checks if we have write permissions for a file or its directory
func checkWritePermissions(path string) error {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, check directory permissions
			dir := filepath.Dir(path)
			dirInfo, dirErr := os.Stat(dir)
			if dirErr != nil {
				return fmt.Errorf("cannot access directory: %w", dirErr)
			}
			// Check if directory is writable
			if dirInfo.Mode().Perm()&0200 == 0 {
				return fmt.Errorf("directory is not writable")
			}
			return nil
		}
		return err
	}

	// File exists, check if it's writable
	if info.Mode().Perm()&0200 == 0 {
		return fmt.Errorf("file is not writable")
	}

	return nil
}

// printShellRestartInstructions prints shell-specific restart instructions
func printShellRestartInstructions(shellType history.ShellType) {
	fmt.Println("  1. Restart your shell or run:")

	switch shellType {
	case history.PowerShell:
		fmt.Println("     . $PROFILE")
	case history.Bash:
		fmt.Println("     source ~/.bashrc")
	case history.Zsh:
		fmt.Println("     source ~/.zshrc")
	case history.Cmd:
		fmt.Println("     Restart your Command Prompt")
	default:
		fmt.Println("     Restart your shell")
	}
}

func runInteractiveSetup() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Command History Tracker Setup Wizard ===")
	fmt.Println("This wizard will help you configure command recording for your shell.")

	// Detect shell
	detector := shell.NewDetector()
	shellType, err := detector.DetectShell()
	if err != nil {
		return fmt.Errorf("failed to detect shell: %w", err)
	}

	fmt.Printf("\nDetected shell: %s\n", shellType)

	// Check if shell is supported
	if !detector.IsShellSupported(shellType) {
		fmt.Printf("⚠ Warning: Shell %s may not be fully supported on this platform.\n", shellType)
		fmt.Println("\nSupported shells on your platform:")
		platform := shell.NewPlatformAbstraction()
		for _, s := range platform.GetSupportedShells() {
			fmt.Printf("  - %s\n", s)
		}
	}

	fmt.Print("\nIs this correct? (Y/n): ")
	response, err := reader.ReadString('\n')
	if err != nil {
		response = ""
	}
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "n" || response == "no" {
		fmt.Println("\nSupported shells on your platform:")
		platform := shell.NewPlatformAbstraction()
		for _, s := range platform.GetSupportedShells() {
			fmt.Printf("  - %s\n", s)
		}
		fmt.Println("\nPlease set your SHELL environment variable or run the tracker from your preferred shell.")
		return nil
	}

	// Check shell config file accessibility
	platform := shell.NewPlatformAbstraction()
	configPath, err := platform.GetShellConfigPath(shellType)
	if err != nil {
		fmt.Printf("\n⚠ Warning: Could not determine shell config path: %v\n", err)
		fmt.Print("Continue anyway? (y/N): ")
		response, err = reader.ReadString('\n')
		if err != nil {
			response = ""
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			return fmt.Errorf("setup cancelled")
		}
	} else {
		fmt.Printf("Shell config file: %s\n", configPath)

		// Check write permissions
		if err := checkWritePermissions(configPath); err != nil {
			fmt.Printf("\n⚠ Warning: %v\n", err)
			fmt.Println("You may need to run with elevated permissions or fix file permissions.")
			fmt.Print("Continue anyway? (y/N): ")
			response, err = reader.ReadString('\n')
			if err != nil {
				response = ""
			}
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				return fmt.Errorf("setup cancelled due to permission issues")
			}
		}
	}

	// Check if already installed
	integrator := shell.NewIntegrator()
	isInstalled, err := integrator.IsInstalled(shellType)
	if err == nil && isInstalled {
		fmt.Println("\n⚠ Command recording is already configured for this shell.")
		fmt.Print("Do you want to reconfigure? (y/N): ")
		response, err = reader.ReadString('\n')
		if err != nil {
			response = ""
		}
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Setup cancelled. Use 'tracker status' to check current configuration.")
			return nil
		}
	}

	// Load or create config
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Println("\nCreating new configuration with defaults...")
		cfg = config.DefaultConfig()
	}

	fmt.Println("\n--- Configuration Options ---")

	// Configure retention
	fmt.Printf("Retention period: How long to keep command history\n")
	fmt.Printf("Current: %d days\n", cfg.RetentionDays)
	fmt.Print("Change retention period? (y/N): ")
	response, err = reader.ReadString('\n')
	if err != nil {
		response = ""
	}
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "y" || response == "yes" {
		for {
			fmt.Print("Enter retention period in days (1-365, default 90): ")
			var days int
			_, err := fmt.Fscanf(reader, "%d\n", &days)
			if err != nil {
				// clear remainder of line
				if _, err := reader.ReadString('\n'); err != nil {
					_ = err
					// ignore error while clearing buffer
				}
				fmt.Println("Invalid input. Please enter a number.")
				continue
			}
			if days > 0 && days <= 365 {
				cfg.RetentionDays = days
				break
			}
			fmt.Println("Please enter a value between 1 and 365.")
		}
	}

	// Configure max commands
	fmt.Printf("\nMax commands per directory: Limit history size per directory\n")
	fmt.Printf("Current: %d commands\n", cfg.MaxCommands)
	fmt.Print("Change max commands? (y/N): ")
	response, err = reader.ReadString('\n')
	if err != nil {
		response = ""
	}
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "y" || response == "yes" {
		for {
			fmt.Print("Enter max commands per directory (100-100000, default 10000): ")
			var max int
			_, err := fmt.Fscanf(reader, "%d\n", &max)
			if err != nil {
				// clear remainder of line
				if _, err := reader.ReadString('\n'); err != nil {
					// ignore error while clearing buffer
				}
				fmt.Println("Invalid input. Please enter a number.")
				continue
			}
			if max >= 100 && max <= 100000 {
				cfg.MaxCommands = max
				break
			}
			fmt.Println("Please enter a value between 100 and 100000.")
		}
	}

	// Configure auto cleanup
	fmt.Printf("\nAuto cleanup: Automatically remove old commands based on retention policy")
	fmt.Printf("Current: %v\n", cfg.AutoCleanup)
	fmt.Print("Enable auto cleanup? (Y/n): ")
	response, err = reader.ReadString('\n')
	if err != nil {
		response = ""
	}
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "n" || response == "no" {
		cfg.AutoCleanup = false
	} else {
		cfg.AutoCleanup = true
	}

	// Configure exclude patterns
	fmt.Printf("\nExclude patterns: Commands to exclude from recording\n")
	fmt.Printf("Current patterns: %v\n", cfg.ExcludePatterns)
	fmt.Print("Modify exclude patterns? (y/N): ")
	response, err = reader.ReadString('\n')
	if err != nil {
		response = ""
	}
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "y" || response == "yes" {
		fmt.Println("Enter commands to exclude (comma-separated, or press Enter to keep current):")
		fmt.Print("> ")
		patterns, err := reader.ReadString('\n')
		if err != nil {
			patterns = ""
		}
		patterns = strings.TrimSpace(patterns)
		if patterns != "" {
			cfg.ExcludePatterns = strings.Split(patterns, ",")
			for i := range cfg.ExcludePatterns {
				cfg.ExcludePatterns[i] = strings.TrimSpace(cfg.ExcludePatterns[i])
			}
		}
	}

	// Save configuration
	fmt.Println("\nSaving configuration...")
	if err := cfg.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println("✓ Configuration saved")

	// Create backup before modifying shell config
	fmt.Println("\nCreating backup of shell configuration...")
	if err := backupShellConfig(shellType); err != nil {
		fmt.Printf("Warning: Failed to create backup: %v\n", err)
	}

	// Install shell hooks
	fmt.Println("\nInstalling shell hooks...")
	if err := interceptor.SetupRecording(); err != nil {
		return fmt.Errorf("failed to setup recording: %w", err)
	}

	fmt.Println("\n=== Setup Complete ===")
	fmt.Println("\n✓ Command recording is now configured!")

	// Offer to verify installation
	fmt.Print("\nWould you like to verify the installation now? (Y/n): ")
	response, err = reader.ReadString('\n')
	if err != nil {
		response = ""
	}
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "n" && response != "no" {
		fmt.Println("\nRunning verification...")
		if err := verifySetup(shellType); err != nil {
			fmt.Printf("⚠ Verification found issues: %v\n", err)
			fmt.Println("You may need to manually check your shell configuration.")
		} else {
			fmt.Println("✓ Verification passed!")
		}
	}

	fmt.Println("\nNext steps:")
	printShellRestartInstructions(shellType)
	fmt.Println("  2. Verify setup with: tracker verify")
	fmt.Println("  3. Browse history with: tracker browse")
	fmt.Println("  4. Check status with: tracker status")
	fmt.Println("\nTips:")
	fmt.Println("  - Your shell configuration has been backed up with a .backup extension")
	fmt.Println("  - Run 'tracker uninstall' if you want to remove the tracker")
	fmt.Println("  - Run 'tracker setup --help' for more setup options")

	return nil
}

func runRemove(cmd *cobra.Command, args []string) error {
	fmt.Println("Removing command recording hooks...")

	if err := interceptor.RemoveRecording(); err != nil {
		return fmt.Errorf("failed to remove recording: %w", err)
	}

	fmt.Println("Command recording hooks removed successfully!")
	fmt.Println("Please restart your shell or source your shell configuration file.")

	return nil
}

func runCleanup(cmd *cobra.Command, args []string) error {
	fmt.Println("Cleaning up old command history...")

	if err := interceptor.CleanupRecording(); err != nil {
		return fmt.Errorf("failed to cleanup recording: %w", err)
	}

	fmt.Println("Command history cleanup completed successfully!")

	return nil
}

func runUninstall(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("Command History Tracker Uninstall")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("\nThis will:")
	fmt.Println("  ✓ Remove shell hooks from your configuration files")
	fmt.Println("  ✓ Restore backup configuration files if available")
	fmt.Println("  ✓ Optionally delete your command history database")
	fmt.Println("  ✓ Clean up all tracker-related files")
	fmt.Println()

	// Confirm uninstall
	fmt.Print("Are you sure you want to uninstall? (y/N): ")
	response, err := reader.ReadString('\n')
	if err != nil {
		response = ""
	}
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		fmt.Println("Uninstall cancelled.")
		return nil
	}

	fmt.Println("\nStarting uninstall process...")

	// Detect current shell for cleanup
	detector := shell.NewDetector()
	shellType, err := detector.DetectShell()
	if err != nil {
		fmt.Printf("⚠ Warning: Failed to detect shell: %v\n", err)
		fmt.Println("  Continuing with uninstall...")
		shellType = history.Unknown
	} else {
		fmt.Printf("Detected shell: %s\n", shellType)
	}

	// Check current installation status
	if shellType != history.Unknown {
		integrator := shell.NewIntegrator()
		isInstalled, err := integrator.IsInstalled(shellType)
		if err != nil {
			fmt.Printf("⚠ Warning: Could not check installation status: %v\n", err)
		} else if !isInstalled {
			fmt.Println("✓ Shell hooks are not currently installed")
		}
	}

	// Remove shell hooks
	fmt.Println("\n1. Removing shell hooks...")
	if err := interceptor.RemoveRecording(); err != nil {
		fmt.Printf("   ⚠ Warning: Failed to remove shell hooks: %v\n", err)
		fmt.Println("   You may need to manually remove hooks from your shell config")
	} else {
		fmt.Println("   ✓ Shell hooks removed")
	}

	// Restore backup if available
	fmt.Println("\n2. Restoring shell configuration backup...")
	if shellType != history.Unknown {
		if err := restoreShellConfigBackup(shellType); err != nil {
			fmt.Printf("   Note: No backup to restore (this is normal if setup was not completed)\n")
		}
	} else {
		fmt.Println("   Skipped (shell type unknown)")
	}

	// Ask about deleting history
	fmt.Println("\n3. Command history database...")
	fmt.Print("   Delete command history database? (y/N): ")
	response, err = reader.ReadString('\n')
	if err != nil {
		response = ""
	}
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "y" || response == "yes" {
		fmt.Println("   Deleting database and configuration files...")
		if err := cleanupAllData(); err != nil {
			fmt.Printf("   ⚠ Warning: Failed to cleanup all data: %v\n", err)
			fmt.Println("   You may need to manually delete files")
		} else {
			fmt.Println("   ✓ All data cleaned up")
		}
	} else {
		fmt.Println("   ✓ Database preserved")

		// Just remove config file
		configPath := config.GetConfigPath()
		if configPath != "" {
			if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
				fmt.Printf("   ⚠ Warning: Failed to delete config file: %v\n", err)
			} else if err == nil {
				fmt.Println("   ✓ Configuration file removed")
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Uninstall Complete!")
	fmt.Println(strings.Repeat("=", 50))

	fmt.Println("\nThe tracker executable is still installed in your Go bin directory.")
	fmt.Println("To completely remove it, run:")

	// Platform-specific removal instructions
	platform := shell.NewPlatformAbstraction()
	if platform.GetPlatform() == shell.PlatformWindows {
		goPath := os.Getenv("GOPATH")
		if goPath == "" {
			goPath = filepath.Join(os.Getenv("USERPROFILE"), "go")
		}
		fmt.Printf("  del %s\\bin\\tracker.exe\n", goPath)
	} else {
		fmt.Println("  rm $(which tracker)")
	}

	fmt.Println("\nPlease restart your shell for changes to take effect.")
	fmt.Println("\nThank you for using Command History Tracker!")

	return nil
}

// restoreShellConfigBackup restores the backup of shell configuration file
func restoreShellConfigBackup(shellType history.ShellType) error {
	platform := shell.NewPlatformAbstraction()
	configPath, err := platform.GetShellConfigPath(shellType)
	if err != nil {
		return err
	}

	backupPath := configPath + ".backup"

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup found")
	}

	// Read backup
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}

	// Restore backup
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		return err
	}

	// Remove backup file
	os.Remove(backupPath)

	fmt.Println("✓ Shell configuration restored from backup")
	return nil
}

// cleanupAllData removes all tracker-related data
func cleanupAllData() error {
	hasErrors := false
	deletedFiles := []string{}

	cfg, err := config.LoadConfig()
	if err != nil {
		// If config can't be loaded, try to clean up with defaults
		cfg = config.DefaultConfig()
	}

	// Delete storage database
	if cfg.StoragePath != "" {
		absPath, _ := filepath.Abs(cfg.StoragePath)
		if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("     ⚠ Failed to delete database at %s: %v\n", absPath, err)
			hasErrors = true
		} else if err == nil {
			deletedFiles = append(deletedFiles, absPath)
		}
	}

	// Also try to delete database in current directory (default location)
	defaultDbPaths := []string{"./commands.db", "./test.db"}
	for _, dbPath := range defaultDbPaths {
		if absPath, err := filepath.Abs(dbPath); err == nil {
			if err := os.Remove(absPath); err == nil {
				deletedFiles = append(deletedFiles, absPath)
			}
		}
	}

	// Delete config file
	configPath := config.GetConfigPath()
	if configPath != "" {
		if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("     ⚠ Failed to delete config file: %v\n", err)
			hasErrors = true
		} else if err == nil {
			deletedFiles = append(deletedFiles, configPath)
		}
	}

	// Delete config directory if empty
	configDir := filepath.Dir(configPath)
	if configDir != "" && configDir != "." {
		if err := os.Remove(configDir); err == nil {
			deletedFiles = append(deletedFiles, configDir)
		}
		// Ignore error if directory is not empty or doesn't exist
	}

	// Delete any backup files for all supported shells
	platform := shell.NewPlatformAbstraction()

	// Try to clean up backups for all supported shells
	for _, shellType := range platform.GetSupportedShells() {
		if configPath, err := platform.GetShellConfigPath(shellType); err == nil {
			backupPath := configPath + ".backup"
			if err := os.Remove(backupPath); err == nil {
				deletedFiles = append(deletedFiles, backupPath)
			}
		}
	}

	// Clean up any temporary files
	tempPatterns := []string{
		filepath.Join(os.TempDir(), "tracker-*"),
		filepath.Join(os.TempDir(), "command-history-*"),
	}

	for _, pattern := range tempPatterns {
		matches, err := filepath.Glob(pattern)
		if err == nil {
			for _, match := range matches {
				if err := os.RemoveAll(match); err == nil {
					deletedFiles = append(deletedFiles, match)
				}
			}
		}
	}

	// Print summary
	if len(deletedFiles) > 0 {
		fmt.Printf("     Deleted %d file(s):\n", len(deletedFiles))
		for _, file := range deletedFiles {
			fmt.Printf("       - %s\n", file)
		}
	}

	if hasErrors {
		return fmt.Errorf("some files could not be deleted")
	}

	return nil
}
