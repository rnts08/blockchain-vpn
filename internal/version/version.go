package version

import "fmt"

var (
	// Application semantic version.
	Version = "0.6.3"
	// Optional build metadata (can be injected via -ldflags).
	GitCommit = "dev"
	BuildDate = "unknown"
)

func String() string {
	return fmt.Sprintf("bcvpn %s (commit=%s, built=%s)", Version, GitCommit, BuildDate)
}
