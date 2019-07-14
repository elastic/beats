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
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/filebeat/input/file"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/libbeat/registry"
	"github.com/elastic/beats/v7/libbeat/registry/backend/memlog"
)

type Registrar struct {
	// registrar event input and output
	Channel chan []file.State
	out     successLogger

	// shutdown handling
	done chan struct{}
	wg   sync.WaitGroup

	// state storage
	states       *file.States       // Map with all file paths inside and the corresponding state
	provider     *registry.Registry // XXX: should not be managed by the Registrar
	store        *registry.Store    // Store keeps states in memory and on disk
	flushTimeout time.Duration

	gcEnabled, gcRequired bool
}

type successLogger interface {
	Published(n int) bool
}

var (
	statesUpdate    = monitoring.NewInt(nil, "registrar.states.update")
	statesCleanup   = monitoring.NewInt(nil, "registrar.states.cleanup")
	statesCurrent   = monitoring.NewInt(nil, "registrar.states.current")
	registryWrites  = monitoring.NewInt(nil, "registrar.writes.total")
	registryFails   = monitoring.NewInt(nil, "registrar.writes.fail")
	registrySuccess = monitoring.NewInt(nil, "registrar.writes.success")
)

// New creates a new Registrar instance, updating the registry file on
// `file.State` updates. New fails if the file can not be opened or created.
func New(cfg config.Registry, out successLogger) (*Registrar, error) {
	home := paths.Resolve(paths.Data, cfg.Path)
	migrateFile := cfg.MigrateFile
	if migrateFile != "" {
		migrateFile = paths.Resolve(paths.Data, migrateFile)
	}

	err := ensureCurrent(home, migrateFile, cfg.Permissions)
	if err != nil {
		return nil, err
	}

	memlog, err := memlog.New(memlog.Settings{
		Root:     cfg.Path,
		FileMode: cfg.Permissions,
	})
	if err != nil {
		logp.Err("Failed to open registry: %+v", err)
		return nil, err
	}

	provider := registry.New(memlog)
	store, err := provider.Get("filebeat")
	if err != nil {
		return nil, err
	}

	r := &Registrar{
		Channel:      make(chan []file.State, 1),
		out:          out,
		done:         make(chan struct{}),
		wg:           sync.WaitGroup{},
		states:       file.NewStates(),
		provider:     provider,
		store:        store,
		flushTimeout: cfg.FlushTimeout,
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
	logp.Info("States Loaded from registrar: %+v", len(states))

	return nil
}

func (r *Registrar) Start() error {
	// Load the previous log file locations now, for use in input
	err := r.loadStates()
	if err != nil {
		return fmt.Errorf("Error loading state: %v", err)
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
	logp.Info("Stopping Registrar")

	defer r.provider.Close()
	defer r.store.Close()

	close(r.done)
	r.wg.Wait()
	logp.Info("Registrar stopped")
}

func (r *Registrar) Run() {
	logp.Debug("registrar", "Starting Registrar")

	var (
		// we keep a long running write transaction.
		// The 'filebeat' store must not be used outside of the Registrar.
		tx     *registry.Tx
		timer  *time.Timer
		flushC <-chan time.Time
	)

	defer func() {
		if tx != nil {
			err := r.commit(tx)
			if err != nil {
				logp.Err("Registrar write error on shutdown: %+v", err)
			}
		}
	}()

	for {
		select {
		case <-r.done:
			logp.Info("Ending Registrar")
			return

		case <-flushC:
			timer.Stop()
			if err := r.commit(tx); err != nil {
				r.failing(err)
				return
			}
			tx = nil
			flushC = nil

		case states := <-r.Channel:
			if tx == nil {
				var err error
				tx, err = r.store.Begin(false)
				if err != nil {
					r.failing(err)
					return
				}
			}

			r.onEvents(tx, states)
			if r.flushTimeout <= 0 {
				if err := r.commit(tx); err != nil {
					r.failing(err)
					return
				}
				tx = nil
			} else if flushC == nil {
				timer = time.NewTimer(r.flushTimeout)
				flushC = timer.C
			}
		}
	}
}

func (r *Registrar) commit(tx *registry.Tx) error {
	defer tx.Close()

	// First clean up states
	r.gcStates(tx)
	states := r.states.GetStates()
	statesCurrent.Set(int64(len(states)))

	registryWrites.Inc()

	if err := tx.Commit(); err != nil {
		logp.Err("Failed to write registry state: %+v", err)
		return err
	}

	logp.Debug("registrar", "Registry file updated. %d active states.", len(states))
	registrySuccess.Inc()
	return nil
}

func (r *Registrar) failing(err error) {
	logp.Err("Registrar storage access failed with: %+v", err)
	logp.Err("Registrar is failing. Wait for shutdown.")
	<-r.done
	logp.Info("Ending failing Registrar.")
	return
}

// onEvents processes events received from the publisher pipeline
func (r *Registrar) onEvents(tx *registry.Tx, states []file.State) {
	r.processEventStates(tx, states)

	// check if we need to enable state cleanup
	if !r.gcEnabled {
		for i := range states {
			if states[i].TTL >= 0 || states[i].Finished {
				r.gcEnabled = true
				break
			}
		}
	}

	logp.Debug("registrar", "Registrar state updates processed. Count: %v", len(states))

	// new set of events received -> mark state registry ready for next
	// cleanup phase in case gc'able events are stored in the registry.
	r.gcRequired = r.gcEnabled
}

// gcStates runs a registry Cleanup. The method check if more event in the
// registry can be gc'ed in the future. If no potential removable state is found,
// the gcEnabled flag is set to false, indicating the current registrar state being
// stable. New registry update events can re-enable state gc'ing.
func (r *Registrar) gcStates(tx *registry.Tx) {
	if !r.gcRequired {
		return
	}

	beforeCount := r.states.Count()
	cleanedStates, pendingClean := r.states.CleanupWith(func(id string) {
		// TODO: report error
		tx.Remove(registry.Key(id))
	})
	statesCleanup.Add(int64(cleanedStates))

	logp.Debug("registrar",
		"Registrar states cleaned up. Before: %d, After: %d, Pending: %d",
		beforeCount, beforeCount-cleanedStates, pendingClean)

	r.gcRequired = false
	r.gcEnabled = pendingClean > 0
}

// processEventStates gets the states from the events and writes them to the registrar state
func (r *Registrar) processEventStates(tx *registry.Tx, states []file.State) {
	logp.Debug("registrar", "Processing %d events", len(states))

	ts := time.Now()
	for i := range states {
		r.states.UpdateWithTs(states[i], ts)

		// TODO: report error
		tx.Set(registry.Key(states[i].ID()), states[i])

		statesUpdate.Add(1)
	}
}
