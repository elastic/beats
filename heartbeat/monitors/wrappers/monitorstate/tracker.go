package monitorstate

import (
	"math/rand"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-elasticsearch/v8"
)

var esClient *elasticsearch.Client

func SetEsClient(c *elasticsearch.Client) {
	esClient = c
}

// NewMonitorStateTracker tracks state across job runs. It takes an optional
// state loader, which will try to fetch the last known state for a never
// before seen monitor, which usually means using ES. If set to nil
// it will use ES if configured, otherwise it will only track state from
// memory.
func NewMonitorStateTracker(sl StateLoader) *MonitorStateTracker {
	mst := &MonitorStateTracker{
		states:      map[string]*MonitorState{},
		mtx:         sync.Mutex{},
		stateLoader: sl,
	}
	if esClient != nil {
		mst.stateLoader = MakeESLoader(esClient, "")
	}
	if mst.stateLoader == nil {
		mst.stateLoader = NilStateLoader
	}
	return mst
}

type MonitorStateTracker struct {
	states      map[string]*MonitorState
	mtx         sync.Mutex
	stateLoader StateLoader
}

// StateLoader has signature as loadLastESState, useful for test mocking, and maybe for a future impl
// other than ES if necessary
type StateLoader func(monitorId string) (*MonitorState, error)

func (mst *MonitorStateTracker) RecordStatus(monitorId string, newStatus StateStatus) (ms *MonitorState) {
	//note: the return values have no concurrency controls, they may be unsafely read unless
	//copied to the stack, copying the structs before  returning
	mst.mtx.Lock()
	defer mst.mtx.Unlock()

	state := mst.getCurrentState(monitorId)
	if state == nil {
		state = newMonitorState(monitorId, newStatus)
		mst.states[monitorId] = state
	} else {
		state.recordCheck(newStatus)
	}
	// return a copy since the state itself is a pointer that is frequently mutated
	return state.copy()
}

func (mst *MonitorStateTracker) getCurrentState(monitorId string) (state *MonitorState) {
	if state, ok := mst.states[monitorId]; ok {
		return state
	}

	tries := 3
	var loadedState *MonitorState
	var err error
	for i := 0; i < tries; i++ {
		loadedState, err = mst.stateLoader(monitorId)
		if err != nil {
			sleepFor := (time.Duration(i*i) * time.Second) + (time.Duration(rand.Intn(500)) * time.Millisecond)
			logp.L().Warnf("could not load last externally recorded state, will retry again in %d milliseconds: %w", sleepFor.Milliseconds(), err)
			time.Sleep(sleepFor)
		}
	}
	if err != nil {
		logp.L().Warn("could not load prior state from elasticsearch after %d attempts, will create new state for monitor %s", tries, monitorId)
	}

	// Return what we found, even if nil
	return loadedState
}

// NilStateLoader always returns nil, nil. It's the default when no ES conn is available
// or during testing
func NilStateLoader(_ string) (*MonitorState, error) {
	return nil, nil
}
