package version

import "fmt"

// Version is the current version of Aura, managed by semantic-release
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
	GoVersion = "unknown"
	Platform  = "unknown"
)

// FullVersion returns a detailed version string
func FullVersion() string {
	return fmt.Sprintf("%s (commit: %s, built: %s, go: %s, platform: %s)",
		Version, GitCommit, BuildDate, GoVersion, Platform)
}
