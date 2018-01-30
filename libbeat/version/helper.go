package version

import "time"

// GetDefaultVersion returns the current libbeat version.
// This method is in a separate file as the version.go file is auto generated
func GetDefaultVersion() string {
	return defaultBeatVersion
}

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
