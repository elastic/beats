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

package journalfield

import (
	"fmt"
	"strings"
)

// Matcher is a single field condition for filtering journal entries.
//
// The Matcher type can be used as is with Beats configuration unpacking. The
// internal default conversion table will be used, similar to BuildMatcher.
type Matcher struct {
	str string
}

// MatcherBuilder can be used to create a custom builder for creating matchers
// based on a conversion table.
type MatcherBuilder struct {
	Conversions map[string]Conversion
}

type journal interface {
	AddMatch(string) error
	AddDisjunction() error
}

var defaultBuilder = MatcherBuilder{Conversions: journaldEventFields}

// Build creates a new Matcher using the configured conversion table.
// If no table has been configured the internal default table will be used.
func (b MatcherBuilder) Build(in string) (Matcher, error) {
	elems := strings.Split(in, "=")
	if len(elems) != 2 {
		return Matcher{}, fmt.Errorf("invalid match format: %s", in)
	}

	conversions := b.Conversions
	if conversions == nil {
		conversions = journaldEventFields
	}

	for journalKey, eventField := range conversions {
		for _, name := range eventField.Names {
			if elems[0] == name {
				return Matcher{journalKey + "=" + elems[1]}, nil
			}
		}
	}

	// pass custom fields as is
	return Matcher{in}, nil
}

// BuildMatcher creates a Matcher from a field filter string.
func BuildMatcher(in string) (Matcher, error) {
	return defaultBuilder.Build(in)
}

// IsValid returns true if the matcher was initialized correctly.
func (m Matcher) IsValid() bool { return m.str != "" }

// String returns the string representation of the field match.
func (m Matcher) String() string { return m.str }

// Apply adds the field match to an open journal for filtering.
func (m Matcher) Apply(j journal) error {
	if !m.IsValid() {
		return fmt.Errorf("can not apply invalid matcher to a journal")
	}

	err := j.AddMatch(m.str)
	if err != nil {
		return fmt.Errorf("error adding match '%s' to journal: %v", m.str, err)
	}
	return nil
}

// Unpack initializes the Matcher from a given string representation. Unpack
// fails if the input string is invalid.
// Unpack can be used with Beats configuration loading.
func (m *Matcher) Unpack(value string) error {
	tmp, err := BuildMatcher(value)
	if err != nil {
		return err
	}
	*m = tmp
	return nil
}

// ApplyMatchersOr adds a list of matchers to a journal, calling AddDisjunction after each matcher being added.
func ApplyMatchersOr(j journal, matchers []Matcher) error {
	for _, m := range matchers {
		if err := m.Apply(j); err != nil {
			return err
		}

		if err := j.AddDisjunction(); err != nil {
			return fmt.Errorf("error adding disjunction to journal: %v", err)
		}
	}

	return nil
}
