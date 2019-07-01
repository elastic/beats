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

package reader

import (
	"fmt"
	"strings"

	"github.com/coreos/go-systemd/sdjournal"
)

type MatcherConfig struct {
	Equals []string   `config:"equals"`
	And    [][]string `config:"and"`
	Or     [][]string `config:"or"`

	isAnd bool
}

func setupUnitMatchers(j *sdjournal.Journal, units []string, kernel bool) error {
	for _, unit := range units {
		cfg := MatcherConfig{
			Or: [][]string{
				[]string{
					sdjournal.SD_JOURNAL_FIELD_SYSTEMD_UNIT + "=" + unit,
				},
				[]string{
					sdjournal.SD_JOURNAL_FIELD_MESSAGE_ID + "=fc2e22bc6ee647b6b90729ab34a250b1",
					sdjournal.SD_JOURNAL_FIELD_UID + "=1",
					"COREDUMP_UNIT=" + unit,
				},
				[]string{
					sdjournal.SD_JOURNAL_FIELD_PID + "=1",
					"UNIT=" + unit,
				},
				[]string{
					sdjournal.SD_JOURNAL_FIELD_UID + "=1",
					"OBJECT_SYSTEMD_UNIT=" + unit,
				},
			},
		}

		err := setupMatcherConfig(j, cfg)
		if err != nil {
			return fmt.Errorf("error while setting up unit matcher for %s: %+v", unit, err)
		}

	}

	if kernel {
		cfg := MatcherConfig{
			Or: [][]string{
				[]string{
					"_TRANSPORT=kernel",
				},
			},
		}
		err := setupMatcherConfig(j, cfg)
		if err != nil {
			return fmt.Errorf("error while adding kernel transport to matchers: %+v", err)
		}
	}

	return nil
}

func setupSyslogIdentifierMatchers(j *sdjournal.Journal, identifiers []string) error {
	identifierFilters := make([]string, 0)
	for _, identifier := range identifiers {
		identifierFilters = append(identifierFilters, sdjournal.SD_JOURNAL_FIELD_SYSLOG_IDENTIFIER+"="+identifier)
	}

	cfg := MatcherConfig{Or: [][]string{identifierFilters}}

	return setupMatcherConfig(j, cfg)
}

func setupMatcherConfig(j *sdjournal.Journal, c MatcherConfig) error {
	if c.And != nil {
		for _, m := range c.And {
			err := setupMatchers(j, m)
			if err != nil {
				return err
			}

			err = j.AddConjunction()
			if err != nil {
				return err
			}
		}
	}

	if c.Or != nil {
		for _, m := range c.Or {
			err := setupMatchers(j, m)
			if err != nil {
				return err
			}

			err = j.AddDisjunction()
			if err != nil {
				return err
			}
		}
	}

	if c.Equals != nil {
		return setupMatchers(j, c.Equals)
	}
	return nil
}

func setupMatchers(j *sdjournal.Journal, matchers []string) error {
	for _, m := range matchers {
		match, err := transformMatcherString(m)
		err = j.AddMatch(match)
		if err != nil {
			return fmt.Errorf("error while adding matcher to journal: %+v", err)
		}
	}
	return nil
}

func transformMatcherString(m string) (string, error) {
	elems := strings.Split(m, "=")
	if len(elems) != 2 {
		return "", fmt.Errorf("invalid match format: %s", m)
	}

	var p string
	for journalKey, eventField := range journaldEventFields {
		if elems[0] == eventField.name {
			p = journalKey + "=" + elems[1]
		}
	}

	// pass custom fields as is
	if p == "" {
		p = m
	}
	return p, nil
}
