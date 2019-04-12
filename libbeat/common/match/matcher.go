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

import "regexp/syntax"

type Matcher struct {
	stringMatcher
}

type ExactMatcher struct {
	stringMatcher
}

type stringMatcher interface {
	// MatchString tries to find a matching substring.
	MatchString(s string) (matched bool)

	// Match tries to find a matching substring.
	Match(bs []byte) (matched bool)

	// Describe the generator
	String() string
}

func MustCompile(pattern string) Matcher {
	m, err := Compile(pattern)
	if err != nil {
		panic(err)
	}
	return m
}

func MustCompileExact(pattern string) ExactMatcher {
	m, err := CompileExact(pattern)
	if err != nil {
		panic(err)
	}
	return m
}

// CompileString matches a substring only, the input is not interpreted as
// regular expression
func CompileString(in string) (Matcher, error) {
	if in == "" {
		return Matcher{(*emptyStringMatcher)(nil)}, nil
	}
	return Matcher{&substringMatcher{in, []byte(in)}}, nil
}

// Compile regular expression to string matcher. String matcher by default uses
// regular expressions as provided by regexp library, but tries to optimize some
// common cases, replacing expensive patterns with cheaper custom implementations
// or removing terms not necessary for string matching.
func Compile(pattern string) (Matcher, error) {
	regex, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return Matcher{}, err
	}

	regex = optimize(regex).Simplify()
	m, err := compile(regex)
	return Matcher{m}, err
}

func CompileExact(pattern string) (ExactMatcher, error) {
	regex, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return ExactMatcher{}, err
	}

	regex = regex.Simplify()
	if regex.Op != syntax.OpConcat {
		regex = &syntax.Regexp{
			Op: syntax.OpConcat,
			Sub: []*syntax.Regexp{
				patBeginText,
				regex,
				patEndText,
			},
			Flags: regex.Flags,
		}
	} else {
		if !eqPrefixRegex(regex, patBeginText) {
			regex.Sub = append([]*syntax.Regexp{patBeginText}, regex.Sub...)
		}
		if !eqSuffixRegex(regex, patEndText) {
			regex.Sub = append(regex.Sub, patEndText)
		}
	}

	regex = optimize(regex).Simplify()
	m, err := compile(regex)
	return ExactMatcher{m}, err
}

func (m *Matcher) Unpack(s string) error {
	tmp, err := Compile(s)
	if err != nil {
		return err
	}

	*m = tmp
	return nil
}

// MatchAnyString succeeds if any string in the given array contains a match.
func (m *Matcher) MatchAnyString(strs interface{}) bool {
	return matchAnyStrings(m.stringMatcher, strs)
}

// MatchAllStrings succeeds if all strings in the given array contain a match.
func (m *Matcher) MatchAllStrings(strs interface{}) bool {
	return matchAllStrings(m.stringMatcher, strs)
}

// MatchAnyString succeeds if any string in the given array is an exact match.
func (m *ExactMatcher) MatchAnyString(strs interface{}) bool {
	return matchAnyStrings(m.stringMatcher, strs)
}

// MatchAllStrings succeeds if all strings in the given array are an exact match.
func (m *ExactMatcher) MatchAllStrings(strs interface{}) bool {
	return matchAllStrings(m.stringMatcher, strs)
}

func (m *ExactMatcher) Unpack(s string) error {
	tmp, err := CompileExact(s)
	if err != nil {
		return err
	}

	*m = tmp
	return nil
}

func matchAnyStrings(m stringMatcher, strs interface{}) bool {
	switch v := strs.(type) {
	case []interface{}:
		for _, s := range v {
			if str, ok := s.(string); ok && m.MatchString(str) {
				return true
			}
		}
	case []string:
		for _, s := range v {
			if m.MatchString(s) {
				return true
			}
		}
	}
	return false
}

func matchAllStrings(m stringMatcher, strs interface{}) bool {
	switch v := strs.(type) {
	case []interface{}:
		for _, s := range v {
			if str, ok := s.(string); ok && !m.MatchString(str) {
				return false
			}
		}
	case []string:
		for _, s := range v {
			if !m.MatchString(s) {
				return false
			}
		}
	}
	return true
}
