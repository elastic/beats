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

//go:build linux

package journald

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// How to write to journal from CLI:
// https://www.baeldung.com/linux/systemd-journal-message-terminal

// TestGenerateJournalEntries generates entries in the user's journal.
// It is kept commented out at the top of the file as reference and
// easy access.
//
// How to generate a journal file with only the entries you want:
//  1. Get a VM
//  2. Uncomment and run this test as a normal user just to make sure you
//     you can write to the journal and find the file
//  3. Find the journal file, usually at
//     /var/log/journal/<machine ID>/user-1000.journal
//  4. Rotate the journal
//  5. Clean and rotate the journal
//     sudo journalctl  --vacuum-time=1s
//     sudo journalctl --rotate
//  6. Copy the journal file somewhere else
//     cp /var/log/journal/21282bcb80a74c08a0d14a047372256c/user-1000.journal /tmp/foo.journal
//  7. Read the journal file:
//     journalctl --file=/tmp/foo.journal -n 100
//  8. Read the journal with all fields as JSON
//     journalctl --file=/tmp/foo.journal -n 100 -o json
// func TestGenerateJournalEntries(t *testing.T) {
// 	// To run this test you need to add the necessary imports.
// 	// 1. Go get:
// 	//   go get github.com/ssgreg/journald
// 	// 2. Add the following import:
// 	//   journaldlogger "github.com/ssgreg/journald"
// 	// 3. Uncomment and run the test:
// 	//   go test -count=1 -run=TestGenerate
// 	fields := []map[string]any{
// 		{
// 			"BAR": "bar",
// 		},
// 		{
// 			"FOO": "foo",
// 		},
// 		{
// 			"BAR": "bar",
// 			"FOO": "foo",
// 		},
// 		{
// 			"FOO_BAR": "foo bar",
// 		},
// 		{
// 			"ANSWER":   42,
// 			"BAR":      "bar",
// 			"FOO":      "foo",
// 			"FOO_BAR":  "foo bar",
// 			"QUESTION": "Answer to the Ultimate Question of Life, The Universe, and Everything",
// 		},
// 	}

// 	for _, m := range fields {
// 		if err := journaldlogger.Send("Hello World!", journaldlogger.PriorityInfo, m); err != nil {
// 			t.Fatal(err)
// 		}
// 	}
// }

func TestInputFieldsTranslation(t *testing.T) {
	// A few random keys to verify
	keysToCheck := map[string]string{
		"systemd.user_unit": "log-service.service",
		"process.pid":       "2084785",
		"systemd.transport": "stdout",
		"host.hostname":     "x-wing",
	}

	testCases := map[string]struct {
		saveRemoteHostname bool
	}{
		"Save hostname enabled":  {saveRemoteHostname: true},
		"Save hostname disabled": {saveRemoteHostname: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			env := newInputTestingEnvironment(t)

			inp := env.mustCreateInput(mapstr.M{
				"paths":                 []string{path.Join("testdata", "input-multiline-parser.journal")},
				"include_matches.match": []string{"_SYSTEMD_USER_UNIT=log-service.service"},
				"save_remote_hostname":  tc.saveRemoteHostname,
			})

			ctx, cancelInput := context.WithCancel(context.Background())
			env.startInput(ctx, inp)
			env.waitUntilEventCount(6)

			for eventIdx, event := range env.pipeline.clients[0].GetEvents() {
				for k, v := range keysToCheck {
					got, err := event.Fields.GetValue(k)
					if err == nil {
						if got, want := fmt.Sprint(got), v; got != want {
							t.Errorf("expecting key %q to have value '%#v', but got '%#v' instead", k, want, got)
						}
					} else {
						t.Errorf("key %q not found on event %d", k, eventIdx)
					}
				}
				if tc.saveRemoteHostname {
					v, err := event.Fields.GetValue("log.source.address")
					if err != nil {
						t.Errorf("key 'log.source.address' not found on evet %d", eventIdx)
					}

					if got, want := fmt.Sprint(v), "x-wing"; got != want {
						t.Errorf("expecting key 'log.source.address' to have value '%#v', but got '%#v' instead", want, got)
					}
				}
			}
			cancelInput()
		})
	}
}

// TestCompareGoSystemdWithJournalctl ensures the new implementation produces
// events in the same format as the original one. We use the events from the
// already existing journal file 'input-multiline-parser.journal'
//
// The following fields are not currently tested:
// __MONOTONIC_TIMESTAMP - it seems to be ignored
// __CURSOR - it will be added to the registry and tested once we have tests for it
func TestCompareGoSystemdWithJournalctl(t *testing.T) {
	env := newInputTestingEnvironment(t)
	inp := env.mustCreateInput(mapstr.M{
		"paths":      []string{path.Join("testdata", "input-multiline-parser.journal")},
		"journalctl": true,
		"seek":       "head",
	})

	ctx2, cancelInput2 := context.WithCancel(context.Background())
	defer cancelInput2()

	env.startInput(ctx2, inp)
	env.waitUntilEventCount(8)

	rawEvents := env.pipeline.GetAllEvents()
	events := []beat.Event{}
	for _, evt := range rawEvents {
		evt.Delete("event.created")
		// Fields that the go-systemd version did not add
		evt.Delete("journald.custom.seqnum")
		evt.Delete("journald.custom.seqnum_id")
		evt.Delete("journald.custom.realtime_timestamp")
		// Marshal and Unmarshal because of type changes
		// We ignore errors as those types can always marshal and unmarshal
		data, _ := json.Marshal(evt)
		newEvt := beat.Event{}
		json.Unmarshal(data, &newEvt)
		if newEvt.Meta == nil {
			// the golden file has it as an empty map
			newEvt.Meta = mapstr.M{}
		}
		events = append(events, newEvt)
	}

	// Read JSON events
	goldenEvents := []beat.Event{}
	data, err := os.ReadFile(filepath.Join("testdata", "input-multiline-parser-events.json"))
	if err != nil {
		t.Fatalf("cannot read golden file: %s", err)
	}

	if err := json.Unmarshal(data, &goldenEvents); err != nil {
		t.Fatalf("cannot unmarshal golden events: %s", err)
	}

	if len(events) != len(goldenEvents) {
		t.Fatalf("expecting %d events, got %d", len(goldenEvents), len(events))
	}

	require.EqualValues(t, goldenEvents, events, "events do not match reference")
}

func TestMatchers(t *testing.T) {
	t.Skip("Skipping the tests until we fix the matchers")
	testCases := []struct {
		name           string
		matchers       map[string]any
		expectedEvents int
	}{
		{
			name: "single marcher",
			matchers: map[string]any{
				"match": []string{
					"FOO=foo",
				},
			},
			expectedEvents: 3,
		},
		{
			name: "two matches, works as AND",
			matchers: map[string]any{
				"match": []string{
					"FOO=foo",
					"BAR=bar",
				},
			},
			expectedEvents: 2,
		},
		{
			name: "AND matches",
			matchers: map[string]any{
				"and": []any{
					map[string]any{
						"match": []string{
							"FOO=foo",
						},
					},
					map[string]any{
						"match": []string{
							"BAR=bar",
						},
					},
				},
			},
			expectedEvents: 2,
		},
		{
			name: "OR matches",
			matchers: map[string]any{
				"or": []any{
					map[string]any{
						"match": []string{
							"FOO=foo",
						},
					},
					map[string]any{
						"match": []string{
							"BAR=bar",
						},
					},
				},
			},
			expectedEvents: 4,
		},
		{
			name: "OR-EQUALS matches",
			matchers: map[string]any{
				"or": []any{
					map[string]any{
						"equals": []string{
							"FOO=foo",
						},
					},
				},
			},
			expectedEvents: 4,
		},
		{
			name: "A OR (B AND C)",
			matchers: map[string]any{
				"or": []any{
					map[string]any{
						"match": []string{
							"FOO_BAR=foo bar",
						},
						"and": []any{
							map[string]any{
								"match": []string{
									"FOO=foo",
								},
							},
							map[string]any{
								"match": []string{
									"BAR=bar",
								},
							},
						},
					},
				},
			},
			expectedEvents: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Printf("==================== %s ==========\n", t.Name())
			env := newInputTestingEnvironment(t)
			inp := env.mustCreateInput(mapstr.M{
				"paths":           []string{path.Join("testdata", "matchers.journal")},
				"include_matches": tc.matchers,
				// "journalctl": true,
			})

			ctx, cancelInput2 := context.WithCancel(context.Background())
			defer cancelInput2()

			env.startInput(ctx, inp)
			env.waitUntilEventCount(tc.expectedEvents)
			for _, evt := range env.pipeline.GetAllEvents() {
				// fmt.Println(evt.Fields.StringToPrint())
				fields, _ := evt.Fields.GetValue("journald.custom")
				fmt.Println(fields)
			}
		})
	}
}
