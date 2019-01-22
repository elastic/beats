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

//+build linux,cgo

package reader

import (
	"reflect"
	"testing"

	"github.com/coreos/go-systemd/sdjournal"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/journalbeat/checkpoint"
	"github.com/elastic/beats/journalbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type ToEventTestCase struct {
	entry          sdjournal.JournalEntry
	expectedFields common.MapStr
}

type SetupMatchesTestCase struct {
	matches     []string
	expectError bool
}

func TestToEvent(t *testing.T) {
	tests := []ToEventTestCase{
		// field name from fields.go
		ToEventTestCase{
			entry: sdjournal.JournalEntry{
				Fields: map[string]string{
					sdjournal.SD_JOURNAL_FIELD_BOOT_ID: "123456",
				},
			},
			expectedFields: common.MapStr{
				"host": common.MapStr{
					"boot_id": "123456",
				},
			},
		},
		// custom field
		ToEventTestCase{
			entry: sdjournal.JournalEntry{
				Fields: map[string]string{
					"my_custom_field": "value",
				},
			},
			expectedFields: common.MapStr{
				"journald": common.MapStr{
					"custom": common.MapStr{
						"my_custom_field": "value",
					},
				},
			},
		},
		// dropped field
		ToEventTestCase{
			entry: sdjournal.JournalEntry{
				Fields: map[string]string{
					"_SOURCE_MONOTONIC_TIMESTAMP": "value",
				},
			},
			expectedFields: common.MapStr{},
		},
	}

	instance.SetupJournalMetrics()
	r, err := NewLocal(Config{Path: "dummy.journal"}, nil, checkpoint.JournalState{}, logp.NewLogger("test"))
	if err != nil {
		t.Fatalf("error creating test journal: %v", err)
	}
	for _, test := range tests {
		event := r.toEvent(&test.entry)
		event.Fields.Delete("event")
		assert.True(t, reflect.DeepEqual(event.Fields, test.expectedFields))
	}
}

func TestSetupMatches(t *testing.T) {
	tests := []SetupMatchesTestCase{
		// correct filter expression
		SetupMatchesTestCase{
			matches:     []string{"systemd.unit=nginx"},
			expectError: false,
		},
		// custom field
		SetupMatchesTestCase{
			matches:     []string{"_MY_CUSTOM_FIELD=value"},
			expectError: false,
		},
		// incorrect separator
		SetupMatchesTestCase{
			matches:     []string{"systemd.unit~nginx"},
			expectError: true,
		},
	}
	journal, err := sdjournal.NewJournal()
	if err != nil {
		t.Fatalf("error while creating test journal: %v", err)
	}

	for _, test := range tests {
		err = setupMatches(journal, test.matches)
		if err != nil && !test.expectError {
			t.Errorf("unexpected outcome of setupMatches: error: '%v', expected error: %v", err, test.expectError)
		}
	}
}
