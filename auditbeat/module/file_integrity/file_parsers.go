// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package file_integrity

import (
	"regexp"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// FileParser is a file analyser providing enrichment for file.* fields.
type FileParser interface {
	Parse(dst mapstr.M, path string) error
}

// FileParsers returns the set of file parsers required to satisfy the config.
func FileParsers(c Config) []FileParser {
	// TODO: Consider whether to allow specification by fileparser name in
	// addition to target field.

	parserNames, fields := parserNamesAndFields(c)
	parsers := make([]FileParser, 0, len(parserNames))
	for name := range parserNames {
		parsers = append(parsers, fileParsers[name](fields))
	}
	return parsers
}

func parserNamesAndFields(c Config) (parserNames, fields map[string]bool) {
	parserNames = make(map[string]bool)
	fields = make(map[string]bool)
	for _, p := range c.FileParsers {
		if pat, ok := unquoteRegexp(p); ok {
			// The Config has been verified by this point, so we know the patterns are valid.
			re := regexp.MustCompile(pat)
			for f := range fileParserFor {
				if re.MatchString(f) {
					fields[f] = true
					parserNames[fileParserFor[f]] = true
				}
			}
			continue
		}

		fields[p] = true
		parserNames[fileParserFor[p]] = true
	}
	return parserNames, fields
}

// wantFields is a helper that a FileParser can use to filter fields. It returns
// true if any of the provided queries is present in the wanted set or if
// the wanted set is nil.
func wantFields(want map[string]bool, queries ...string) bool {
	if want == nil {
		return true
	}
	for _, f := range queries {
		if want[f] {
			return true
		}
	}
	return false
}

// unquoteRegexp returns whether s is a regexp quoted by // and returns the
// quoted pattern.
func unquoteRegexp(s string) (pat string, ok bool) {
	if len(s) >= 2 && s[0] == '/' && s[len(s)-1] == '/' {
		return s[1 : len(s)-1], true
	}
	return "", false
}

// fileParsers contains the set of file parsers that can be executed. Fields used
// by the parsers must be present in the flatbuffers schema. This level of indirection
// exists to deduplicate parsers from field requests.
var fileParsers = map[string]func(fields map[string]bool) FileParser{
	"executable_object": func(fields map[string]bool) FileParser { return exeObjParser(fields) },
}
