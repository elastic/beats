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
	"math/rand"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

// NewTracker tracks state across job runs. It takes an optional
// state loader, which will try to fetch the last known state for a never
// before seen monitor, which usually means using ES. If set to nil
// it will use ES if configured, otherwise it will only track state from
// memory.
func NewTracker(sl StateLoader) *Tracker {
	if sl == nil {
		sl = NilStateLoader
	}
	return &Tracker{
		states:      map[string]*State{},
		mtx:         sync.Mutex{},
		stateLoader: sl,
	}
}

type Tracker struct {
	states      map[string]*State
	mtx         sync.Mutex
	stateLoader StateLoader
}

// StateLoader has signature as loadLastESState, useful for test mocking, and maybe for a future impl
// other than ES if necessary
type StateLoader func(monitorID string) (*State, error)

func (t *Tracker) RecordStatus(monitorID string, newStatus StateStatus) (ms *State) {
	//note: the return values have no concurrency controls, they may be unsafely read unless
	//copied to the stack, copying the structs before  returning
	t.mtx.Lock()
	defer t.mtx.Unlock()

	state := t.getCurrentState(monitorID)
	if state == nil {
		state = newMonitorState(monitorID, newStatus, 0)
		t.states[monitorID] = state
	} else {
		state.recordCheck(monitorID, newStatus)
	}
	// return a copy since the state itself is a pointer that is frequently mutated
	return state.copy()
}

func (t *Tracker) getCurrentState(monitorID string) (state *State) {
	if state, ok := t.states[monitorID]; ok {
		return state
	}

	tries := 3
	var loadedState *State
	var err error
	for i := 0; i < tries; i++ {
		loadedState, err = t.stateLoader(monitorID)
		if err != nil {
			sleepFor := (time.Duration(i*i) * time.Second) + (time.Duration(rand.Intn(500)) * time.Millisecond)
			logp.L().Warnf("could not load last externally recorded state, will retry again in %d milliseconds: %w", sleepFor.Milliseconds(), err)
			time.Sleep(sleepFor)
		}
	}
	if err != nil {
		logp.L().Warn("could not load prior state from elasticsearch after %d attempts, will create new state for monitor %s", tries, monitorID)
	}

	if loadedState != nil {
		t.states[monitorID] = loadedState
	}

	// Return what we found, even if nil
	return loadedState
}

// NilStateLoader always returns nil, nil. It's the default when no ES conn is available
// or during testing
func NilStateLoader(_ string) (*State, error) {
	return nil, nil
}
