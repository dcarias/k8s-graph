package version

import (
	"fmt"
)

var (
	// GitCommit will be set at build time via ldflags
	GitCommit = "unknown"
	// GitBranch will be set at build time via ldflags
	GitBranch = "unknown"
	// Version is the semantic version
	Version = "1.0.0"
)

// GetVersionInfo returns version information as a map
func GetVersionInfo() map[string]string {
	return map[string]string{
		"version": Version,
		"commit":  GitCommit,
		"branch":  GitBranch,
		"full":    fmt.Sprintf("%s (%s@%s)", Version, GitBranch, GitCommit),
	}
}

// GetFullVersion returns the full version string
func GetFullVersion() string {
	return fmt.Sprintf("%s (%s@%s)", Version, GitBranch, GitCommit)
}
