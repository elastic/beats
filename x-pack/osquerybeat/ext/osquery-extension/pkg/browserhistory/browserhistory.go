// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
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
	userFilters := getUserFilters(queryContext)
	userPaths := discoverUsers(userFilters, log)

	browsersFilters := getBrowserFilters(queryContext)
	browsers := defaultBrowsers
	if len(browsersFilters) != 0 {
		browsers = make([]string, len(browsersFilters))
		copy(browsers, browsersFilters)
	}

	var merr error
	for _, browser := range browsers {
		for _, userPath := range userPaths {
			browserBaseDir := getBrowserPath(browser)
			if browserBaseDir == "" {
				continue
			}

			fullBasePath := filepath.Join(userPath, browserBaseDir)
			parser := getParser(browser, fullBasePath, log)
			if parser == nil {
				log("no supported parser found for path", "path", fullBasePath)
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
	}

	return results, merr
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
