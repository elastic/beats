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

package match

import (
	"reflect"
	"testing"
)

func TestMatchers(t *testing.T) {
	typeOf := func(v interface{}) reflect.Type {
		return reflect.TypeOf(v)
	}

	tests := []struct {
		pattern     string
		matcherType reflect.Type
		matches     []string
		noMatches   []string
	}{
		{
			`.*`,
			typeOf((*matchAny)(nil)),
			[]string{
				"any matches always",
			},
			nil,
		},
		{
			`^$`,
			typeOf((*emptyStringMatcher)(nil)),
			[]string{""},
			[]string{"not empty"},
		},
		{
			`^\s*$`,
			typeOf((*emptyWhiteStringMatcher)(nil)),
			[]string{"", " ", "   ", "\t", "\n"},
			[]string{"not empty"},
		},
		{
			`substring`,
			typeOf((*substringMatcher)(nil)),
			[]string{
				"has substring in middle",
				"substring at beginning",
				"ends with substring",
			},
			[]string{"missing sub-string"},
		},
		{
			`^.*substring`,
			typeOf((*substringMatcher)(nil)),
			[]string{
				"has substring in middle",
				"substring at beginning",
				"ends with substring",
			},
			[]string{"missing sub-string"},
		},
		{
			`substring.*$`,
			typeOf((*substringMatcher)(nil)),
			[]string{
				"has substring in middle",
				"substring at beginning",
				"ends with substring",
			},
			[]string{"missing sub-string"},
		},
		{
			`^.*substring.*$`,
			typeOf((*substringMatcher)(nil)),
			[]string{
				"has substring in middle",
				"substring at beginning",
				"ends with substring",
			},
			[]string{"missing sub-string"},
		},
		{
			`^equals$`,
			typeOf((*equalsMatcher)(nil)),
			[]string{"equals"},
			[]string{"not equals"},
		},
		{
			`(alt|substring)`,
			typeOf((*altSubstringMatcher)(nil)),
			[]string{
				"has alt in middle",
				"alt at beginning",
				"uses substring",
			},
			[]string{"missing sub-string"},
		},
		{
			`alt|substring`,
			typeOf((*altSubstringMatcher)(nil)),
			[]string{
				"has alt in middle",
				"alt at beginning",
				"uses substring",
			},
			[]string{"missing sub-string"},
		},
		{
			`^prefix`,
			typeOf((*prefixMatcher)(nil)),
			[]string{"prefix string match"},
			[]string{"missing prefix string"},
		},
		{
			`^(DEBUG|INFO|ERROR)`,
			typeOf((*altPrefixMatcher)(nil)),
			[]string{
				"DEBUG - should match",
				"INFO - should match too",
				"ERROR - yep",
			},
			[]string{
				"This should not match",
			},
		},
		{
			`^\d\d\d\d-\d\d-\d\d`,
			typeOf((*prefixNumDate)(nil)),
			[]string{
				"2017-01-02 should match",
				"2017-01-03 should also match",
			},
			[]string{
				"- 2017-01-02 should not match",
				"fail",
			},
		},
		{
			`^\d{4}-\d{2}-\d{2}`,
			typeOf((*prefixNumDate)(nil)),
			[]string{
				"2017-01-02 should match",
				"2017-01-03 should also match",
			},
			[]string{
				"- 2017-01-02 should not match",
				"fail",
			},
		},
		{
			`^(\d{2}){2}-\d{2}-\d{2}`,
			typeOf((*prefixNumDate)(nil)),
			[]string{
				"2017-01-02 should match",
				"2017-01-03 should also match",
			},
			[]string{
				"- 2017-01-02 should not match",
				"fail",
			},
		},
		{
			`^\d{4}-\d{2}-\d{2} - `,
			typeOf((*prefixNumDate)(nil)),
			[]string{
				"2017-01-02 - should match",
				"2017-01-03 - should also match",
			},
			[]string{
				"- 2017-01-02 should not match",
				"fail",
			},
		},
		{
			`^20\d{2}-\d{2}-\d{2}`,
			typeOf((*prefixNumDate)(nil)),
			[]string{
				"2017-01-02 should match",
				"2017-01-03 should also match",
			},
			[]string{
				"- 2017-01-02 should not match",
				"fail",
			},
		},
		{
			`^20\d{2}-\d{2}-\d{2} \d{2}:\d{2}`,
			typeOf((*prefixNumDate)(nil)),
			[]string{
				"2017-01-02 10:10 should match",
				"2017-01-03 10:11 should also match",
			},
			[]string{
				"- 2017-01-02 should not match",
				"fail",
			},
		},
	}

	for i, test := range tests {
		t.Logf("run test (%v): %v", i, test.pattern)

		matcher, err := Compile(test.pattern)
		if err != nil {
			t.Error(err)
			continue
		}

		t.Logf("  matcher: %v", matcher)

		matcherType := reflect.TypeOf(matcher.stringMatcher)
		if matcherType != test.matcherType {
			t.Errorf("  Matcher type mismatch (expected=%v, actual=%v)",
				test.matcherType,
				matcherType,
			)
		}

		for _, content := range test.matches {
			if !matcher.MatchString(content) {
				t.Errorf("  failed to match string: '%v'", content)
				continue
			}

			if !matcher.Match([]byte(content)) {
				t.Errorf("  failed to match byte string: '%v'", content)
				continue
			}
		}

		for _, content := range test.noMatches {
			if matcher.MatchString(content) {
				t.Errorf("  should not match string: '%v'", content)
				continue
			}

			if matcher.Match([]byte(content)) {
				t.Errorf("  should not match string: '%v'", content)
				continue
			}
		}
	}
}

func TestExactMatchers(t *testing.T) {
	typeOf := func(v interface{}) reflect.Type {
		return reflect.TypeOf(v)
	}

	tests := []struct {
		pattern     string
		matcherType reflect.Type
		matches     []string
		noMatches   []string
	}{
		{
			`.*`,
			typeOf((*matchAny)(nil)),
			[]string{
				"any matches always",
			},
			nil,
		},
		{
			`^$`,
			typeOf((*emptyStringMatcher)(nil)),
			[]string{""},
			[]string{"not empty"},
		},
		{
			`^\s*$`,
			typeOf((*emptyWhiteStringMatcher)(nil)),
			[]string{"", " ", "   ", "\t", "\n"},
			[]string{"not empty"},
		},
		{
			`.*substring.*`,
			typeOf((*substringMatcher)(nil)),
			[]string{
				"has substring in middle",
				"substring at beginning",
				"ends with substring",
			},
			[]string{"missing sub-string"},
		},
		{
			`^.*substring.*`,
			typeOf((*substringMatcher)(nil)),
			[]string{
				"has substring in middle",
				"substring at beginning",
				"ends with substring",
			},
			[]string{"missing sub-string"},
		},
		{
			`.*substring.*$`,
			typeOf((*substringMatcher)(nil)),
			[]string{
				"has substring in middle",
				"substring at beginning",
				"ends with substring",
			},
			[]string{"missing sub-string"},
		},
		{
			`^.*substring.*$`,
			typeOf((*substringMatcher)(nil)),
			[]string{
				"has substring in middle",
				"substring at beginning",
				"ends with substring",
			},
			[]string{"missing sub-string"},
		},
		{
			`equals`,
			typeOf((*equalsMatcher)(nil)),
			[]string{"equals"},
			[]string{"not equals"},
		},
		{
			`^equals`,
			typeOf((*equalsMatcher)(nil)),
			[]string{"equals"},
			[]string{"not equals"},
		},
		{
			`equals$`,
			typeOf((*equalsMatcher)(nil)),
			[]string{"equals"},
			[]string{"not equals"},
		},
		{
			`DEBUG|INFO`,
			typeOf((*oneOfMatcher)(nil)),
			[]string{
				"DEBUG",
				"INFO",
			},
			[]string{"none"},
		},
	}

	for i, test := range tests {
		t.Logf("run test (%v): %v", i, test.pattern)

		matcher, err := CompileExact(test.pattern)
		if err != nil {
			t.Error(err)
			continue
		}

		t.Logf("  matcher: %v", matcher)

		matcherType := reflect.TypeOf(matcher.stringMatcher)
		if matcherType != test.matcherType {
			t.Errorf("  Matcher type mismatch (expected=%v, actual=%v)",
				test.matcherType,
				matcherType,
			)
		}

		for _, content := range test.matches {
			if !matcher.MatchString(content) {
				t.Errorf("  failed to match string: '%v'", content)
				continue
			}

			if !matcher.Match([]byte(content)) {
				t.Errorf("  failed to match byte string: '%v'", content)
				continue
			}
		}

		for _, content := range test.noMatches {
			if matcher.MatchString(content) {
				t.Errorf("  should not match string: '%v'", content)
				continue
			}

			if matcher.Match([]byte(content)) {
				t.Errorf("  should not match string: '%v'", content)
				continue
			}
		}
	}
}
