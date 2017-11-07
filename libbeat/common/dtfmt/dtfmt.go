package dtfmt

import (
	"time"
)

// Format applies the format-pattern to the given timestamp.
// Returns the formatted string or an error if pattern is invalid.
func Format(t time.Time, pattern string) (string, error) {
	f, err := NewFormatter(pattern)
	if err != nil {
		return "", err
	}
	return f.Format(t)
}
