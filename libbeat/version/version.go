package version

import "time"

const defaultBeatVersion = "7.0.0-alpha1"

var (
	buildTime = "unknown"
	commit    = "unknown"
)

// BuildTime exposes the compile-time build time information.
// It will represent the zero time instant if parsing fails.
func BuildTime() time.Time {
	t, err := time.Parse(time.RFC3339, buildTime)
	if err != nil {
		return time.Time{}
	}
	return t
}

// Commit exposes the compile-time commit hash.
func Commit() string {
	return commit
}
