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

package wildcard

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// CaseSensitivity controls the case sensitivity of matching.
type CaseSensitivity bool

// CaseSensitivity values.
const (
	CaseSensitive   CaseSensitivity = true
	CaseInsensitive CaseSensitivity = false
)

// NewMatcher constructs a new wildcard matcher for the given pattern.
//
// If p is the empty string, it will match only the empty string.
// If p is not a valid UTF-8 string, matching behaviour is undefined.
func NewMatcher(p string, caseSensitive CaseSensitivity) *Matcher {
	parts := strings.Split(p, "*")
	m := &Matcher{
		wildcardBegin: strings.HasPrefix(p, "*"),
		wildcardEnd:   strings.HasSuffix(p, "*"),
		caseSensitive: caseSensitive,
	}
	for _, part := range parts {
		if part == "" {
			continue
		}
		if !m.caseSensitive {
			part = strings.ToLower(part)
		}
		m.parts = append(m.parts, part)
	}
	return m
}

// Matcher matches strings against a wildcard pattern with configurable case sensitivity.
type Matcher struct {
	parts         []string
	wildcardBegin bool
	wildcardEnd   bool
	caseSensitive CaseSensitivity
}

// Match reports whether s matches m's wildcard pattern.
func (m *Matcher) Match(s string) bool {
	if len(m.parts) == 0 && !m.wildcardBegin && !m.wildcardEnd {
		return s == ""
	}
	if len(m.parts) == 1 && !m.wildcardBegin && !m.wildcardEnd {
		if m.caseSensitive {
			return s == m.parts[0]
		}
		return len(s) == len(m.parts[0]) && hasPrefixLower(s, m.parts[0]) == 0
	}
	parts := m.parts
	if !m.wildcardEnd && len(parts) > 0 {
		part := parts[len(parts)-1]
		if m.caseSensitive {
			if !strings.HasSuffix(s, part) {
				return false
			}
		} else {
			if len(s) < len(part) {
				return false
			}
			if hasPrefixLower(s[len(s)-len(part):], part) != 0 {
				return false
			}
		}
		parts = parts[:len(parts)-1]
	}
	for i, part := range parts {
		j := -1
		if m.caseSensitive {
			if i > 0 || m.wildcardBegin {
				j = strings.Index(s, part)
			} else {
				if !strings.HasPrefix(s, part) {
					return false
				}
				j = 0
			}
		} else {
			off := 0
			for j == -1 && len(s)-off >= len(part) {
				skip := hasPrefixLower(s[off:], part)
				if skip == 0 {
					j = off
				} else {
					if i == 0 && !m.wildcardBegin {
						return false
					}
					off += skip
				}
			}
		}
		if j == -1 {
			return false
		}
		s = s[j+len(part):]
	}
	return true
}

// hasPrefixLower reports whether or not s begins with prefixLower,
// returning 0 if it does, and the number of bytes representing the
// first rune in s otherwise.
func hasPrefixLower(s, prefixLower string) (skip int) {
	var firstSize int
	for i, r := range prefixLower {
		r2, size := utf8.DecodeRuneInString(s[i:])
		if firstSize == 0 {
			firstSize = size
		}
		if r2 != r && r2 != unicode.ToUpper(r) {
			return firstSize
		}
	}
	return 0
}
