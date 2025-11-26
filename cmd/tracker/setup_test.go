package main

import (
	"os"
	"path/filepath"
	"testing"
	"github.com/ValGrace/command-history-tracker/internal/config"
	"github.com/ValGrace/command-history-tracker/pkg/history"
	"github.com/ValGrace/command-history-tracker/pkg/shell"
)

func TestSetupWorkflow(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Set up test environment
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir)
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()

	t.Run("DefaultConfigCreation", func(t *testing.T) {
		cfg := config.DefaultConfig()

		if cfg.RetentionDays != 90 {
			t.Errorf("Expected retention days 90, got %d", cfg.RetentionDays)
		}

		if cfg.MaxCommands != 10000 {
			t.Errorf("Expected max commands 10000, got %d", cfg.MaxCommands)
		}

		if !cfg.AutoCleanup {
			t.Error("Expected auto cleanup to be enabled")
		}
	})

	t.Run("ConfigSaveAndLoad", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.RetentionDays = 180
		cfg.MaxCommands = 50000

		if err := cfg.SaveConfig(); err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		loadedCfg, err := config.LoadConfig()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if loadedCfg.RetentionDays != 180 {
			t.Errorf("Expected retention days 180, got %d", loadedCfg.RetentionDays)
		}

		if loadedCfg.MaxCommands != 50000 {
			t.Errorf("Expected max commands 50000, got %d", loadedCfg.MaxCommands)
		}
	})

	t.Run("ConfigValidation", func(t *testing.T) {
		cfg := config.DefaultConfig()

		if err := cfg.Validate(); err != nil {
			t.Errorf("Valid config failed validation: %v", err)
		}

		// Test invalid config
		invalidCfg := config.DefaultConfig()
		invalidCfg.StoragePath = ""

		if err := invalidCfg.Validate(); err == nil {
			t.Error("Expected validation error for empty storage path")
		}
	})
}

func TestShellDetection(t *testing.T) {
	t.Run("DetectCurrentShell", func(t *testing.T) {
		detector := shell.NewDetector()

		shellType, err := detector.DetectShell()
		if err != nil {
			t.Fatalf("Failed to detect shell: %v", err)
		}

		if shellType == history.Unknown {
			t.Error("Expected to detect a known shell type")
		}

		t.Logf("Detected shell: %s", shellType)
	})
}

func TestConfigPath(t *testing.T) {
	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir)
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()

	configPath := config.GetConfigPath()

	expectedPath := filepath.Join(tmpDir, ".command-history-tracker", "config.json")
	if configPath != expectedPath {
		t.Errorf("Expected config path %s, got %s", expectedPath, configPath)
	}
}

func TestUninstallCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir)
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()

	// Create config
	cfg := config.DefaultConfig()
	cfg.StoragePath = filepath.Join(tmpDir, ".command-history-tracker", "storage")

	if err := cfg.SaveConfig(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Create storage directory
	if err := os.MkdirAll(cfg.StoragePath, 0755); err != nil {
		t.Fatalf("Failed to create storage directory: %v", err)
	}

	// Verify files exist
	configPath := config.GetConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should exist")
	}

	if _, err := os.Stat(cfg.StoragePath); os.IsNotExist(err) {
		t.Error("Storage directory should exist")
	}

	// Simulate uninstall cleanup
	os.RemoveAll(cfg.StoragePath)
	os.Remove(configPath)

	// Verify cleanup
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Error("Config file should be deleted")
	}

	if _, err := os.Stat(cfg.StoragePath); !os.IsNotExist(err) {
		t.Error("Storage directory should be deleted")
	}
}
