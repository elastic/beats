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

func newMonitorState(monitorId string, status StateStatus) *MonitorState {
	startedAtMs := float64(time.Now().UnixMilli())
	ms := &MonitorState{
		Id:          fmt.Sprintf("%s-%x", monitorId, startedAtMs),
		MonitorId:   monitorId,
		StartedAtMs: startedAtMs,
		Status:      status,
	}
	ms.recordCheck(status)

	return ms
}

type HistoricalStatus struct {
	TsMs   float64     `json:"ts_ms"`
	Status StateStatus `json:"status"`
}

type MonitorState struct {
	MonitorId   string        `json:"monitorId"`
	Id          string        `json:"id"`
	StartedAtMs float64       `json:"started_at_ms"`
	Status      StateStatus   `json:"status"`
	Checks      int           `json:"checks"`
	Up          int           `json:"up"`
	Down        int           `json:"down"`
	FlapHistory []StateStatus `json:"flap_history"`
	Ends        *MonitorState `json:"ends"`
}

func (ms *MonitorState) incrementCounters(status StateStatus) {
	ms.Checks++
	if status == StatusUp {
		ms.Up++
	} else {
		ms.Down++
	}
}

// truncate flap history to be at most as many items as the threshold indicates, minus one
func (ms *MonitorState) truncateFlapHistory() {
	endIdx := len(ms.FlapHistory)
	if endIdx < 0 {
		return // flap history is empty
	}
	// truncate to one less than the threshold since our later calculations
	// an item that would stabilize the history at the threshold would start a new state
	startIdx := endIdx - (FlappingThreshold - 1)
	if startIdx <= 0 {
		return
	}
	ms.FlapHistory = ms.FlapHistory[startIdx:endIdx]
}

// recordCheck updates the current state pointer to what the new state should be.
// If the current state is continued it just updates counters and other record keeping,
// if the state ends it actually swaps out the full value the state points to
// and sets state.Ends.
func (ms *MonitorState) recordCheck(newStatus StateStatus) {
	if ms.Status == StatusFlapping {
		ms.truncateFlapHistory()

		// Check if all statuses in flap history are identical, including the new status
		hasStabilized := true
		for _, histStatus := range ms.FlapHistory {
			if newStatus != histStatus {
				hasStabilized = false
				break
			}
		}

		if !hasStabilized { // continue flapping
			// Use the new flap history as part of the state
			ms.FlapHistory = append(ms.FlapHistory, newStatus)
			ms.incrementCounters(newStatus)
		} else { // flap has ended
			oldState := *ms
			*ms = *newMonitorState(ms.MonitorId, newStatus)
			ms.Ends = &oldState
		}
	} else if ms.Status == newStatus { // stable state, status has not changed
		// The state is stable, no changes needed
		ms.incrementCounters(newStatus)
	} else if ms.Checks < FlappingThreshold {
		// The state changed too quickly, we're now flapping
		ms.incrementCounters(newStatus)
		ms.Status = StatusFlapping
		ms.FlapHistory = append(ms.FlapHistory, newStatus)
	} else {
		// state has changed, but we aren't flapping (yet), since we've been stable past the
		// flapping threshold
		oldState := *ms
		*ms = *newMonitorState(ms.MonitorId, newStatus)
		ms.Ends = &oldState
	}
}

// copy returns a threadsafe copy since the instance used in the tracker is frequently mutated
func (ms *MonitorState) copy() *MonitorState {
	copied := *ms
	copied.FlapHistory = make([]StateStatus, len(ms.FlapHistory))
	copy(copied.FlapHistory, ms.FlapHistory)
	return &copied
}
