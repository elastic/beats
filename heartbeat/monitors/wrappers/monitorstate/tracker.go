package monitorstate

import (
	"math/rand"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

// NewMonitorStateTracker tracks state across job runs. It takes an optional
// state loader, which will try to fetch the last known state for a never
// before seen monitor, which usually means using ES. If set to nil
// it will use ES if configured, otherwise it will only track state from
// memory.
func NewMonitorStateTracker(sl StateLoader) *Tracker {
	t := &Tracker{
		states:      map[string]*State{},
		mtx:         sync.Mutex{},
		stateLoader: sl,
	}
	if t.stateLoader == nil {
		t.stateLoader = NilStateLoader
	}

	return t
}

type Tracker struct {
	states      map[string]*State
	mtx         sync.Mutex
	stateLoader StateLoader
}

// StateLoader has signature as loadLastESState, useful for test mocking, and maybe for a future impl
// other than ES if necessary
type StateLoader func(monitorId string) (*State, error)

func (t *Tracker) RecordStatus(monitorId string, newStatus StateStatus) (ms *State) {
	//note: the return values have no concurrency controls, they may be unsafely read unless
	//copied to the stack, copying the structs before  returning
	t.mtx.Lock()
	defer t.mtx.Unlock()

	state := t.getCurrentState(monitorId)
	if state == nil {
		state = newMonitorState(monitorId, newStatus)
		t.states[monitorId] = state
	} else {
		state.recordCheck(newStatus)
	}
	// return a copy since the state itself is a pointer that is frequently mutated
	return state.copy()
}

func (t *Tracker) getCurrentState(monitorId string) (state *State) {
	if state, ok := t.states[monitorId]; ok {
		return state
	}

	tries := 3
	var loadedState *State
	var err error
	for i := 0; i < tries; i++ {
		loadedState, err = t.stateLoader(monitorId)
		if err != nil {
			sleepFor := (time.Duration(i*i) * time.Second) + (time.Duration(rand.Intn(500)) * time.Millisecond)
			logp.L().Warnf("could not load last externally recorded state, will retry again in %d milliseconds: %w", sleepFor.Milliseconds(), err)
			time.Sleep(sleepFor)
		}
	}
	if err != nil {
		logp.L().Warn("could not load prior state from elasticsearch after %d attempts, will create new state for monitor %s", tries, monitorId)
	}

	if loadedState != nil {
		t.states[monitorId] = loadedState
	}

	// Return what we found, even if nil
	return loadedState
}

// NilStateLoader always returns nil, nil. It's the default when no ES conn is available
// or during testing
func NilStateLoader(_ string) (*State, error) {
	return nil, nil
}
