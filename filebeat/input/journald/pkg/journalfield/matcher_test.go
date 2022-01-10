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

//go:build linux && cgo
// +build linux,cgo

package journalfield

import (
	"testing"

	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/stretchr/testify/require"
)

func TestApplyMatchersOr(t *testing.T) {
	cases := map[string]struct {
		filters []string
		wantErr bool
	}{
		"correct filter expression": {
			filters: []string{"systemd.unit=nginx"},
			wantErr: false,
		},
		"custom field": {
			filters: []string{"_MY_CUSTOM_FIELD=value"},
			wantErr: false,
		},
		"mixed filters": {
			filters: []string{"systemd.unit=nginx", "_MY_CUSTOM_FIELD=value"},
			wantErr: false,
		},
		"same field filters": {
			filters: []string{"systemd.unit=nginx", "systemd.unit=mysql"},
			wantErr: false,
		},
		"incorrect separator": {
			filters: []string{"systemd.unit~nginx"},
			wantErr: true,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			journal, err := sdjournal.NewJournal()
			if err != nil {
				t.Fatalf("error while creating test journal: %v", err)
			}
			defer journal.Close()

			matchers := make([]Matcher, len(test.filters))
			for i, str := range test.filters {
				m, err := BuildMatcher(str)
				if err != nil && !test.wantErr {
					t.Fatalf("unexpected error compiling the filters: %v", err)
				}
				matchers[i] = m
			}

			// double check if journald likes our filters
			err = ApplyMatchersOr(journal, matchers)
			fail := (test.wantErr && err == nil) || (!test.wantErr && err != nil)
			if fail {
				t.Errorf("unexpected outcome: error: '%v', expected error: %v", err, test.wantErr)
			}
		})
	}
}

func TestApplySyslogIdentifier(t *testing.T) {
	journal, err := sdjournal.NewJournal()
	if err != nil {
		t.Fatalf("error while creating test journal: %v", err)
	}
	defer journal.Close()

	err = ApplySyslogIdentifierMatcher(journal, []string{"audit"})
	require.NoError(t, err)
}

func TestApplyUnit(t *testing.T) {
	journal, err := sdjournal.NewJournal()
	if err != nil {
		t.Fatalf("error while creating test journal: %v", err)
	}
	defer journal.Close()

	err = ApplyUnitMatchers(journal, []string{"docker.service"})
	require.NoError(t, err)
}
