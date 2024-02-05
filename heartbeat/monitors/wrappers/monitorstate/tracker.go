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
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/elastic-agent-libs/logp"
)

// NewTracker tracks state across job runs. It takes an optional
// state loader, which will try to fetch the last known state for a never
// before seen monitor, which usually means using ES. If set to nil
// it will use ES if configured, otherwise it will only track state from
// memory.
func NewTracker(sl StateLoader, flappingEnabled bool) *Tracker {
	if sl == nil {
		sl = NilStateLoader
	}
	return &Tracker{
		states:          map[string]*State{},
		mtx:             sync.Mutex{},
		stateLoader:     sl,
		flappingEnabled: flappingEnabled,
	}
}

type Tracker struct {
	states          map[string]*State
	mtx             sync.Mutex
	stateLoader     StateLoader
	flappingEnabled bool
}

// StateLoader has signature as loadLastESState, useful for test mocking, and maybe for a future impl
// other than ES if necessary
type StateLoader func(stdfields.StdMonitorFields) (*State, error)

func (t *Tracker) RecordStatus(sf stdfields.StdMonitorFields, newStatus StateStatus, isFinalAttempt bool) (ms *State) {
	//note: the return values have no concurrency controls, they may be unsafely read unless
	//copied to the stack, copying the structs before  returning
	t.mtx.Lock()
	defer t.mtx.Unlock()

	state := t.GetCurrentState(sf)
	if state == nil {
		state = newMonitorState(sf, newStatus, 0, t.flappingEnabled)
		logp.L().Infof("initializing new state for monitor %s: %s", sf.ID, state.String())
		t.states[sf.ID] = state
	} else {
		state.recordCheck(sf, newStatus, isFinalAttempt)
	}
	// return a copy since the state itself is a pointer that is frequently mutated
	return state.copy()
}

func (t *Tracker) GetCurrentStatus(sf stdfields.StdMonitorFields) StateStatus {
	s := t.GetCurrentState(sf)
	if s == nil {
		return StatusEmpty
	}
	return s.Status
}

func (t *Tracker) GetCurrentState(sf stdfields.StdMonitorFields) (state *State) {
	if state, ok := t.states[sf.ID]; ok {
		return state
	}

	tries := 3
	var loadedState *State
	var err error
	for i := 0; i < tries; i++ {
		loadedState, err = t.stateLoader(sf)
		if err == nil {
			if loadedState != nil {
				logp.L().Infof("loaded previous state for monitor %s: %s", sf.ID, loadedState.String())
			}
			break
		}
		var loaderError LoaderError
		if errors.As(err, &loaderError) && !loaderError.Retry {
			break
		}

		sleepFor := (time.Duration(i*i) * time.Second) + (time.Duration(rand.Intn(500)) * time.Millisecond)
		logp.L().Warnf("could not load last externally recorded state, will retry again in %d milliseconds: %w", sleepFor.Milliseconds(), err)
		time.Sleep(sleepFor)
	}
	if err != nil {
		logp.L().Warnf("could not load prior state from elasticsearch after %d attempts, will create new state for monitor: %s", tries, sf.ID)
	}

	if loadedState != nil {
		t.states[sf.ID] = loadedState
	}

	// Return what we found, even if nil
	return loadedState
}

// NilStateLoader always returns nil, nil. It's the default when no ES conn is available
// or during testing
func NilStateLoader(_ stdfields.StdMonitorFields) (*State, error) {
	return nil, nil
}

func AtomicStateLoader(inner StateLoader) (sl StateLoader, replace func(StateLoader)) {
	mtx := &sync.Mutex{}
	return func(currentSL stdfields.StdMonitorFields) (*State, error) {
			mtx.Lock()
			defer mtx.Unlock()

			return inner(currentSL)
		}, func(sl StateLoader) {
			mtx.Lock()
			defer mtx.Unlock()
			inner = sl
			logp.L().Info("Updated atomic state loader")
		}
}

func DeferredStateLoader(inner StateLoader, timeout time.Duration) (sl StateLoader, replace func(StateLoader)) {
	stateLoader, replaceStateLoader := AtomicStateLoader(inner)

	wg := sync.WaitGroup{}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	wg.Add(1)
	go func() {
		defer cancel()
		defer wg.Done()

		<-ctx.Done()

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			logp.L().Warn("Timeout trying to defer state loader")
		}
	}()

	return func(currentSL stdfields.StdMonitorFields) (*State, error) {
			wg.Wait()

			return stateLoader(currentSL)
		}, func(sl StateLoader) {
			defer cancel()

			replaceStateLoader(sl)
		}
}
