package monitorstate

import (
	"fmt"
	"time"
)

const FlappingThreshold = 3

type StateStatus string

const (
	StatusUp       StateStatus = "up"
	StatusDown     StateStatus = "down"
	StatusFlapping StateStatus = "flap"
)

func newMonitorState(monitorId string, status StateStatus) *State {
	nowMillis := time.Now().UnixMilli()
	ms := &State{
		Id:          fmt.Sprintf("%s-%x", monitorId, nowMillis),
		MonitorId:   monitorId,
		StartedAtMs: float64(nowMillis),
		Status:      status,
	}
	ms.recordCheck(status)

	return ms
}

type HistoricalStatus struct {
	TsMs   float64     `json:"ts_ms"`
	Status StateStatus `json:"status"`
}

type State struct {
	MonitorId   string        `json:"monitorId"`
	Id          string        `json:"id"`
	StartedAtMs float64       `json:"started_at_ms"`
	Status      StateStatus   `json:"status"`
	Checks      int           `json:"checks"`
	Up          int           `json:"up"`
	Down        int           `json:"down"`
	FlapHistory []StateStatus `json:"flap_history"`
	Ends        *State        `json:"ends"`
}

func (s *State) incrementCounters(status StateStatus) {
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
func (s *State) recordCheck(newStatus StateStatus) {
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
			*s = *newMonitorState(s.MonitorId, newStatus)
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
		*s = *newMonitorState(s.MonitorId, newStatus)
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
