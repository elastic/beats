package monitorstate

import (
	"fmt"
	"time"
)

const FlappingThreshold time.Duration = time.Second * 10

const (
	StatusUp       = "up"
	StatusDown     = "down"
	StatusFlapping = "flap"
)

func NewMonitorState(monitorId string, isUp bool) *MonitorState {
	startedAtMs := float64(time.Now().UnixMilli())
	ms := &MonitorState{
		Id:          fmt.Sprintf("%s-%x", monitorId, startedAtMs),
		MonitorId:   monitorId,
		StartedAtMs: startedAtMs,
		Checks:      1,
	}
	if isUp {
		ms.Status = StatusUp
	} else {
		ms.Status = StatusDown
	}
	return ms
}

type HistoricalStatus struct {
	TsMs   float64 `json:"ts_ms"`
	Status string  `json:"status"`
}

type MonitorState struct {
	MonitorId   string             `json:"monitorId"`
	Id          string             `json:"id"`
	StartedAtMs float64            `json:"started_at_ms"`
	Status      string             `json:"status"`
	Checks      int                `json:"checks"`
	Up          int                `json:"up"`
	Down        int                `json:"down"`
	FlapHistory []HistoricalStatus `json:"flap_history"`
	Ends        *MonitorState      `json:"ends"`
}

func (state *MonitorState) IsFlapping() bool {
	return len(state.FlapHistory) > 0
}

func (state *MonitorState) recordCheck(up bool) {
	state.Checks++
	if up {
		state.Up++
	} else {
		state.Down++
	}
}

func (state *MonitorState) isStateStillStable(currentStatus string) bool {
	return state.Status == currentStatus && state.IsFlapping()
}

// flapCompute returns true if we are still flapping, false if we no longer are.
func (state *MonitorState) flapCompute(currentStatus string) bool {
	state.FlapHistory = append(state.FlapHistory, HistoricalStatus{float64(time.Now().UnixMilli()), state.Status})
	state.Status = currentStatus

	// Figure out which values are old enough that we can discard them for our calculation
	cutOff := time.Now().Add(-FlappingThreshold).UnixMilli()
	discardIndex := -1
	for idx, hs := range state.FlapHistory {
		if int64(hs.TsMs) < cutOff {
			discardIndex = idx
		} else {
			break
		}
	}
	// Do the discarding
	if discardIndex != -1 {
		state.FlapHistory = state.FlapHistory[discardIndex+1:]
	}

	// Check to see if we are no longer flapping, and if so clear flap history
	for _, hs := range state.FlapHistory {
		if hs.Status != currentStatus {
			return false
		}
	}
	return true
}
