package executor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ValGrace/command-history-tracker/internal/errors"
)

// SecurityPolicy defines security rules for command execution
type SecurityPolicy struct {
	BlacklistedCommands []string
	BlacklistedPatterns []*regexp.Regexp
	AllowedDirectories  []string
	DenyDirectories     []string
	MaxCommandLength    int
	RequireConfirm      bool
}

// DefaultSecurityPolicy returns a security policy with sensible defaults
func DefaultSecurityPolicy() *SecurityPolicy {
	return &SecurityPolicy{
		BlacklistedCommands: []string{
			"rm -rf /",
			"del /s /q C:\\",
			"format C:",
			"mkfs",
			"dd if=/dev/zero",
		},
		BlacklistedPatterns: compileBlacklistPatterns(),
		MaxCommandLength:    10000,
		RequireConfirm:      true,
	}
}

// compileBlacklistPatterns returns compiled regex patterns for blacklisted commands
func compileBlacklistPatterns() []*regexp.Regexp {
	patterns := []string{
		`^rm\s+-rf\s+/\s*$`,
		`^rm\s+-rf\s+/\*`,
		`^del\s+/[sS]\s+/[qQ]\s+[cC]:\\`,
		`^format\s+[cC]:`,
		`^dd\s+if=.*of=/dev/sd`,
		`^mkfs\.`,
		`^:(){ :|:& };:`,
		`>\s*/dev/sd[a-z]`,
		`curl.*\|\s*bash`,
		`wget.*\|\s*sh`,
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if re, err := regexp.Compile(pattern); err == nil {
			compiled = append(compiled, re)
		}
	}
	return compiled
}

// ValidateWithPolicy validates a command against a security policy
func (p *SecurityPolicy) ValidateWithPolicy(command string, directory string) error {
	// Check command length
	if p.MaxCommandLength > 0 && len(command) > p.MaxCommandLength {
		return errors.NewValidationError(
			fmt.Sprintf("command exceeds maximum length of %d characters", p.MaxCommandLength),
			nil,
		)
	}

	// Check blacklisted commands
	for _, blacklisted := range p.BlacklistedCommands {
		if strings.TrimSpace(command) == blacklisted {
			return errors.NewExecutionError(
				fmt.Sprintf("command is blacklisted: %s", command),
				nil,
			).WithContext("command", command).WithContext("reason", "blacklisted")
		}
	}

	// Check blacklisted patterns
	for _, pattern := range p.BlacklistedPatterns {
		if pattern.MatchString(command) {
			return errors.NewExecutionError(
				fmt.Sprintf("command matches blacklisted pattern: %s", command),
				nil,
			).WithContext("command", command).WithContext("reason", "blacklisted_pattern")
		}
	}

	// Check directory restrictions
	if len(p.DenyDirectories) > 0 {
		for _, denied := range p.DenyDirectories {
			if strings.HasPrefix(directory, denied) {
				return errors.NewExecutionError(
					fmt.Sprintf("execution not allowed in directory: %s", directory),
					nil,
				).WithContext("directory", directory).WithContext("reason", "denied_directory")
			}
		}
	}

	if len(p.AllowedDirectories) > 0 {
		allowed := false
		for _, allowedDir := range p.AllowedDirectories {
			if strings.HasPrefix(directory, allowedDir) {
				allowed = true
				break
			}
		}
		if !allowed {
			return errors.NewExecutionError(
				fmt.Sprintf("execution only allowed in specific directories, not: %s", directory),
				nil,
			).WithContext("directory", directory).WithContext("reason", "not_allowed_directory")
		}
	}

	return nil
}

// IsCommandSafe performs basic safety checks on a command
func IsCommandSafe(command string) bool {
	policy := DefaultSecurityPolicy()
	return policy.ValidateWithPolicy(command, "") == nil
}

// CommandValidator provides command validation functionality
type CommandValidator struct {
	policy *SecurityPolicy
}

// NewCommandValidator creates a new command validator with default policy
func NewCommandValidator() *CommandValidator {
	return &CommandValidator{
		policy: DefaultSecurityPolicy(),
	}
}

// NewCommandValidatorWithPolicy creates a validator with a custom policy
func NewCommandValidatorWithPolicy(policy *SecurityPolicy) *CommandValidator {
	return &CommandValidator{
		policy: policy,
	}
}

// Validate validates a command against the security policy
func (v *CommandValidator) Validate(command string, directory string) error {
	return v.policy.ValidateWithPolicy(command, directory)
}

// SetPolicy updates the security policy
func (v *CommandValidator) SetPolicy(policy *SecurityPolicy) {
	v.policy = policy
}

// GetPolicy returns the current security policy
func (v *CommandValidator) GetPolicy() *SecurityPolicy {
	return v.policy
}

// AddBlacklistedCommand adds a command to the blacklist
func (v *CommandValidator) AddBlacklistedCommand(command string) {
	v.policy.BlacklistedCommands = append(v.policy.BlacklistedCommands, command)
}

// RemoveBlacklistedCommand removes a command from the blacklist
func (v *CommandValidator) RemoveBlacklistedCommand(command string) {
	filtered := make([]string, 0)
	for _, cmd := range v.policy.BlacklistedCommands {
		if cmd != command {
			filtered = append(filtered, cmd)
		}
	}
	v.policy.BlacklistedCommands = filtered
}

// IsBlacklisted checks if a command is blacklisted
func (v *CommandValidator) IsBlacklisted(command string) bool {
	for _, blacklisted := range v.policy.BlacklistedCommands {
		if strings.TrimSpace(command) == blacklisted {
			return true
		}
	}

	for _, pattern := range v.policy.BlacklistedPatterns {
		if pattern.MatchString(command) {
			return true
		}
	}

	return false
}
