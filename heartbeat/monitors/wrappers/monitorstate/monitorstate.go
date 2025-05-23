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

package monitorstate

import (
	"fmt"
	"regexp"
	"time"

	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
)

// FlappingThreshold defines how many consecutive checks with the same status
// must occur for us to end a flapping state. FlappingThreshold-1 is the number
// of consecutive checks that is insufficient to start a new state, but rather to
// keep the current state and turn it into a flapping state.
const FlappingThreshold = 3

type StateStatus string

const (
	StatusUp       StateStatus = "up"
	StatusDown     StateStatus = "down"
	StatusFlapping StateStatus = "flap"
	// Nil, essentially
	StatusEmpty StateStatus = ""
)

func newMonitorState(sf stdfields.StdMonitorFields, status StateStatus, ctr int, flappingEnabled bool) *State {
	now := time.Now()
	ms := &State{
		// ID is unique and sortable by time for easier aggregations
		// Note that we add an incrementing counter to help with the fact that
		// millisecond res isn't quite enough for uniqueness (esp. in tests)
		ID:              LoaderDBKey(sf, now, ctr),
		StartedAt:       now,
		DurationMs:      0,
		Status:          status,
		flappingEnabled: flappingEnabled,
		ctr:             ctr + 1,
	}
	ms.recordCheck(sf, status, false)

	return ms
}

type State struct {
	ID string `json:"id"`
	// StartedAt is the start time of the state, should be the same for a given state ID
	StartedAt  time.Time   `json:"started_at"`
	DurationMs int64       `json:"duration_ms,string"`
	Status     StateStatus `json:"status"`
	Checks     int         `json:"checks"`
	Up         int         `json:"up"`
	Down       int         `json:"down"`
	// FlapHistory retains enough info so we can resume our flap
	// computation if loading from ES or another source
	FlapHistory []StateStatus `json:"flap_history"`
	// Ends is a pointer to the prior state if this is the start of a new state
	Ends            *State `json:"ends"`
	flappingEnabled bool
	ctr             int
}

func (s *State) String() string {
	if s == nil {
		return "<monitorstate:nil>"
	}
	return fmt.Sprintf("<monitorstate:id=%s,started=%s,up=%d,down=%d>", s.ID, s.StartedAt, s.Up, s.Down)
}

func (s *State) incrementCounters(status StateStatus) {
	s.DurationMs = time.Since(s.StartedAt).Milliseconds()
	s.Checks++
	if status == StatusUp {
		s.Up++
	} else {
		s.Down++
	}
}

// truncate flap history to be at most as many items as the threshold indicates, minus one
func (s *State) truncateFlapHistory() {
	endIdx := len(s.FlapHistory)
	if endIdx <= 0 {
		return // flap history is empty
	}
	// truncate to one less than the threshold since our later calculations
	// an item that would stabilize the history at the threshold would start a new state
	startIdx := endIdx - (FlappingThreshold - 1)
	if startIdx <= 0 {
		return
	}
	s.FlapHistory = s.FlapHistory[startIdx:endIdx]
}

// recordCheck updates the current state pointer to what the new state should be.
// If the current state is continued it just updates counters and other record keeping,
// if the state ends it actually swaps out the full value the state points to
// and sets state.Ends.
func (s *State) recordCheck(sf stdfields.StdMonitorFields, newStatus StateStatus, isFinalAttempt bool) {
	if s.Status == StatusFlapping {
		s.truncateFlapHistory()

		// Check if all statuses in flap history are identical, including the new status
		hasStabilized := true
		for _, histStatus := range s.FlapHistory {
			if newStatus != histStatus {
				hasStabilized = false
				break
			}
		}

		if !hasStabilized || !isFinalAttempt { // continue flapping
			// Use the new flap history as part of the state
			s.FlapHistory = append(s.FlapHistory, newStatus)
			s.incrementCounters(newStatus)
		} else { // flap has ended
			s.transitionTo(sf, newStatus)
		}
		// stable state, status has not changed
		// or this will be retried, so no state change yet
	} else if s.Status == newStatus || !isFinalAttempt {
		// The state is stable, no changes needed
		s.incrementCounters(newStatus)
	} else if s.Checks < FlappingThreshold && s.flappingEnabled {
		// The state changed too quickly, we're now flapping
		s.incrementCounters(newStatus)
		s.Status = StatusFlapping
		s.FlapHistory = append(s.FlapHistory, newStatus)
	} else {
		s.transitionTo(sf, newStatus)
	}

	// Ensure that the ends field is set to nil
	// It's only needed on transitions
	if s.Checks > 1 {
		s.Ends = nil
	}
}

func (s *State) transitionTo(sf stdfields.StdMonitorFields, newStatus StateStatus) {
	// state has changed, but we aren't flapping (yet), since we've been stable past the
	// flapping threshold
	oldState := *s
	*s = *newMonitorState(sf, newStatus, s.ctr, s.flappingEnabled)
	// We don't need to retain extra data when transitioning
	oldState.FlapHistory = nil
	// W edon't want an infinite linked list!
	oldState.Ends = nil
	s.Ends = &oldState
}

// copy returns a threadsafe copy since the instance used in the tracker is frequently mutated
func (s *State) copy() *State {
	copied := *s
	copied.FlapHistory = make([]StateStatus, len(s.FlapHistory))
	copy(copied.FlapHistory, s.FlapHistory)
	return &copied
}

var normalizeRunFromIDRegexp = regexp.MustCompile("[^A-Za-z0-9_-]")

func LoaderDBKey(sf stdfields.StdMonitorFields, at time.Time, ctr int) string {
	rfid := "default"
	if sf.RunFrom != nil {
		rfid = normalizeRunFromIDRegexp.ReplaceAllString(sf.RunFrom.ID, "_")

	}
	key := fmt.Sprintf("%s-%x-%x", rfid, at.UnixMilli(), ctr)
	return key
}
