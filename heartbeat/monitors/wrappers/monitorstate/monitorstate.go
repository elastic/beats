package monitorstate

import (
	"sync"
	"time"
)

const FlappingThreshold time.Duration = time.Minute

const (
	StatusUp stateStatus = iota
	StatusDown
	StatusFlapping
)

func NewMonitorStateTracker() *MonitorStateTracker {
	return &MonitorStateTracker{
		states: map[string]*monitorState{},
		mtx:    sync.Mutex{},
	}
}

type MonitorStateTracker struct {
	states map[string]*monitorState
	mtx    sync.Mutex
}

func (mst *MonitorStateTracker) Compute(monitorId string, isUp bool) (curState *monitorState, stateEnded *monitorState, startedState *monitorState) {
	currentStatus := StatusDown
	if isUp {
		currentStatus = StatusUp
	}

	//note: the return values have no concurrency controls, they may be unsafely read unless
	//copied to the stack, copying the structs before  returning
	mst.mtx.Lock()
	defer mst.mtx.Unlock()

	if state, ok := mst.states[monitorId]; ok {
		if state.IsFlapping() {
			// Check to see if there's still an ongoing flap after recording
			// the new status
			if state.flapCompute(currentStatus) {
				state.recordCheck(isUp)
				return state, nil, nil
			} else {
				oldState := *state
				newState := *NewMonitorState(isUp)
				internalNewState := newState // Copy the struct since the returned value is read after the mutex
				mst.states[monitorId] = &internalNewState
				return &newState, &newState, &oldState
			}
		} else if state.status == currentStatus {
			// The state is stable, no changes needed
			state.Checks++
			return state, nil, nil
		} else if state.StartedAt.After(time.Now().Add(-FlappingThreshold)) {
			// The state changed too quickly, we're now flapping
			// TODO: is the above conditional right?
			state.flapCompute(currentStatus) // record the new state to the flap history
			state.Checks++
			return state, nil, nil
		}
	}

	// No previous state, so make a new one
	newState := *NewMonitorState(isUp)
	internalNewState := newState
	// Use a copy of the struct so that return values can safely be used concurrently
	mst.states[monitorId] = &internalNewState
	return &newState, &newState, nil
}

type stateStatus int8

func NewMonitorState(isUp bool) *monitorState {
	ms := &monitorState{
		StartedAt: time.Now(),
		Checks:    1,
	}
	if isUp {
		ms.status = StatusUp
	} else {
		ms.status = StatusDown
	}
	return ms
}

type historicalStatus struct {
	ts     time.Time
	status stateStatus
}

type monitorState struct {
	StartedAt   time.Time
	status      stateStatus
	Checks      int
	Up          int
	Down        int
	flapHistory []historicalStatus
}

func (state monitorState) Id() int64 {
	return state.StartedAt.UnixMilli()
}

func (state *monitorState) IsFlapping() bool {
	return len(state.flapHistory) > 0
}

func (state *monitorState) recordCheck(up bool) {
	state.Checks++
	if up {
		state.Up++
	} else {
		state.Down++
	}
}

func (state *monitorState) isStateStillStable(currentStatus stateStatus) bool {
	return state.status == currentStatus && state.IsFlapping()
}

// flapCompute returns true if we are still flapping, false if we no longer are.
func (state *monitorState) flapCompute(currentStatus stateStatus) bool {
	state.flapHistory = append(state.flapHistory, historicalStatus{time.Now(), state.status})
	state.status = currentStatus

	// Figure out which values are old enough that we can discard them for our calculation
	cutOff := time.Now().Add(-FlappingThreshold)
	discardIndex := -1
	for idx, hs := range state.flapHistory {
		if hs.ts.Before(cutOff) {
			discardIndex = idx
		} else {
			break
		}
	}
	// Do the discarding
	if discardIndex != -1 {
		state.flapHistory = state.flapHistory[discardIndex+1:]
	}

	// Check to see if we are no longer flapping, and if so clear flap history
	for _, hs := range state.flapHistory {
		if hs.status != currentStatus {
			return false
		}
	}
	return true
}

func (ms *monitorState) Status() string {
	if ms.Up > 0 && ms.Down > 0 {
		return "flapping"
	} else if ms.Up > 0 {
		return "up"
	}
	return "down"
}
