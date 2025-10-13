// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
	"sync"

	"github.com/osquery/osquery-go/plugin/table"
)

var (
	newParserFuncs = map[string]func(location searchLocation, log func(m string, kvs ...any)) historyParser{}
	once           sync.Once
)

type profile struct {
	name        string
	user        string
	profilePath string
	historyPath string
	searchPath  string
}

type historyParser interface {
	parse(ctx context.Context, queryContext table.QueryContext, profileFilters []string) ([]*visit, error)
}

func initParsers() {
	newParserFuncs["chromium"] = newChromiumParser
	newParserFuncs["firefox"] = newFirefoxParser
	newParserFuncs["safari"] = newSafariParser
}

func getParser(location searchLocation, log func(m string, kvs ...any)) historyParser {
	for parserName, newParser := range newParserFuncs {
		if parser := newParser(location, log); parser != nil {
			log("created new parser", "parser", parserName)
			return parser
		}
	}
	return nil
}
