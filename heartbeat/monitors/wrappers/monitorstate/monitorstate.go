package monitorstate

import (
	"fmt"
	"time"
)

const FlappingThreshold = 3

type MonitorStatus string

const (
	StatusUp       MonitorStatus = "up"
	StatusDown     MonitorStatus = "down"
	StatusFlapping MonitorStatus = "flap"
)

func newMonitorState(monitorId string, status MonitorStatus) *MonitorState {
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
	TsMs   float64       `json:"ts_ms"`
	Status MonitorStatus `json:"status"`
}

type MonitorState struct {
	MonitorId   string          `json:"monitorId"`
	Id          string          `json:"id"`
	StartedAtMs float64         `json:"started_at_ms"`
	Status      MonitorStatus   `json:"status"`
	Checks      int             `json:"checks"`
	Up          int             `json:"up"`
	Down        int             `json:"down"`
	FlapHistory []MonitorStatus `json:"flap_history"`
	Ends        *MonitorState   `json:"ends"`
}

func (state *MonitorState) isFlapping() bool {
	return len(state.FlapHistory) > 0
}

func (state *MonitorState) incrementCounters(status MonitorStatus) {
	state.Checks++
	if status == StatusUp {
		state.Up++
	} else {
		state.Down++
	}
}

func (state *MonitorState) truncateFlapHistoryToThreshold() {
	endIdx := len(state.FlapHistory) - 1
	startIdx := endIdx - FlappingThreshold
	if startIdx < 0 {
		startIdx = 0
	}
	state.FlapHistory = state.FlapHistory[startIdx:endIdx]
}

// recordCheck records a new check to the stat counters only, it does not do any flap computation
func (state *MonitorState) recordCheck(newStatus MonitorStatus) {
	if state.isFlapping() {
		newFlapHistory := append(state.FlapHistory, newStatus)
		var lastStatus MonitorStatus
		isStable := true
		for _, histStatus := range newFlapHistory {
			if lastStatus != histStatus {
				isStable = false
				break
			}
			lastStatus = histStatus
		}

		if !isStable { // continue flapping
			// Use the new flap history as part of the state
			state.FlapHistory = newFlapHistory
			state.incrementCounters(newStatus)
		} else { // flap has ended
			oldState := *state
			state = newMonitorState(state.MonitorId, newStatus)
			state.Ends = &oldState
		}
	} else if state.Status == newStatus { // stable state, status has not changed
		// The state is stable, no changes needed
		state.incrementCounters(newStatus)
	} else if state.Checks < FlappingThreshold {
		// The state changed too quickly, we're now flapping
		state.incrementCounters(newStatus)
		state.FlapHistory = append(state.FlapHistory, newStatus)
	} else {
		// state has changed, but we aren't flapping (yet), since we've been stable past the
		// flapping threshold
		oldState := *state
		state = newMonitorState(state.MonitorId, newStatus)
		state.Ends = &oldState
	}
}

// copy returns a threadsafe copy since the instance used in the tracker is frequently mutated
func (state *MonitorState) copy() *MonitorState {
	copied := *state
	copied.FlapHistory = make([]MonitorStatus, len(state.FlapHistory))
	copy(copied.FlapHistory, state.FlapHistory)
	return &copied
}
