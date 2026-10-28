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
	"errors"
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

// ErrStopped is returned by Start when the registrar was already stopped, so
// callers do not build inputs on a registrar that will never run.
var ErrStopped = errors.New("registrar stopped")

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

	// The registrar is started lazily by the first V1 input to be created, which
	// cannot know whether another already did. startOnce makes Start safe to
	// call repeatedly; startErr is kept so every call reports a failed start.
	startOnce sync.Once
	startErr  error

	done     chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

type successLogger interface {
	Published(n int) bool
}

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

	pending      map[*Registrar]int
	mu           sync.Mutex
	owned        map[*Registrar]map[string]struct{}
	acked        map[*Registrar]map[string]file.State
	warned       map[string]struct{}
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
	return r.shared.statesFor(r, r.shared.states.GetStates())
}

// StartStates returns the states for a new input: what was flushed when this
// owner started, plus everything this owner acknowledged since, as in a
// registrar of its own. Other owners' progress is never visible.
func (r *Registrar) StartStates() []file.State {
	if r.shared == nil {
		return nil
	}
	r.shared.mu.Lock()
	states := make([]file.State, 0, len(r.shared.acked[r]))
	for _, st := range r.shared.acked[r] {
		states = append(states, st)
	}
	r.shared.mu.Unlock()
	return r.shared.statesFor(r, states)
}

func (s *sharedRegistrar) statesFor(owner *Registrar, states []file.State) []file.State {
	s.mu.Lock()
	defer s.mu.Unlock()
	mine := s.owned[owner]
	for i := range states {
		if _, ok := mine[states[i].Id]; !ok {
			states[i].Finished = true
		}
	}
	return states
}

// Start joins or creates the shared registrar for this owner's backend key and
// begins forwarding states from Channel to it. It is called when a V1 input is
// created (see input.RunnerFactory.Create), and is safe to call any number of
// times: only the first call does anything. If that first start fails, every
// later call returns the same error, and after Stop every call returns
// ErrStopped.
func (r *Registrar) Start() error {
	r.startOnce.Do(func() {
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
			r.startErr = fmt.Errorf("error loading state: %w", err)
			return
		}
		r.shared = shared
		r.release = release
		if err := shared.seedBaseline(r); err != nil {
			release()
			r.shared, r.release = nil, nil
			r.startErr = fmt.Errorf("error loading state: %w", err)
			return
		}

		effective, lowered := shared.lowerFlushTimeout(r.flushTimeout)
		if effective != r.flushTimeout {
			r.log.Debugf("Shared registrar %q flushes every %v, shorter than the configured %v", key, effective, r.flushTimeout)
		}
		if lowered {
			flushed := make(chan struct{})
			shared.updates <- update{owner: r, flushed: flushed}
			<-flushed
		}
		r.log.Infof("States Loaded from registrar: %+v", shared.states.Count())

		r.wg.Go(r.forward)
	})

	return r.startErr
}

// Stop flushes this owner's pending states, waits for their acknowledgement,
// and leaves the shared registrar. The store is closed once the last owner
// leaves. It is safe to call more than once.
func (r *Registrar) Stop() {
	r.stopOnce.Do(r.stop)
}

func (r *Registrar) stop() {
	r.log.Info("Stopping Registrar")
	defer r.log.Info("Registrar stopped")

	// Claim startOnce so an input created concurrently with shutdown cannot
	// start a registrar after this point. Start then reports ErrStopped
	// instead of joining a shared registrar that is about to go away.
	r.startOnce.Do(func() { r.startErr = ErrStopped })

	close(r.done)
	r.wg.Wait()
	if r.release != nil {
		r.release()
	}
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
		acked:   make(map[*Registrar]map[string]file.State),
		warned:  make(map[string]struct{}),
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

// seedBaseline records the states flushed at the time the owner starts, the
// part of its view that it did not acknowledge itself. The lock is held across
// the read so a concurrent cleanup cannot be undone by a stale seed.
func (s *sharedRegistrar) seedBaseline(owner *Registrar) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	states, err := readStatesFrom(s.store, owner.log)
	if err != nil {
		return err
	}
	baseline := make(map[string]file.State, len(states))
	for _, st := range states {
		baseline[st.Id] = st
	}
	s.acked[owner] = baseline
	return nil
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

// finishStatesOf marks every state the owner is harvesting as finished and
// forgets the owner. The owner's harvesters have stopped, so a later input matching
// these files must be allowed to start, exactly as after a process restart.
func (s *sharedRegistrar) finishStatesOf(owner *Registrar) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id := range s.owned[owner] {
		delete(s.warned, id)
		st := s.states.FindPrevious(file.State{Id: id})
		if st.Id == "" || st.Finished {
			continue
		}
		st.Finished = true
		s.states.UpdateWithTs(st, time.Now())
	}
	delete(s.owned, owner)
	delete(s.acked, owner)
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
	s.trackOwnership(states, owner)

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

func (s *sharedRegistrar) trackOwnership(states []file.State, owner *Registrar) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := s.owned[owner]
	if ids == nil {
		ids = make(map[string]struct{})
		s.owned[owner] = ids
	}
	acked := s.acked[owner]
	if acked == nil {
		acked = make(map[string]file.State)
		s.acked[owner] = acked
	}
	for i := range states {
		st := &states[i]
		acked[st.Id] = *st
		if st.Finished {
			delete(ids, st.Id)
			delete(s.warned, st.Id)
			continue
		}
		if _, ok := ids[st.Id]; ok {
			continue
		}
		ids[st.Id] = struct{}{}
		if _, ok := s.warned[st.Id]; ok {
			continue
		}
		for other, otherIDs := range s.owned {
			if other == owner {
				continue
			}
			if _, ok := otherIDs[st.Id]; ok {
				s.warned[st.Id] = struct{}{}
				for _, r := range []*Registrar{owner, other} {
					r.log.Warnf("File %s is harvested by more than one receiver sharing this registry: "+
						"overlapping log input paths are unsupported and duplicate events", st.Source)
				}
				break
			}
		}
	}
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
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.warned, id)
		for _, ids := range s.owned {
			delete(ids, id)
		}
		for _, acked := range s.acked {
			delete(acked, id)
		}
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
