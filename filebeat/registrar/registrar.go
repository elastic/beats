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
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/beats/v7/filebeat/input/file"
	"github.com/elastic/beats/v7/filebeat/input/v2/statemanager"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// Registrar is one owner's handle onto the registrar shared by every owner
// whose state store resolves to the same backend key. States sent on Channel
// are forwarded to the shared registrar, which holds the single in-memory
// table for the backend and is the only writer to it. Acknowledgements are
// reported back to this owner's successLogger only.
type Registrar struct {
	log *logp.Logger

	// Channel receives acknowledged states from this owner's pipeline.
	Channel chan []file.State
	out     successLogger

	stateStore   statestore.States
	flushTimeout time.Duration

	shared  *sharedRegistrar
	release func()

	state    atomic.Int32
	done     chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

type successLogger interface {
	Published(n int) bool
}

// Registrar handle lifecycle. A handle is started only once it holds a
// reference to the shared registrar, so Stop can rely on release being set.
const (
	handleIdle int32 = iota
	handleStarting
	handleStarted
)

// update is one message from an owner to the shared registrar. A nil states
// slice with a non-nil flushed channel requests a commit; flushed is closed
// once every state received before it has been persisted and acknowledged.
// leaving marks the owner's final flush: its harvesters are stopped, so every
// state it ever reported is marked finished before the commit.
type update struct {
	states  []file.State
	owner   *Registrar
	flushed chan struct{}
	leaving bool
}

// sharedRegistrar owns the in-memory state table and the store for one
// backend key. It is created by the first owner to start and closed after the
// last owner stops.
type sharedRegistrar struct {
	log *logp.Logger

	updates chan update
	states  *file.States
	store   *statestore.Store

	// flushTimeout is the shortest flush timeout requested by any owner, in
	// nanoseconds. Zero or less commits after every update.
	flushTimeout atomic.Int64

	// pending counts states applied but not yet committed, per owner.
	pending map[*Registrar]int
	// owned records the state IDs each active owner has reported.
	owned        map[*Registrar]map[string]struct{}
	flushWaiters []chan struct{}

	gcEnabled, gcRequired bool
}

var sharedRegistrars = statemanager.NewCache[*sharedRegistrar](func(s *sharedRegistrar) {
	if err := s.store.Close(); err != nil {
		s.log.Errorf("Error closing registrar store: %v", err)
	}
})

var (
	statesUpdate    = monitoring.NewInt(nil, "registrar.states.update")
	statesCleanup   = monitoring.NewInt(nil, "registrar.states.cleanup")
	statesCurrent   = monitoring.NewInt(nil, "registrar.states.current")
	registryWrites  = monitoring.NewInt(nil, "registrar.writes.total")
	registryFails   = monitoring.NewInt(nil, "registrar.writes.fail")
	registrySuccess = monitoring.NewInt(nil, "registrar.writes.success")
)

const fileStatePrefix = "filebeat::logs::"

// New creates a Registrar handle. The backing store is opened, and previous
// states loaded, on Start; owners sharing a backend key share both.
func New(stateStore statestore.States, out successLogger, flushTimeout time.Duration, logger *logp.Logger) (*Registrar, error) {
	if stateStore == nil {
		return nil, fmt.Errorf("registrar requires a state store")
	}
	return &Registrar{
		log:          logger.Named("registrar"),
		Channel:      make(chan []file.State, 1),
		out:          out,
		stateStore:   stateStore,
		flushTimeout: flushTimeout,
		done:         make(chan struct{}),
	}, nil
}

// GetStates returns the states of the shared registrar. It returns nil before
// Start.
func (r *Registrar) GetStates() []file.State {
	if r.shared == nil {
		return nil
	}
	return r.shared.states.GetStates()
}

// Start joins or creates the shared registrar for this owner's backend key and
// begins forwarding states from Channel to it. Start may be called once, and
// not after Stop.
func (r *Registrar) Start() error {
	select {
	case <-r.done:
		return fmt.Errorf("registrar already stopped")
	default:
	}
	if !r.state.CompareAndSwap(handleIdle, handleStarting) {
		return fmt.Errorf("registrar already started")
	}

	key := r.stateStore.StoreKey("", "")
	shared, release, err := sharedRegistrars.Acquire(
		key,
		func() (*sharedRegistrar, error) {
			return openShared(r.stateStore, r.flushTimeout, r.log)
		},
		func(ctx context.Context, s *sharedRegistrar) { s.run(ctx) },
		func() { r.log.Debugf("Waiting for shared registrar %q to initialize", key) },
		func() { r.log.Debugf("Joining shared registrar %q", key) },
	)
	if err != nil {
		r.state.Store(handleIdle)
		return fmt.Errorf("error loading state: %w", err)
	}
	r.shared = shared
	r.release = release
	r.state.Store(handleStarted)

	effective, lowered := shared.lowerFlushTimeout(r.flushTimeout)
	if effective != r.flushTimeout {
		r.log.Debugf("Shared registrar %q flushes every %v, shorter than the configured %v", key, effective, r.flushTimeout)
	}
	if lowered {
		// Commit whatever is waiting on the previous, longer timeout.
		flushed := make(chan struct{})
		shared.updates <- update{owner: r, flushed: flushed}
		<-flushed
	}
	r.log.Infof("States Loaded from registrar: %+v", shared.states.Count())

	r.wg.Go(r.forward)
	return nil
}

// Stop flushes this owner's pending states, waits for their acknowledgement,
// and leaves the shared registrar. The store is closed once the last owner
// leaves.
func (r *Registrar) Stop() {
	r.stopOnce.Do(func() {
		r.log.Info("Stopping Registrar")
		defer r.log.Info("Registrar stopped")

		close(r.done)
		r.wg.Wait()
		if r.release != nil {
			r.release()
		}
	})
}

func (r *Registrar) forward() {
	for {
		select {
		case <-r.done:
			for drained := false; !drained; {
				select {
				case states := <-r.Channel:
					r.shared.updates <- update{states: states, owner: r}
				default:
					drained = true
				}
			}
			flushed := make(chan struct{})
			r.shared.updates <- update{owner: r, flushed: flushed, leaving: true}
			<-flushed
			return

		case states := <-r.Channel:
			r.shared.updates <- update{states: states, owner: r}
		}
	}
}

func openShared(stateStore statestore.States, flushTimeout time.Duration, logger *logp.Logger) (*sharedRegistrar, error) {
	store, err := stateStore.StoreFor("", "")
	if err != nil {
		return nil, err
	}

	s := &sharedRegistrar{
		log:     logger,
		updates: make(chan update),
		states:  file.NewStates(logger),
		store:   store,
		pending: make(map[*Registrar]int),
		owned:   make(map[*Registrar]map[string]struct{}),
	}
	s.flushTimeout.Store(int64(max(flushTimeout, 0)))

	states, err := readStatesFrom(store, logger)
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("can not load filebeat registry state: %w", err)
	}
	s.states.SetStates(states)
	return s, nil
}

// lowerFlushTimeout shortens the shared flush timeout to d if d is shorter.
// It returns the timeout in effect afterwards and whether it was lowered.
func (s *sharedRegistrar) lowerFlushTimeout(d time.Duration) (time.Duration, bool) {
	d = max(d, 0)
	for {
		current := time.Duration(s.flushTimeout.Load())
		if current <= d {
			return current, false
		}
		if s.flushTimeout.CompareAndSwap(int64(current), int64(d)) {
			return d, true
		}
	}
}

func (s *sharedRegistrar) run(ctx context.Context) {
	s.log.Debug("Starting Registrar")
	defer s.log.Debug("Stopping Registrar")

	var (
		timer  *time.Timer
		flushC <-chan time.Time
	)

	stopTimer := func() {
		if timer != nil {
			timer.Stop()
		}
		timer = nil
		flushC = nil
	}

	for {
		select {
		case <-ctx.Done():
			s.log.Info("Ending Registrar")
			s.drain()
			stopTimer()
			s.commitStateUpdates()
			return

		case u := <-s.updates:
			s.apply(u)
			timeout := time.Duration(s.flushTimeout.Load())
			if u.flushed != nil || timeout <= 0 {
				stopTimer()
				s.commitStateUpdates()
				continue
			}
			s.gcStates()
			if flushC == nil && len(u.states) > 0 {
				timer = time.NewTimer(timeout)
				flushC = timer.C
			}

		case <-flushC:
			stopTimer()
			s.commitStateUpdates()
		}
	}
}

// drain applies every update that is already queued without blocking. It runs
// only after the last owner has released, so no owner is still sending.
func (s *sharedRegistrar) drain() {
	for {
		select {
		case u := <-s.updates:
			s.apply(u)
		default:
			return
		}
	}
}

func (s *sharedRegistrar) apply(u update) {
	if len(u.states) > 0 {
		s.onEvents(u.states, u.owner)
	}
	if u.leaving {
		s.finishStatesOf(u.owner)
	}
	if u.flushed != nil {
		s.flushWaiters = append(s.flushWaiters, u.flushed)
	}
}

// finishStatesOf marks every state the owner reported as finished and forgets
// the owner. The owner's harvesters have stopped, so a later input matching
// these files must be allowed to start, exactly as after a process restart.
func (s *sharedRegistrar) finishStatesOf(owner *Registrar) {
	for id := range s.owned[owner] {
		st := s.states.FindPrevious(file.State{Id: id})
		if st.Id == "" || st.Finished {
			continue
		}
		st.Finished = true
		s.states.UpdateWithTs(st, st.Timestamp)
	}
	delete(s.owned, owner)
}

func (s *sharedRegistrar) commitStateUpdates() {
	// First clean up states
	s.gcStates()
	states := s.states.GetStates()
	statesCurrent.Set(int64(len(states)))

	registryWrites.Inc()
	if err := writeStates(s.store, states); err != nil {
		s.log.Errorf("Error writing registrar state to statestore: %v", err)
		registryFails.Inc()
	} else {
		registrySuccess.Inc()
	}
	s.log.Debugf("Registry file updated. %d active states.", len(states))

	for owner, n := range s.pending {
		if owner != nil && owner.out != nil {
			owner.out.Published(n)
		}
	}
	clear(s.pending)

	for _, ch := range s.flushWaiters {
		close(ch)
	}
	s.flushWaiters = nil
}

// onEvents processes events received from one owner's publisher pipeline
func (s *sharedRegistrar) onEvents(states []file.State, owner *Registrar) {
	s.processEventStates(states)
	s.pending[owner] += len(states)
	ids := s.owned[owner]
	if ids == nil {
		ids = make(map[string]struct{})
		s.owned[owner] = ids
	}
	for i := range states {
		ids[states[i].Id] = struct{}{}
	}

	// check if we need to enable state cleanup
	if !s.gcEnabled {
		for i := range states {
			if states[i].TTL >= 0 || states[i].Finished {
				s.gcEnabled = true
				break
			}
		}
	}

	s.log.Debugf("Registrar state updates processed. Count: %v", len(states))

	// new set of events received -> mark state registry ready for next
	// cleanup phase in case gc'able events are stored in the registry.
	s.gcRequired = s.gcEnabled
}

// gcStates runs a registry Cleanup. The method check if more event in the
// registry can be gc'ed in the future. If no potential removable state is found,
// the gcEnabled flag is set to false, indicating the current registrar state being
// stable. New registry update events can re-enable state gc'ing.
func (s *sharedRegistrar) gcStates() {
	if !s.gcRequired {
		return
	}

	beforeCount := s.states.Count()
	cleanedStates, pendingClean := s.states.CleanupWith(func(id string) {
		s.store.Remove(fileStatePrefix + id) //nolint:errcheck // TODO: report error
	})
	statesCleanup.Add(int64(cleanedStates))

	s.log.Debugf(
		"Registrar states cleaned up. Before: %d, After: %d, Pending: %d",
		beforeCount, beforeCount-cleanedStates, pendingClean)

	s.gcRequired = false
	s.gcEnabled = pendingClean > 0
}

// processEventStates gets the states from the events and writes them to the registrar state
func (s *sharedRegistrar) processEventStates(states []file.State) {
	s.log.Debugf("Processing %d events", len(states))

	ts := time.Now()
	for i := range states {
		s.states.UpdateWithTs(states[i], ts)
		statesUpdate.Add(1)
	}
}

func readStatesFrom(store *statestore.Store, logger *logp.Logger) ([]file.State, error) {
	var states []file.State

	err := store.Each(func(key string, dec statestore.ValueDecoder) (bool, error) {
		if !strings.HasPrefix(key, fileStatePrefix) {
			return true, nil
		}

		// try to decode. Ignore faulty/incompatible values.
		var st file.State
		if err := dec.Decode(&st); err != nil {
			// XXX: Do we want to log here? In case we start to store other
			// state types in the registry, then this operation will likely fail
			// quite often, producing some false-positives in the logs...
			return true, nil //nolint:nilerr // Ignore per comment above
		}

		st.Id = key[len(fileStatePrefix):]
		states = append(states, st)
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	states = fixStates(states, logger)
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
