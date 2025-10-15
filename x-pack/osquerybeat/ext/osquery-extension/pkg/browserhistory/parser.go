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
)

var (
	newParserFuncs = map[string]func(location searchLocation, log func(m string, kvs ...any)) historyParser{}
	once           sync.Once
)

type profile struct {
	name          string
	user          string
	browser       string
	profilePath   string
	historyPath   string
	customDataDir string
}

type historyParser interface {
	parse(ctx context.Context, queryContext table.QueryContext, filters []filter) ([]*visit, error)
}

func initParsers() {
	newParserFuncs["chromium"] = newChromiumParser
	newParserFuncs["firefox"] = newFirefoxParser
	newParserFuncs["safari"] = newSafariParser
}

func getParsers(location searchLocation, log func(m string, kvs ...any)) []historyParser {
	var parsers []historyParser
	for parserName, newParser := range newParserFuncs {
		if parser := newParser(location, log); parser != nil {
			log("created new parser", "parser", parserName)
			parsers = append(parsers, parser)
		}
	}
	return parsers
}

// findFilesRecursively searches for files with a specific name recursively
func findFilesRecursively(basePath, fileName string, log func(m string, kvs ...any)) []string {
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
