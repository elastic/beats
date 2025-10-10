// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/osquery/osquery-go/plugin/table"
	"go.uber.org/multierr"
)

func GetTableRows(ctx context.Context, queryContext table.QueryContext, log func(m string, kvs ...any)) ([]map[string]string, error) {
	results := make([]map[string]string, 0)

	profileFilters := getProfileFilters(queryContext)
	userFilters := getUserFilters(queryContext)
	userPaths := discoverUsers(userFilters, log)

	browsers := getBrowserFilters(queryContext)

	var merr error
	for _, browser := range browsers {
		for _, userPath := range userPaths {
			browserBaseDir := getBrowserPath(browser)
			if browserBaseDir == "" {
				continue
			}

			fullBasePath := filepath.Join(userPath, browserBaseDir)
			entries, err := os.ReadDir(fullBasePath)
			if err != nil {
				log("cannot read browser directory", "path", fullBasePath, "error", err)
				continue
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				// Filter profiles based on constraints
				if len(profileFilters) > 0 && !matchesFilter(entry.Name(), profileFilters) {
					continue
				}

				profileDir := filepath.Join(fullBasePath, entry.Name())

				parserFunc := getParserFunc(profileDir, log)
				if parserFunc == nil {
					continue
				}

				bresults, err := processProfile(ctx, queryContext, browser, profileDir, log, parserFunc)
				if err != nil {
					merr = multierr.Append(merr, err)
					continue
				}
				results = append(results, bresults...)
			}
		}
	}

	return results, merr
}

func getConstraintFilters(queryContext table.QueryContext, fieldName string, validateAgainst []string) []string {
	clist, ok := queryContext.Constraints[fieldName]
	if !ok || len(clist.Constraints) == 0 {
		if validateAgainst != nil {
			return validateAgainst // Return default for browsers
		}
		return nil // Return nil for user/profile
	}

	results := make([]string, 0, len(clist.Constraints))
	for _, c := range clist.Constraints {
		switch c.Operator {
		case table.OperatorEquals:
			results = append(results, c.Expression)
		case table.OperatorLike:
			// Convert SQL LIKE pattern to filepath.Match pattern
			pattern := strings.ReplaceAll(c.Expression, "%", "*")
			if validateAgainst != nil {
				// For browsers: validate pattern against known browsers
				for _, item := range validateAgainst {
					if matched, _ := filepath.Match(pattern, item); matched {
						results = append(results, item)
					}
				}
			} else {
				// For user/profile: store pattern for later validation
				results = append(results, pattern)
			}
		case table.OperatorGlob:
			if validateAgainst != nil {
				// For browsers: validate pattern against known browsers
				for _, item := range validateAgainst {
					if matched, _ := filepath.Match(c.Expression, item); matched {
						results = append(results, item)
					}
				}
			} else {
				// For user/profile: store pattern for later validation
				results = append(results, c.Expression)
			}
		case table.OperatorRegexp:
			// Compile and validate regexp pattern
			re, err := regexp.Compile(c.Expression)
			if err != nil {
				// Skip invalid regexp patterns
				continue
			}
			if validateAgainst != nil {
				// For browsers: validate pattern against known browsers
				for _, item := range validateAgainst {
					if re.MatchString(item) {
						results = append(results, item)
					}
				}
			} else {
				// For user/profile: store pattern for later validation
				// We store the original expression since we'll need to recompile it later
				results = append(results, c.Expression)
			}
		}
	}

	if len(results) > 0 {
		return results
	}
	if validateAgainst != nil {
		return validateAgainst // Return default for browsers when no matches
	}
	return nil
}

func getProfileFilters(queryContext table.QueryContext) []string {
	return getConstraintFilters(queryContext, "profile", nil)
}

func getUserFilters(queryContext table.QueryContext) []string {
	return getConstraintFilters(queryContext, "user", nil)
}

func getBrowserFilters(queryContext table.QueryContext) []string {
	return getConstraintFilters(queryContext, "browser", defaultBrowsers)
}

type timestampConstraint struct {
	Operator table.Operator
	Value    int64 // Unix timestamp in seconds
}

func getTimestampConstraints(queryContext table.QueryContext) []timestampConstraint {
	clist, ok := queryContext.Constraints["timestamp"]
	if !ok || len(clist.Constraints) == 0 {
		return nil
	}

	var constraints []timestampConstraint
	for _, c := range clist.Constraints {
		// Parse and validate timestamp value
		osqueryTimestamp, err := strconv.ParseInt(c.Expression, 10, 64)
		if err != nil {
			continue // Skip invalid timestamp values
		}

		constraints = append(constraints, timestampConstraint{
			Operator: c.Operator,
			Value:    osqueryTimestamp,
		})
	}

	return constraints
}

// matchesFilter checks if a name matches any of the provided filters.
// Supports exact string matches, glob patterns (*, ?), and regexp patterns.
// Used for filtering browsers, profiles, and users.
func matchesFilter(name string, filters []string) bool {
	for _, filter := range filters {
		// Check for exact match
		if name == filter {
			return true
		}
		// Check for glob pattern match
		if matched, _ := filepath.Match(filter, name); matched {
			return true
		}
		// Check for regexp pattern match
		if re, err := regexp.Compile(filter); err == nil {
			if re.MatchString(name) {
				return true
			}
		}
	}
	return false
}

func processProfile(ctx context.Context, queryContext table.QueryContext, browser, profilePath string, log func(m string, kvs ...any), parserFunc parserFunc) ([]map[string]string, error) {
	log("processing profile", "browser", browser, "path", profilePath)

	entries, err := parserFunc(ctx, queryContext, browser, profilePath, log)
	if err != nil {
		return nil, fmt.Errorf("failed to read history: %w", err)
	}

	rows := make([]map[string]string, len(entries))
	for i, entry := range entries {
		rows[i] = entry.toMap()
	}
	return rows, nil
}

func getParserFunc(basePath string, log func(m string, kvs ...any)) parserFunc {
	firefoxPath := filepath.Join(basePath, "places.sqlite")
	if _, err := os.Stat(firefoxPath); err == nil {
		log("detected firefox parser", "path", firefoxPath)
		return firefoxParser
	}

	chromiumPath := filepath.Join(basePath, "History")
	if _, err := os.Stat(chromiumPath); err == nil {
		log("detected chromium parser", "path", chromiumPath)
		return chromiumParser
	}

	safariPath := filepath.Join(basePath, "History.db")
	if _, err := os.Stat(safariPath); err == nil {
		log("detected safari parser", "path", safariPath)
		return safariParser
	}
	return nil
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
