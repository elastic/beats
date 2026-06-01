// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/client"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	elasticbrowserhistory "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/elastic_browser_history"
)

func init() {
	elasticbrowserhistory.RegisterGenerateFunc(func(ctx context.Context, queryContext table.QueryContext, log *logger.Logger, _ *client.ResilientClient) ([]elasticbrowserhistory.Result, error) {
		return getResults(ctx, queryContext, log)
	})
}

func getResults(ctx context.Context, queryContext table.QueryContext, log *logger.Logger) ([]elasticbrowserhistory.Result, error) {
	once.Do(initParsers)

	var results []elasticbrowserhistory.Result

	allFilters := filters.GetConstraintFilters(queryContext)
	locations, err := getSearchLocations(queryContext, log)
	if err != nil {
		return nil, err
	}
	var merr error
	for _, location := range locations {
		parsers := getParsers(ctx, location, log)
		if len(parsers) == 0 {
			continue
		}
		log.Infof("Parsing browser history for location: %#v", location)
		for _, parser := range parsers {
			res, err := parser.parse(ctx, queryContext, allFilters)
			if err != nil {
				merr = errors.Join(merr, err)
			}
			results = append(results, res...)
		}
	}

	return results, merr
}

type searchLocation struct {
	browser  string
	path     string
	isCustom bool
}

func getSearchLocations(queryContext table.QueryContext, log *logger.Logger) ([]searchLocation, error) {
	searchLocations, err := getSearchLocationsFromFilters(queryContext)
	if err != nil {
		return nil, err
	}
	if len(searchLocations) > 0 {
		return searchLocations, nil
	}

	userPaths := discoverUsers(log)
	var results []searchLocation
	for _, browser := range defaultBrowsers {
		for _, userPath := range userPaths {
			for _, browserBaseDir := range getBrowserPaths(browser) {
				results = append(results, searchLocation{
					browser: browser,
					path:    filepath.Join(userPath, browserBaseDir),
				})
			}
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
			results = append(results, searchLocation{
				path:     path,
				isCustom: true,
			})
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
func extractUserFromPath(filePath string, log *logger.Logger) string {
	// Normalize path separators
	normalizedPath := filepath.ToSlash(filePath)
	parts := strings.Split(normalizedPath, "/")

	// Find user directory - look for patterns like /Users/username, /home/username, C:/Users/username
	for i, part := range parts {
		if (part == "Users" || part == "home") && i+1 < len(parts) {
			user := parts[i+1]
			return user
		} else if part == "root" {
			return "root"
		}
	}

	log.Infof("no user found in path: %s", filePath)
	return ""
}
