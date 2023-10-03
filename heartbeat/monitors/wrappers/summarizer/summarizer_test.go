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

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/summarizer/jobsummary"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
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

	testURL := "https://example.net"
	// these tests use strings to describe sequences of events
	tests := []struct {
		name        string
		maxAttempts int
		// The sequence of up down states the monitor should emit
		// Equivalent to monitor.status
		statusSequence string
		// The expected states on each event
		expectedStates string
		// the attempt number of the given event
		expectedAttempts  string
		expectedSummaries int
		url               string
	}{
		{
			"start down, transition to up",
			2,
			"du",
			"du",
			"11",
			2,
			testURL,
		},
		{
			"start up, stay up",
			2,
			"uuuuuuuu",
			"uuuuuuuu",
			"11111111",
			8,
			testURL,
		},
		{
			"start down, stay down",
			2,
			"dddddddd",
			"dddddddd",
			"11111111",
			8,
			testURL,
		},
		{
			"start up - go down with one retry - thenrecover",
			2,
			"udddduuu",
			"uuddduuu",
			"11211111",
			8,
			testURL,
		},
		{
			"start up, transient down, recover",
			2,
			"uuuduuuu",
			"uuuuuuuu",
			"11112111",
			8,
			testURL,
		},
		{
			"start up, multiple transient down, recover",
			2,
			"uuudududu",
			"uuuuuuuuu",
			"111121212",
			9,
			testURL,
		},
		{
			"no retries, single down",
			1,
			"uuuduuuu",
			"uuuduuuu",
			"11111111",
			8,
			testURL,
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
			sf := stdfields.StdMonitorFields{ID: "testmon", Name: "testmon", Type: "http", MaxAttempts: uint16(tt.maxAttempts)}

			rcvdStatuses := ""
			rcvdStates := ""
			rcvdAttempts := ""
			rcvdEvents := []*beat.Event{}
			rcvdSummaries := []*jobsummary.JobSummary{}
			i := 0
			var lastSummary *jobsummary.JobSummary
			for {
				s := NewSummarizer(job, sf, tracker)
				// Shorten retry delay to make tests run faster
				s.retryDelay = 2 * time.Millisecond
				wrapped := s.Wrap(job)
				events, _ := jobs.ExecJobAndConts(t, wrapped)
				for _, event := range events {
					rcvdEvents = append(rcvdEvents, event)
					eventStatus, _ := event.GetValue("monitor.status")
					eventStatusStr := eventStatus.(string)
					rcvdStatuses += eventStatusStr[:1]
					state, _ := event.GetValue("state")
					if state != nil {
						rcvdStates += string(state.(*monitorstate.State).Status)[:1]
					} else {
						rcvdStates += "_"
					}
					summaryIface, _ := event.GetValue("summary")
					summary := summaryIface.(*jobsummary.JobSummary)
					duration, _ := event.GetValue("monitor.duration.us")

					// Ensure that only summaries have a duration
					if summary != nil {
						rcvdSummaries = append(rcvdSummaries, summary)
						require.GreaterOrEqual(t, duration, int64(0))
						// down summaries should always have errors
						if eventStatusStr == "down" {
							require.NotNil(t, event.Fields["error"])
						} else {
							require.Nil(t, event.Fields["error"])
						}
					} else {
						require.Nil(t, duration)
					}

					if summary == nil {
						// note missing summaries
						rcvdAttempts += "!"
					} else if lastSummary != nil {
						if summary.Attempt > 1 {
							require.Equal(t, lastSummary.RetryGroup, summary.RetryGroup)
						} else {
							require.NotEqual(t, lastSummary.RetryGroup, summary.RetryGroup)
						}
					}

					rcvdAttempts += fmt.Sprintf("%d", summary.Attempt)
					lastSummary = summary
				}
				i += len(events)
				if i >= len(tt.statusSequence) {
					break
				}
			}
			require.Equal(t, tt.statusSequence, rcvdStatuses)
			require.Equal(t, tt.expectedStates, rcvdStates)
			require.Equal(t, tt.expectedAttempts, rcvdAttempts)
			require.Len(t, rcvdEvents, len(tt.statusSequence))
			require.Len(t, rcvdSummaries, tt.expectedSummaries)
		})
	}
}
