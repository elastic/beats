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

//go:build linux && cgo && withjournald

package journald

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/require"
)

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

func TestCompare(t *testing.T) {
	expectedEvents := 1 //8
	env := newInputTestingEnvironment(t)

	inp := env.mustCreateInput(mapstr.M{
		"paths": []string{path.Join("testdata", "input-multiline-parser.journal")},
		// "include_matches.match": []string{"_SYSTEMD_USER_UNIT=log-service.service"},
		"include_matches.match": []string{"MESSAGE=1st line"},
	})

	ctx, cancelInput := context.WithCancel(context.Background())
	defer cancelInput()

	env.startInput(ctx, inp)
	env.waitUntilEventCount(expectedEvents)
	t.Log("Legacy journald input ok, starting journalctl")

	env2 := newInputTestingEnvironment(t)
	inp2 := env2.mustCreateInput(mapstr.M{
		"paths": []string{path.Join("testdata", "input-multiline-parser.journal")},
		// "include_matches.match": []string{"_SYSTEMD_USER_UNIT=log-service.service"},
		"include_matches.match": []string{"MESSAGE=1st line"},
		"journalctl":            true,
	})

	ctx2, cancelInput2 := context.WithCancel(context.Background())
	defer cancelInput2()

	env2.startInput(ctx2, inp2)
	env2.waitUntilEventCount(expectedEvents)
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
	gold, got := goldenEvents[3], events[3]
	fmt.Println(gold.Timestamp, got.Timestamp, gold.Meta, got.Meta)
	require.EqualValues(t, goldenEvents[3], events[3], "events do not match reference")
}
