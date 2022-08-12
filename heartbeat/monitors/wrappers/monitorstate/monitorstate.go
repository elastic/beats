package monitorstate

import (
	"fmt"
	"time"
)

// FlappingThreshold defines how many consecutive checks with the same status
// must occur for us to end a flapping state. FlappingThreshold-1 is the number
// of consecutive checks that is insufficient to start a new state, but rather to
// keep the current state and turn it into a flapping state.
const FlappingThreshold = 7

type StateStatus string

const (
	StatusUp       StateStatus = "up"
	StatusDown     StateStatus = "down"
	StatusFlapping StateStatus = "flap"
)

func newMonitorState(monitorId string, status StateStatus) *State {
	now := time.Now()
	ms := &State{
		ID:         fmt.Sprintf("%s-%x", monitorId, now.UnixMilli()),
		StartedAt:  now,
		DurationMs: 0,
		Status:     status,
	}
	ms.recordCheck(monitorId, status)

	return ms
}

type State struct {
	ID string `json:"id"`
	// StartedAt is the start time of the state, should be the same for a given state ID
	StartedAt  time.Time   `json:"started_at"`
	DurationMs int64       `json:"duration_ms"`
	Status     StateStatus `json:"status"`
	Checks     int         `json:"checks"`
	Up         int         `json:"up"`
	Down       int         `json:"down"`
	// FlapHistory retains enough info so we can resume our flap
	// computation if loading from ES or another source
	FlapHistory []StateStatus `json:"flap_history"`
	// Ends is a pointer to the prior state if this is the start of a new state
	Ends *State `json:"ends"`
}

func (s *State) incrementCounters(status StateStatus) {
	s.DurationMs = time.Until(s.StartedAt).Milliseconds()
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
	if endIdx < 0 {
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
func (s *State) recordCheck(monitorId string, newStatus StateStatus) {
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

		if !hasStabilized { // continue flapping
			// Use the new flap history as part of the state
			s.FlapHistory = append(s.FlapHistory, newStatus)
			s.incrementCounters(newStatus)
		} else { // flap has ended
			oldState := *s
			// Remove the flap history, or we'll create a linked list
			// of our full history!
			oldState.FlapHistory = nil
			*s = *newMonitorState(monitorId, newStatus)
			s.Ends = &oldState
		}
	} else if s.Status == newStatus { // stable state, status has not changed
		// The state is stable, no changes needed
		s.incrementCounters(newStatus)
	} else if s.Checks < FlappingThreshold {
		// The state changed too quickly, we're now flapping
		s.incrementCounters(newStatus)
		s.Status = StatusFlapping
		s.FlapHistory = append(s.FlapHistory, newStatus)
	} else {
		// state has changed, but we aren't flapping (yet), since we've been stable past the
		// flapping threshold
		oldState := *s
		*s = *newMonitorState(monitorId, newStatus)
		s.Ends = &oldState
	}
}

// copy returns a threadsafe copy since the instance used in the tracker is frequently mutated
func (s *State) copy() *State {
	copied := *s
	copied.FlapHistory = make([]StateStatus, len(s.FlapHistory))
	copy(copied.FlapHistory, s.FlapHistory)
	return &copied
}
