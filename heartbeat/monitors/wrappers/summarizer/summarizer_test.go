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

package summarizer

import (
	"fmt"
	"testing"
	"time"

	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/require"
)

func TestSummarizer(t *testing.T) {
	t.Parallel()
	charToStatus := func(c uint8) monitorstate.StateStatus {
		if c == 'u' {
			return monitorstate.StatusUp
		} else {
			return monitorstate.StatusDown
		}
	}

	tests := []struct {
		name           string
		maxAttempts    int
		statusSequence string
		expectedStates string
	}{
		{
			"start down, transition to up",
			2,
			"du",
			"du",
		},
		{
			"start up, stay up",
			2,
			"uuuuuuuu",
			"uuuuuuuu",
		},
		{
			"start down, stay down",
			2,
			"dddddddd",
			"dddddddd",
		},
		{
			"start up - go down with one retry - thenrecover",
			2,
			"udddduuu",
			"uuddduuu",
		},
		{
			"start up, transient down, recover",
			2,
			"uuuduuuu",
			"uuuuuuuu",
		},
		{
			"start up, multiple transient down, recover",
			2,
			"uuudududu",
			"uuuuuuuuu",
		},
		{
			"no retries, single down",
			1,
			"uuuduuuu",
			"uuuduuuu",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dummyErr := fmt.Errorf("dummyerr")

			// The job runs through each char in the status sequence and
			// returns an error if it's set to 'd'
			pos := 0
			job := func(event *beat.Event) (j []jobs.Job, retErr error) {
				status := charToStatus(tt.statusSequence[pos])
				if status == monitorstate.StatusDown {
					retErr = dummyErr
				}
				event.Fields = mapstr.M{
					"monitor": mapstr.M{
						"id":     "test",
						"status": string(status),
					},
				}

				pos++
				return nil, retErr
			}

			tracker := monitorstate.NewTracker(monitorstate.NilStateLoader, false)
			sf := stdfields.StdMonitorFields{ID: "testmon", Name: "testmon", MaxAttempts: uint16(tt.maxAttempts)}

			rcvdStatuses := ""
			rcvdStates := ""
			i := 0
			for {
				s := NewSummarizer(job, sf, tracker)
				// Shorten retry delay to make tests run faster
				s.retryDelay = 2 * time.Millisecond
				wrapped := s.Wrap(job)
				events, _ := jobs.ExecJobAndConts(t, wrapped)
				for _, event := range events {
					eventStatus, _ := event.GetValue("monitor.status")
					eventStatusStr := eventStatus.(string)
					rcvdStatuses += eventStatusStr[:1]
					state, _ := event.GetValue("state")
					if state != nil {
						rcvdStates += string(state.(*monitorstate.State).Status)[:1]
					} else {
						rcvdStates += "_"
					}
				}
				i += len(events)
				if i >= len(tt.statusSequence) {
					break
				}
			}
			require.Equal(t, tt.statusSequence, rcvdStatuses)
			require.Equal(t, tt.expectedStates, rcvdStates)
		})
	}
}
