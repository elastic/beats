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
	"compress/gzip"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalctl"
	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalfield"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestInputCanReadAllBoots(t *testing.T) {
	out := decompress(t, filepath.Join("testdata", "multiple-boots.journal.gz"))

	env := newInputTestingEnvironment(t)
	cfg := mapstr.M{
		"paths": []string{out},
	}
	inp := env.mustCreateInput(cfg)

	ctx, cancelInput := context.WithCancel(context.Background())
	t.Cleanup(cancelInput)

	env.startInput(ctx, inp)
	env.waitUntilEventCount(6)
}

func TestInputFieldsTranslation(t *testing.T) {
	out := decompress(t, filepath.Join("testdata", "input-multiline-parser.journal.gz"))
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
				"paths":                 []string{out},
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
// __CURSOR - it is added to the registry and there are other tests for it
// __MONOTONIC_TIMESTAMP - it is part of the cursor
func TestCompareGoSystemdWithJournalctl(t *testing.T) {
	out := decompress(t, filepath.Join("testdata", "input-multiline-parser.journal.gz"))
	env := newInputTestingEnvironment(t)
	inp := env.mustCreateInput(mapstr.M{
		"paths": []string{out},
		"seek":  "head",
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	defer cancelInput()

	env.startInput(ctx, inp)
	env.waitUntilEventCount(8)

	rawEvents := env.pipeline.GetAllEvents()
	events := []beat.Event{}
	for _, evt := range rawEvents {
		_ = evt.Delete("event.created")
		// Fields that the go-systemd version did not add
		_ = evt.Delete("journald.custom.seqnum")
		_ = evt.Delete("journald.custom.seqnum_id")
		_ = evt.Delete("journald.custom.realtime_timestamp")
		// Marshal and Unmarshal because of type changes
		// We ignore errors as those types can always marshal and unmarshal
		data, _ := json.Marshal(evt)
		newEvt := beat.Event{}
		json.Unmarshal(data, &newEvt) //nolint: errcheck // this will never fail
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

	// The timestamps can have different locations set, but still be equal,
	// this causes the require.EqualValues to fail, so we compare them manually
	// and set them all to the same time.
	for i, goldEvent := range goldenEvents {
		// We have compared the length already, both slices have
		// have the same number of elements
		evt := events[i]
		if !goldEvent.Timestamp.Equal(evt.Timestamp) {
			t.Errorf(
				"event %d timestamp is different than expected. Expecting %s, got %s",
				i, goldEvent.Timestamp.String(), evt.Timestamp.String())
		}

		events[i].Timestamp = goldEvent.Timestamp
	}

	require.EqualValues(t, goldenEvents, events, "events do not match reference")
}

func TestMatchers(t *testing.T) {
	out := decompress(t, filepath.Join("testdata", "matchers.journal.gz"))
	// If this test fails, uncomment the following line to see the debug logs
	// logp.DevelopmentSetup()
	testCases := []struct {
		name           string
		matchers       map[string]any
		confiFields    map[string]any
		expectedEvents int
	}{
		{ // FOO=foo
			name: "single marcher",
			matchers: map[string]any{
				"match": []string{
					"FOO=foo",
				},
			},
			expectedEvents: 2,
		},
		{ // FOO=foo AND BAR=bar
			name: "different keys work as AND",
			matchers: map[string]any{
				"match": []string{
					"FOO=foo",
					"BAR=bar",
				},
			},
			expectedEvents: 1,
		},
		{ // FOO_BAR=foo OR FOO_BAR=bar
			name: "same keys work as OR",
			matchers: map[string]any{
				"match": []string{
					"FOO_BAR=foo",
					"FOO_BAR=bar",
				},
			},
			expectedEvents: 2,
		},
		{ // (FOO_BAR=foo OR FOO_BAR=bar) AND message="message 4"
			name: "same keys work as OR, AND the odd one, one match",
			matchers: map[string]any{
				"match": []string{
					"FOO_BAR=foo",
					"FOO_BAR=bar",
					"MESSAGE=message 4",
				},
			},
			expectedEvents: 1,
		},
		{ // (FOO_BAR=foo OR FOO_BAR=bar) AND message="message 1"
			name: "same keys work as OR, AND the odd one. No matches",
			matchers: map[string]any{
				"match": []string{
					"FOO_BAR=foo",
					"FOO_BAR=bar",
					"MESSAGE=message 1",
				},
			},
			expectedEvents: 0,
		},
		{
			name:     "transport: journal",
			matchers: map[string]any{},
			confiFields: map[string]any{
				"transports": []string{"journal"},
			},
			expectedEvents: 6,
		},
		{
			name:     "syslog identifier: sudo",
			matchers: map[string]any{},
			confiFields: map[string]any{
				"syslog_identifiers": []string{"sudo"},
			},
			expectedEvents: 1,
		},
		{
			name:     "unit",
			matchers: map[string]any{},
			confiFields: map[string]any{
				"units": []string{"session-39.scope"},
			},
			expectedEvents: 7,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			env := newInputTestingEnvironment(t)
			cfg := mapstr.M{
				"paths":           []string{out},
				"include_matches": tc.matchers,
			}
			cfg.Update(mapstr.M(tc.confiFields))
			inp := env.mustCreateInput(cfg)

			ctx, cancelInput := context.WithCancel(context.Background())
			defer cancelInput()

			env.startInput(ctx, inp)
			env.waitUntilEventCount(tc.expectedEvents)
		})
	}
}

//go:embed pkg/journalctl/testdata/corner-cases.json
var msgByteArrayJSON []byte

func TestReaderAdapterCanHandleNonStringFields(t *testing.T) {
	testCases := []map[string]any{}
	if err := json.Unmarshal(msgByteArrayJSON, &testCases); err != nil {
		t.Fatalf("could not unmarshal the contents from 'testdata/message-byte-array.json' into map[string]any: %s", err)
	}

	for idx, event := range testCases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			mock := journalReaderMock{
				NextFunc: func(cancel v2.Canceler) (journalctl.JournalEntry, error) {
					return journalctl.JournalEntry{
						Fields: event,
					}, nil
				}}
			ra := readerAdapter{
				r:         &mock,
				converter: journalfield.NewConverter(logp.L(), nil),
				canceler:  context.Background(),
			}

			evt, err := ra.Next()
			if err != nil {
				t.Fatalf("readerAdapter.Next must succeed, got an error: %s", err)
			}
			if len(evt.Content) == 0 {
				t.Fatal("event.Content must be populated")
			}
		})
	}
}

func decompress(t *testing.T, namegz string) string {
	t.Helper()

	ingz, err := os.Open(namegz)
	require.NoError(t, err)
	defer ingz.Close()

	out := filepath.Join(t.TempDir(), strings.TrimSuffix(filepath.Base(namegz), ".gz"))

	dst, err := os.Create(out)
	require.NoError(t, err)
	defer dst.Close()

	gr, err := gzip.NewReader(ingz)
	require.NoError(t, err)
	defer gr.Close()

	//nolint:gosec // this is used in tests
	_, err = io.Copy(dst, gr)
	require.NoError(t, err)

	return out
}
