package file

import (
	"fmt"
	"path/filepath"
)

func wildcards(doubleStarPatternDepth uint8, dir string, suffix string) []string {
	wildcardList := []string{}
	w := ""
	i := uint8(0)
	if dir == "" && suffix == "" {
		// Don't expand to "" on relative paths
		w = "*"
		i = 1
	}
	for ; i <= doubleStarPatternDepth; i++ {
		wildcardList = append(wildcardList, w)
		w = filepath.Join(w, "*")
	}
	return wildcardList
}

// GlobPatterns detects the use of "**" and expands it to standard glob patterns up to a max depth
func GlobPatterns(pattern string, doubleStarPatternDepth uint8) ([]string, error) {
	if doubleStarPatternDepth == 0 {
		return []string{pattern}, nil
	}
	var wildcardList []string
	var prefix string
	var suffix string
	dir, file := filepath.Split(filepath.Clean(pattern))
	for file != "" && file != "." {
		if file == "**" {
			if len(wildcardList) > 0 {
				return nil, fmt.Errorf("multiple ** in %q", pattern)
			}
			wildcardList = wildcards(doubleStarPatternDepth, dir, suffix)
			prefix = dir
		} else if len(wildcardList) == 0 {
			suffix = filepath.Join(file, suffix)
		}
		dir, file = filepath.Split(filepath.Clean(dir))
	}
	if len(wildcardList) == 0 {
		return []string{pattern}, nil
	}
	var patterns []string
	for _, w := range wildcardList {
		patterns = append(patterns, filepath.Join(prefix, w, suffix))
	}
	return patterns, nil
}

// Glob expands '**' patterns into multiple patterns to satisfy https://golang.org/pkg/path/filepath/#Match
func Glob(pattern string, doubleStarPatternDepth uint8) ([]string, error) {
	patterns, err := GlobPatterns(pattern, doubleStarPatternDepth)
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
