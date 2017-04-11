package file

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/libbeat/logp"
)

// globPattern detects the use of "**" and expands it to standard glob patterns up to a max depth
func globPatterns(pattern string, doubleStarPatternDepth uint8) ([]string, error) {
	if doubleStarPatternDepth == 0 {
		return []string{pattern}, nil
	}
	var patterns []string
	isAbs := filepath.IsAbs(pattern)
	patternList := strings.Split(pattern, string(os.PathSeparator))
	for i, dir := range patternList {
		if len(patterns) > 0 {
			if dir == "**" {
				err := fmt.Sprintf("glob(%s) failed: cannot specify multiple ** within a pattern", pattern)
				logp.Err(err)
				return nil, errors.New(err)
			}
			for i := range patterns {
				patterns[i] = filepath.Join(patterns[i], dir)
			}
		} else if dir == "**" {
			prefix := filepath.Join(patternList[:i]...)
			if isAbs {
				prefix = string(os.PathSeparator) + prefix
			}
			wildcards := ""
			for j := uint8(0); j <= doubleStarPatternDepth; j++ {
				patterns = append(patterns, filepath.Join(prefix, wildcards))
				wildcards = filepath.Join(wildcards, "*")
			}
		}
	}
	if len(patterns) == 0 {
		patterns = []string{pattern}
	}
	return patterns, nil
}

// Glob expands '**' patterns into multiple patterns to satisfy https://golang.org/pkg/path/filepath/#Match
func Glob(pattern string, doubleStarPatternDepth uint8) ([]string, error) {
	patterns, err := globPatterns(pattern, doubleStarPatternDepth)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, p := range patterns {
		// Evaluate the path as a wildcards/shell glob
		match, err := filepath.Glob(p)
		if err != nil {
			return nil, err
		}
		matches = append(matches, match...)
	}
	return matches, nil
}
