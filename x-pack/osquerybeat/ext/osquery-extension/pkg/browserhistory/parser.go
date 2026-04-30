// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	elasticbrowserhistory "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/elastic_browser_history"
)

var (
	newParserFuncs = map[string]func(context.Context, searchLocation, *logger.Logger) historyParser{}
	once           sync.Once
)

type profile struct {
	Name          string `osquery:"profile_name"`
	User          string `osquery:"user"`
	Browser       string `osquery:"browser"`
	ProfilePath   string
	HistoryPath   string
	CustomDataDir string
}

type historyParser interface {
	parse(ctx context.Context, queryContext table.QueryContext, filters []filters.Filter) ([]elasticbrowserhistory.Result, error)
}

func initParsers() {
	newParserFuncs["chromium"] = newChromiumParser
	newParserFuncs["firefox"] = newFirefoxParser
	newParserFuncs["safari"] = newSafariParser
}

func getParsers(ctx context.Context, location searchLocation, log *logger.Logger) []historyParser {
	var parsers []historyParser
	for _, newParser := range newParserFuncs {
		if parser := newParser(ctx, location, log); parser != nil {
			parsers = append(parsers, parser)
		}
	}
	return parsers
}

// findFilesRecursively searches for files with a specific name recursively
func findFilesRecursively(basePath, fileName string, log *logger.Logger) []string {
	var foundFiles []string

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return foundFiles
	}

	for _, entry := range entries {
		fullPath := filepath.Join(basePath, entry.Name())

		if entry.IsDir() {
			// Recursively search subdirectories
			subFiles := findFilesRecursively(fullPath, fileName, log)
			foundFiles = append(foundFiles, subFiles...)
		} else if entry.Name() == fileName {
			// Found the target file
			foundFiles = append(foundFiles, fullPath)
		}
	}

	return foundFiles
}
