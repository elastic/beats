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

// IncludeMatches stores the advanced matching configuratio
// provided by the user.
type IncludeMatches struct {
	Matches []Matcher        `config:"match"`
	AND     []IncludeMatches `config:"and"`
	OR      []IncludeMatches `config:"or"`
}

type journal interface {
	AddMatch(string) error
	AddDisjunction() error
	AddConjunction() error
}

var (
	defaultBuilder = MatcherBuilder{Conversions: journaldEventFields}
	coreDumpMsgID  = MustBuildMatcher("message_id=fc2e22bc6ee647b6b90729ab34a250b1") // matcher for messages from coredumps
	journaldUID    = MustBuildMatcher("journald.uid=0")                              // matcher for messages from root (UID 0)
	journaldPID    = MustBuildMatcher("journald.pid=1")                              // matcher for messages from init process (PID 1)
)

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

func MustBuildMatcher(in string) Matcher {
	m, err := BuildMatcher(in)
	if err != nil {
		panic(err)
	}
	return m
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

// ApplyUnitMatchers adds unit based filtering to the journal reader.
// Filtering is similar to what systemd does here:
// https://github.com/systemd/systemd/blob/641e2124de6047e6010cd2925ea22fba29b25309/src/shared/logs-show.c#L1409-L1455
func ApplyUnitMatchers(j journal, units []string) error {
	for _, unit := range units {
		systemdUnit, err := BuildMatcher("systemd.unit=" + unit)
		if err != nil {
			return fmt.Errorf("failed to build matcher for _SYSTEMD_UNIT: %+w", err)
		}
		coredumpUnit, err := BuildMatcher("journald.coredump.unit=" + unit)
		if err != nil {
			return fmt.Errorf("failed to build matcher for COREDUMP_UNIT: %+w", err)
		}
		journaldUnit, err := BuildMatcher("journald.unit=" + unit)
		if err != nil {
			return fmt.Errorf("failed to build matcher for UNIT: %+w", err)
		}
		journaldObjectUnit, err := BuildMatcher("journald.object.systemd.unit=" + unit)
		if err != nil {
			return fmt.Errorf("failed to build matcher for OBJECT_SYSTEMD_UNIT: %+w", err)
		}

		matchers := [][]Matcher{
			// match for the messages of the service
			{
				systemdUnit,
			},
			// match for the coredumps of the service
			{
				coreDumpMsgID,
				journaldUID,
				coredumpUnit,
			},
			// match for messages about the service with PID value of 1
			{
				journaldPID,
				journaldUnit,
			},
			// match for messages about the service from authorized daemons
			{
				journaldUID,
				journaldObjectUnit,
			},
		}
		if strings.HasSuffix(unit, ".slice") {
			if sliceMatcher, err := BuildMatcher("systemd.slice=" + unit); err != nil {
				matchers = append(matchers, []Matcher{sliceMatcher})
			}
		}

		for _, m := range matchers {
			if err := ApplyMatchersOr(j, m); err != nil {
				return fmt.Errorf("error while setting up unit matcher for %s: %+v", unit, err)
			}
		}

	}

	return nil
}

// ApplyTransportMatcher adds matchers for the configured transports.
func ApplyTransportMatcher(j journal, transports []string) error {
	if len(transports) == 0 {
		return nil
	}

	transportMatchers := make([]Matcher, len(transports))
	for i, transport := range transports {
		transportMatcher, err := BuildMatcher("_TRANSPORT=" + transport)
		if err != nil {
			return err
		}
		transportMatchers[i] = transportMatcher
	}
	if err := ApplyMatchersOr(j, transportMatchers); err != nil {
		return fmt.Errorf("error while adding %+v transport to matchers: %+v", transports, err)
	}
	return nil
}

// ApplySyslogIdentifierMatcher adds syslog identifier filtering to the journal reader.
func ApplySyslogIdentifierMatcher(j journal, identifiers []string) error {
	identifierMatchers := make([]Matcher, len(identifiers))
	for i, identifier := range identifiers {
		identifierMatchers[i] = MustBuildMatcher("syslog.identifier=" + identifier)
	}

	return ApplyMatchersOr(j, identifierMatchers)
}

// ApplyIncludeMatches adds advanced filtering to journals.
func ApplyIncludeMatches(j journal, m IncludeMatches) error {
	for _, or := range m.OR {
		if err := ApplyIncludeMatches(j, or); err != nil {
			return err
		}
		if err := j.AddDisjunction(); err != nil {
			return fmt.Errorf("error adding disjunction to journal: %v", err)
		}
	}

	for _, and := range m.AND {
		if err := ApplyIncludeMatches(j, and); err != nil {
			return err
		}
		if err := j.AddConjunction(); err != nil {
			return fmt.Errorf("error adding conjunction to journal: %v", err)
		}
	}

	for _, match := range m.Matches {
		if err := match.Apply(j); err != nil {
			return fmt.Errorf("failed to apply %s expression: %+v", match.str, err)
		}
	}

	return nil
}
