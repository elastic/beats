// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/osquery/osquery-go/plugin/table"
	"go.uber.org/multierr"
)

func GetTableRows(ctx context.Context, queryContext table.QueryContext, log func(m string, kvs ...any)) ([]map[string]string, error) {
	results := make([]map[string]string, 0)

	userPaths := discoverUsers(log)

	var merr error
	for _, browser := range defaultBrowsers {
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

func processProfile(ctx context.Context, queryContext table.QueryContext, browser, profilePath string, log func(m string, kvs ...any), parserFunc parserFunc) ([]map[string]string, error) {
	log("processing profile", "browser", browser, "path", profilePath)

	entries, err := parserFunc(ctx, queryContext, profilePath, browser, log)
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
