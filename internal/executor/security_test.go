package executor

import (
	"strings"
	"testing"
)

func TestDefaultSecurityPolicy(t *testing.T) {
	policy := DefaultSecurityPolicy()

	if policy == nil {
		t.Fatal("DefaultSecurityPolicy returned nil")
	}

	if len(policy.BlacklistedCommands) == 0 {
		t.Error("expected blacklisted commands to be populated")
	}

	if len(policy.BlacklistedPatterns) == 0 {
		t.Error("expected blacklisted patterns to be populated")
	}

	if policy.MaxCommandLength <= 0 {
		t.Error("expected positive max command length")
	}
}

func TestSecurityPolicy_ValidateWithPolicy_SafeCommand(t *testing.T) {
	policy := DefaultSecurityPolicy()

	err := policy.ValidateWithPolicy("echo hello", "/tmp")
	if err != nil {
		t.Errorf("expected no error for safe command, got: %v", err)
	}
}

func TestSecurityPolicy_ValidateWithPolicy_BlacklistedCommand(t *testing.T) {
	policy := DefaultSecurityPolicy()

	for _, cmd := range policy.BlacklistedCommands {
		err := policy.ValidateWithPolicy(cmd, "/tmp")
		if err == nil {
			t.Errorf("expected error for blacklisted command '%s', got nil", cmd)
		}
	}
}

func TestSecurityPolicy_ValidateWithPolicy_TooLong(t *testing.T) {
	policy := DefaultSecurityPolicy()
	policy.MaxCommandLength = 10

	longCommand := strings.Repeat("a", 100)
	err := policy.ValidateWithPolicy(longCommand, "/tmp")

	if err == nil {
		t.Error("expected error for command exceeding max length, got nil")
	}
}

func TestSecurityPolicy_ValidateWithPolicy_DeniedDirectory(t *testing.T) {
	policy := DefaultSecurityPolicy()
	policy.DenyDirectories = []string{"/etc", "/sys"}

	err := policy.ValidateWithPolicy("echo test", "/etc/config")
	if err == nil {
		t.Error("expected error for denied directory, got nil")
	}

	err = policy.ValidateWithPolicy("echo test", "/tmp")
	if err != nil {
		t.Errorf("expected no error for allowed directory, got: %v", err)
	}
}

func TestSecurityPolicy_ValidateWithPolicy_AllowedDirectories(t *testing.T) {
	policy := DefaultSecurityPolicy()
	policy.AllowedDirectories = []string{"/home", "/tmp"}

	err := policy.ValidateWithPolicy("echo test", "/home/user")
	if err != nil {
		t.Errorf("expected no error for allowed directory, got: %v", err)
	}

	err = policy.ValidateWithPolicy("echo test", "/etc")
	if err == nil {
		t.Error("expected error for non-allowed directory, got nil")
	}
}

func TestIsCommandSafe(t *testing.T) {
	tests := []struct {
		command  string
		expected bool
	}{
		{"echo hello", true},
		{"ls -la", true},
		{"rm -rf /", false},
		{"format C:", false},
	}

	for _, tt := range tests {
		result := IsCommandSafe(tt.command)
		if result != tt.expected {
			t.Errorf("IsCommandSafe(%q) = %v, want %v", tt.command, result, tt.expected)
		}
	}
}

func TestNewCommandValidator(t *testing.T) {
	validator := NewCommandValidator()

	if validator == nil {
		t.Fatal("NewCommandValidator returned nil")
	}

	if validator.policy == nil {
		t.Error("validator policy not initialized")
	}
}

func TestCommandValidator_Validate(t *testing.T) {
	validator := NewCommandValidator()

	err := validator.Validate("echo test", "/tmp")
	if err != nil {
		t.Errorf("expected no error for safe command, got: %v", err)
	}

	err = validator.Validate("rm -rf /", "/tmp")
	if err == nil {
		t.Error("expected error for dangerous command, got nil")
	}
}

func TestCommandValidator_SetPolicy(t *testing.T) {
	validator := NewCommandValidator()

	customPolicy := &SecurityPolicy{
		MaxCommandLength: 50,
	}

	validator.SetPolicy(customPolicy)

	if validator.GetPolicy() != customPolicy {
		t.Error("policy not set correctly")
	}
}

func TestCommandValidator_AddBlacklistedCommand(t *testing.T) {
	validator := NewCommandValidator()

	initialCount := len(validator.policy.BlacklistedCommands)

	validator.AddBlacklistedCommand("dangerous-command")

	if len(validator.policy.BlacklistedCommands) != initialCount+1 {
		t.Error("blacklisted command not added")
	}

	if !validator.IsBlacklisted("dangerous-command") {
		t.Error("added command should be blacklisted")
	}
}

func TestCommandValidator_RemoveBlacklistedCommand(t *testing.T) {
	validator := NewCommandValidator()

	validator.AddBlacklistedCommand("test-command")

	if !validator.IsBlacklisted("test-command") {
		t.Error("command should be blacklisted before removal")
	}

	validator.RemoveBlacklistedCommand("test-command")

	if validator.IsBlacklisted("test-command") {
		t.Error("command should not be blacklisted after removal")
	}
}

func TestCommandValidator_IsBlacklisted(t *testing.T) {
	validator := NewCommandValidator()

	// Test with default blacklist
	if !validator.IsBlacklisted("rm -rf /") {
		t.Error("'rm -rf /' should be blacklisted")
	}

	if validator.IsBlacklisted("echo hello") {
		t.Error("'echo hello' should not be blacklisted")
	}

	// Test with custom blacklist
	validator.AddBlacklistedCommand("custom-dangerous")

	if !validator.IsBlacklisted("custom-dangerous") {
		t.Error("custom command should be blacklisted")
	}
}

func TestCompileBlacklistPatterns(t *testing.T) {
	patterns := compileBlacklistPatterns()

	if len(patterns) == 0 {
		t.Error("expected blacklist patterns to be compiled")
	}

	// Test that patterns are valid regex
	for _, pattern := range patterns {
		if pattern == nil {
			t.Error("pattern should not be nil")
		}
	}
}

func TestSecurityPolicy_ValidateWithPolicy_BlacklistedPattern(t *testing.T) {
	policy := DefaultSecurityPolicy()

	dangerousCommands := []string{
		"rm -rf /",
		"rm -rf /*",
		"dd if=/dev/zero of=/dev/sda",
		"mkfs.ext4 /dev/sda",
	}

	for _, cmd := range dangerousCommands {
		err := policy.ValidateWithPolicy(cmd, "/tmp")
		if err == nil {
			t.Errorf("expected error for dangerous command '%s', got nil", cmd)
		}
	}
}
