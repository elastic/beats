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

package registrar

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/input/file"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
)

type Registrar struct {
	log *logp.Logger

	// registrar event input and output
	Channel              chan []file.State
	out                  successLogger
	bufferedStateUpdates int

	// shutdown handling
	done chan struct{}
	wg   sync.WaitGroup

	// state storage
	states       *file.States      // Map with all file paths inside and the corresponding state
	store        *statestore.Store // Store keeps states in memory and on disk
	flushTimeout time.Duration

	gcEnabled, gcRequired bool
}

type successLogger interface {
	Published(n int) bool
}

type StateStore interface {
	Access() (*statestore.Store, error)
}

var (
	statesUpdate    = monitoring.NewInt(nil, "registrar.states.update")
	statesCleanup   = monitoring.NewInt(nil, "registrar.states.cleanup")
	statesCurrent   = monitoring.NewInt(nil, "registrar.states.current")
	registryWrites  = monitoring.NewInt(nil, "registrar.writes.total")
	registryFails   = monitoring.NewInt(nil, "registrar.writes.fail")
	registrySuccess = monitoring.NewInt(nil, "registrar.writes.success")
)

const fileStatePrefix = "filebeat::logs::"

// New creates a new Registrar instance, updating the registry file on
// `file.State` updates. New fails if the file can not be opened or created.
func New(stateStore StateStore, out successLogger, flushTimeout time.Duration) (*Registrar, error) {
	store, err := stateStore.Access()
	if err != nil {
		return nil, err
	}

	r := &Registrar{
		log:          logp.NewLogger("registrar"),
		Channel:      make(chan []file.State, 1),
		out:          out,
		done:         make(chan struct{}),
		wg:           sync.WaitGroup{},
		states:       file.NewStates(),
		store:        store,
		flushTimeout: flushTimeout,
	}
	return r, nil
}

// GetStates return the registrar states
func (r *Registrar) GetStates() []file.State {
	return r.states.GetStates()
}

// loadStates fetches the previous reading state from the configure RegistryFile file
// The default file is `registry` in the data path.
func (r *Registrar) loadStates() error {
	states, err := readStatesFrom(r.store)
	if err != nil {
		return errors.Wrap(err, "can not load filebeat registry state")
	}

	r.states.SetStates(states)
	r.log.Infof("States Loaded from registrar: %+v", len(states))

	return nil
}

func (r *Registrar) Start() error {
	// Load the previous log file locations now, for use in input
	err := r.loadStates()
	if err != nil {
		return fmt.Errorf("error loading state: %v", err)
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.Run()
	}()

	return nil
}

// Stop stops the registry. It waits until Run function finished.
func (r *Registrar) Stop() {
	r.log.Info("Stopping Registrar")
	defer r.log.Info("Registrar stopped")

	close(r.done)
	r.wg.Wait()
}

func (r *Registrar) Run() {
	r.log.Debug("Starting Registrar")
	defer r.log.Debug("Stopping Registrar")

	defer r.store.Close()

	defer func() {
		writeStates(r.store, r.states.GetStates())
	}()

	var (
		timer  *time.Timer
		flushC <-chan time.Time

		directIn  chan []file.State
		collectIn chan []file.State
	)

	if r.flushTimeout <= 0 {
		directIn = r.Channel
	} else {
		collectIn = r.Channel
	}

	for {
		select {
		case <-r.done:
			r.log.Info("Ending Registrar")
			return

		case states := <-directIn:
			// no flush timeout configured. Directly update registry
			r.onEvents(states)
			r.commitStateUpdates()

		case states := <-collectIn:
			// flush timeout configured. Only update internal state and track pending
			// updates to be written to registry.
			r.onEvents(states)
			r.gcStates()
			if flushC == nil && len(states) > 0 {
				timer = time.NewTimer(r.flushTimeout)
				flushC = timer.C
			}

		case <-flushC:
			timer.Stop()
			r.commitStateUpdates()

			flushC = nil
			timer = nil
		}
	}
}

func (r *Registrar) commitStateUpdates() {
	// First clean up states
	r.gcStates()
	states := r.states.GetStates()
	statesCurrent.Set(int64(len(states)))

	registryWrites.Inc()
	if err := writeStates(r.store, states); err != nil {
		r.log.Errorf("Error writing registrar state to statestore: %v", err)
		registryFails.Inc()
	}
	r.log.Debugf("Registry file updated. %d active states.", len(states))
	registrySuccess.Inc()

	if r.out != nil {
		r.out.Published(r.bufferedStateUpdates)
	}
	r.bufferedStateUpdates = 0
}

// onEvents processes events received from the publisher pipeline
func (r *Registrar) onEvents(states []file.State) {
	r.processEventStates(states)
	r.bufferedStateUpdates += len(states)

	// check if we need to enable state cleanup
	if !r.gcEnabled {
		for i := range states {
			if states[i].TTL >= 0 || states[i].Finished {
				r.gcEnabled = true
				break
			}
		}
	}

	r.log.Debugf("Registrar state updates processed. Count: %v", len(states))

	// new set of events received -> mark state registry ready for next
	// cleanup phase in case gc'able events are stored in the registry.
	r.gcRequired = r.gcEnabled
}

// gcStates runs a registry Cleanup. The method check if more event in the
// registry can be gc'ed in the future. If no potential removable state is found,
// the gcEnabled flag is set to false, indicating the current registrar state being
// stable. New registry update events can re-enable state gc'ing.
func (r *Registrar) gcStates() {
	if !r.gcRequired {
		return
	}

	beforeCount := r.states.Count()
	cleanedStates, pendingClean := r.states.CleanupWith(func(id string) {
		// TODO: report error
		r.store.Remove(fileStatePrefix + id)
	})
	statesCleanup.Add(int64(cleanedStates))

	r.log.Debugf(
		"Registrar states cleaned up. Before: %d, After: %d, Pending: %d",
		beforeCount, beforeCount-cleanedStates, pendingClean)

	r.gcRequired = false
	r.gcEnabled = pendingClean > 0
}

// processEventStates gets the states from the events and writes them to the registrar state
func (r *Registrar) processEventStates(states []file.State) {
	r.log.Debugf("Processing %d events", len(states))

	ts := time.Now()
	for i := range states {
		r.states.UpdateWithTs(states[i], ts)
		statesUpdate.Add(1)
	}
}

func readStatesFrom(store *statestore.Store) ([]file.State, error) {
	var states []file.State

	err := store.Each(func(key string, dec statestore.ValueDecoder) (bool, error) {
		if !strings.HasPrefix(key, fileStatePrefix) {
			return true, nil
		}

		// try to decode. Ingore faulty/incompatible values.
		var st file.State
		if err := dec.Decode(&st); err != nil {
			// XXX: Do we want to log here? In case we start to store other
			// state types in the registry, then this operation will likely fail
			// quite often, producing some false-positives in the logs...
			return true, nil
		}

		st.Id = key[len(fileStatePrefix):]
		states = append(states, st)
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	states = fixStates(states)
	states = resetStates(states)
	return states, nil
}

func writeStates(store backend.Store, states []file.State) error {
	for i := range states {
		key := fileStatePrefix + states[i].Id
		if err := store.Set(key, states[i]); err != nil {
			return err
		}
	}
	return nil
}
