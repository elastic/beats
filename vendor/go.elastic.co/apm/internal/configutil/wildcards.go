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

package configutil

import (
	"strings"

	"go.elastic.co/apm/internal/wildcard"
)

// ParseWildcardPatterns parses s as a comma-separated list of wildcard patterns,
// and returns wildcard.Matchers for each.
//
// Patterns support the "*" wildcard, which will match zero or more characters.
// A prefix of (?-i) treats the pattern case-sensitively, while a prefix of (?i)
// treats the pattern case-insensitively (the default). All other characters in
// the pattern are matched exactly.
func ParseWildcardPatterns(s string) wildcard.Matchers {
	patterns := ParseList(s, ",")
	matchers := make(wildcard.Matchers, len(patterns))
	for i, p := range patterns {
		matchers[i] = ParseWildcardPattern(p)
	}
	return matchers
}

// ParseWildcardPattern parses p as a wildcard pattern, returning a wildcard.Matcher.
//
// Patterns support the "*" wildcard, which will match zero or more characters.
// A prefix of (?-i) treats the pattern case-sensitively, while a prefix of (?i)
// treats the pattern case-insensitively (the default). All other characters in
// the pattern are matched exactly.
func ParseWildcardPattern(p string) *wildcard.Matcher {
	const (
		caseSensitivePrefix   = "(?-i)"
		caseInsensitivePrefix = "(?i)"
	)
	caseSensitive := wildcard.CaseInsensitive
	switch {
	case strings.HasPrefix(p, caseSensitivePrefix):
		caseSensitive = wildcard.CaseSensitive
		p = p[len(caseSensitivePrefix):]
	case strings.HasPrefix(p, caseInsensitivePrefix):
		p = p[len(caseInsensitivePrefix):]
	}
	return wildcard.NewMatcher(p, caseSensitive)
}
