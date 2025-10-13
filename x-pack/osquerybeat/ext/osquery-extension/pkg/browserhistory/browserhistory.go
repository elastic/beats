// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/osquery/osquery-go/plugin/table"
	"go.uber.org/multierr"
)

func GetTableRows(ctx context.Context, queryContext table.QueryContext, log func(m string, kvs ...any)) ([]map[string]string, error) {
	once.Do(initParsers)

	results := make([]map[string]string, 0)

	profileFilters := getProfileNameFilters(queryContext)
	locations, err := getSearchLocations(queryContext, log)
	if err != nil {
		return nil, err
	}
	var merr error
	for _, location := range locations {
		parser := getParser(location, log)
		if parser == nil {
			log("no supported parser found for path", "path", location.path)
			continue
		}
		visits, err := parser.parse(ctx, queryContext, profileFilters)
		if err != nil {
			merr = multierr.Append(merr, err)
		}
		if len(visits) == 0 {
			continue
		}
		rows := make([]map[string]string, len(visits))
		for i, visit := range visits {
			rows[i] = visit.toMap()
		}
		results = append(results, rows...)
	}

	return results, merr
}

type searchLocation struct {
	browser  string
	path     string
	isCustom bool
}

func getSearchLocations(queryContext table.QueryContext, log func(m string, kvs ...any)) ([]searchLocation, error) {
	searchLocations, err := getSearchLocationsFromFilters(queryContext)
	if err != nil {
		return nil, err
	}
	if len(searchLocations) > 0 {
		return searchLocations, nil
	}

	userFilters := getUserFilters(queryContext)
	userPaths := discoverUsers(userFilters, log)

	browsersFilters := getBrowserFilters(queryContext)
	browsers := defaultBrowsers
	if len(browsersFilters) != 0 {
		browsers = make([]string, len(browsersFilters))
		copy(browsers, browsersFilters)
	}

	var results []searchLocation
	for _, browser := range browsers {
		for _, userPath := range userPaths {
			browserBaseDir := getBrowserPath(browser)
			if browserBaseDir == "" {
				continue
			}
			results = append(results, searchLocation{
				browser: browser,
				path:    filepath.Join(userPath, browserBaseDir),
			})
		}
	}
	return results, nil
}

func getSearchLocationsFromFilters(queryContext table.QueryContext) ([]searchLocation, error) {
	searchFilters, err := getCustomDataDirFilters(queryContext)
	if err != nil {
		return nil, err
	}
	if len(searchFilters) == 0 {
		return nil, nil
	}
	var results []searchLocation
	for _, pattern := range searchFilters {
		// Expand the pattern to get actual paths
		expandedPaths, err := expandPattern(pattern)
		if err != nil {
			return nil, err
		}

		for _, path := range expandedPaths {
			// Look for the first path backwards that does not contain Profile, Default, or User Data
			// and assume it is the browser name, then lowercase and snake case it
			normalizedPath := filepath.ToSlash(path)
			parts := strings.Split(normalizedPath, "/")

			var browserName string
			for i := len(parts) - 1; i >= 0; i-- {
				part := parts[i]
				if !strings.Contains(strings.ToLower(part), "profile") &&
					!strings.Contains(strings.ToLower(part), "default") &&
					!strings.Contains(strings.ToLower(part), "user data") {
					browserName = strings.ToLower(strings.ReplaceAll(part, " ", "_"))
					break
				}
			}

			if browserName != "" {
				results = append(results, searchLocation{
					browser:  browserName,
					path:     path,
					isCustom: true,
				})
			}
		}
	}
	return results, nil
}

func expandPattern(pattern string) ([]string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return []string{pattern}, nil
	}
	var dirs []string
	for _, match := range matches {
		if info, err := os.Stat(match); err == nil && info.IsDir() {
			dirs = append(dirs, match)
		}
	}
	return dirs, nil
}

// extractUserFromPath extracts user information from a file path
func extractUserFromPath(filePath string, log func(m string, kvs ...any)) string {
	// Normalize path separators
	normalizedPath := filepath.ToSlash(filePath)
	parts := strings.Split(normalizedPath, "/")

	// Find user directory - look for patterns like /Users/username, /home/username, C:/Users/username
	for i, part := range parts {
		if (part == "Users" || part == "home") && i+1 < len(parts) {
			user := parts[i+1]
			log("extracted user", "user", user, "path", filePath)
			return user
		}
	}

	log("no user found in path", "path", filePath)
	return ""
}
