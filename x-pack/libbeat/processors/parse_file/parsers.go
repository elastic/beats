// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package parse_file

import (
	parserCommon "github.com/elastic/beats/v7/x-pack/libbeat/processors/parse_file/common"
	"github.com/elastic/beats/v7/x-pack/libbeat/processors/parse_file/pe"
)

type parser struct {
	Name    string
	Factory parserCommon.ParserFactory
}

// this is an array in order to preserve priority for magic number collision
var allParsers = []parser{
	makeParser("pe", pe.NewParser),
}

func makeParser(name string, factory parserCommon.ParserFactory) parser {
	return parser{Name: name, Factory: factory}
}

func filterParsers(exclude []string) []parser {
	parsers := []parser{}
	exclusionSet := map[string]struct{}{}
	for _, exclusion := range exclude {
		exclusionSet[exclusion] = struct{}{}
	}

	for _, parser := range allParsers {
		if _, ok := exclusionSet[parser.Name]; ok {
			continue
		}
		parsers = append(parsers, parser)
	}
	return parsers
}

func onlyParsers(only []string) []parser {
	parsers := []parser{}
	inclusionSet := map[string]struct{}{}
	for _, inclusion := range only {
		inclusionSet[inclusion] = struct{}{}
	}

	for _, parser := range allParsers {
		if _, ok := inclusionSet[parser.Name]; ok {
			parsers = append(parsers, parser)
		}
	}
	return parsers
}
