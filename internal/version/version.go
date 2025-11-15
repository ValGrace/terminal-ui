package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the current version of the application
	Version = "0.1.0"

	// GitCommit is the git commit hash (set during build)
	GitCommit = "dev"

	// BuildDate is the build date (set during build)
	BuildDate = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = runtime.Version()
)

// Info returns version information
func Info() string {
	return fmt.Sprintf("Command History Tracker v%s", Version)
}

// FullInfo returns detailed version information
func FullInfo() string {
	return fmt.Sprintf(`Command History Tracker
Version:    %s
Git Commit: %s
Build Date: %s
Go Version: %s
OS/Arch:    %s/%s`,
		Version,
		GitCommit,
		BuildDate,
		GoVersion,
		runtime.GOOS,
		runtime.GOARCH,
	)
}

// GetVersion returns the version string
func GetVersion() string {
	return Version
}

// GetGitCommit returns the git commit hash
func GetGitCommit() string {
	return GitCommit
}

// GetBuildDate returns the build date
func GetBuildDate() string {
	return BuildDate
}
