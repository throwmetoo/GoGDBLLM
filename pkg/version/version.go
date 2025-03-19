package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the current version of the application
	Version = "0.1.0"

	// GitCommit is the git commit hash of the build
	GitCommit = "unknown"

	// BuildDate is the date when the binary was built
	BuildDate = "unknown"
)

// Info returns version information as a string
func Info() string {
	return fmt.Sprintf("Version: %s\nGit Commit: %s\nBuild Date: %s\nGo Version: %s\nOS/Arch: %s/%s",
		Version, GitCommit, BuildDate, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

// ShortInfo returns a short version string
func ShortInfo() string {
	return fmt.Sprintf("%s (%s)", Version, GitCommit[:7])
}

// GetVersion returns the current version
func GetVersion() string {
	return Version
}

// GetCommit returns the git commit hash
func GetCommit() string {
	return GitCommit
}

// GetBuildDate returns the build date
func GetBuildDate() string {
	return BuildDate
}
