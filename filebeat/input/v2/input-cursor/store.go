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

package cursor

import (
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/common/cleanup"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/go-concert"
	"github.com/elastic/go-concert/unison"
)

// store encapsulates the persistent store and the in memory state store, that
// can be ahead of the the persistent store.
// The store lifetime is managed by a reference counter. Once all owners (the
// session, and the resource cleaner) have dropped ownership, backing resources
// will be released and closed.
type store struct {
	log             *logp.Logger
	refCount        concert.RefCount
	persistentStore *statestore.Store
	ephemeralStore  *states
}

// states stores resource states in memory. When a cursor for an input is updated,
// it's state is updated first. The entry in the persistent store 'follows' the internal state.
// As long as a resources stored in states is not 'Finished', the in memory
// store is assumed to be ahead (in memory and persistent state are out of
// sync).
type states struct {
	mu    sync.Mutex
	table map[string]*resource
}

// resource holds the in memory state and keeps track of pending updates and inputs collecting
// event for the resource its key.
// A resource is assumed active for as long as at least one input has (or tries
// to) acuired the lock, and as long as there are pending updateOp instances in
// the pipeline not ACKed yet. The key can not gc'ed by the cleaner, as long as the resource is active.
//
// State chagnes and writes to the persistent store are protected using the
// stateMutex, to ensure full consistency between direct writes and updates
// after ACK.
type resource struct {
	// pending counts the number of Inputs and outstanding registry updates.
	// as long as pending is > 0 the resource is in used and must not be garbage collected.
	pending atomic.Uint64

	// lock guarantees only one input create updates for this entry
	lock unison.Mutex

	// key of the resource as used for the registry.
	key string

	// stateMutex is used to lock the resource when it is update/read from
	// multiple go-routines like the ACK handler or the input publishing an
	// event.
	// stateMutex is used to access the fields 'stored', 'state' and 'internalInSync'
	stateMutex sync.Mutex

	// stored indicates that the state is available in the registry file. It is false for new entries.
	stored bool

	// internalInSync is true if all 'Internal' metadata like TTL or update timestamp are in sync.
	// Normally resources are added when being created. But if operations failed we will retry inserting
	// them on each update operation until we eventually succeeded
	internalInSync bool

	activeCursorOperations uint
	internalState          stateInternal

	// cursor states. The cursor holds the state as it is currently known to the
	// persistent store, while pendingCursor contains the most recent update
	// (in-memory state), that still needs to be synced to the persistent store.
	// The pendingCursor is nil if there are no pending updates.
	// When processing update operations on ACKs, the state is applied to cursor
	// first, which is finally written to the persistent store. This ensures that
	// we always write the complete state of the key/value pair.
	cursor        interface{}
	pendingCursor interface{}
}

type (
	// state represents the full document as it is stored in the registry.
	//
	// The TTL and Update fields are for internal use only.
	//
	// The `Cursor` namespace is used to store the cursor information that are
	// required to continue processing from the last known position. Cursor
	// updates in the registry file are only executed after events have been
	// ACKed by the outputs. Therefore the cursor MUST NOT include any
	// information that are require to identify/track the source we are
	// collecting from.
	state struct {
		TTL     time.Duration
		Updated time.Time
		Cursor  interface{}
	}

	stateInternal struct {
		TTL     time.Duration
		Updated time.Time
	}
)

// hook into store close for testing purposes
var closeStore = (*store).close

func openStore(log *logp.Logger, statestore StateStore, prefix string) (*store, error) {
	ok := false

	persistentStore, err := statestore.Access()
	if err != nil {
		return nil, err
	}
	defer cleanup.IfNot(&ok, func() { persistentStore.Close() })

	states, err := readStates(log, persistentStore, prefix)
	if err != nil {
		return nil, err
	}

	ok = true
	return &store{
		log:             log,
		persistentStore: persistentStore,
		ephemeralStore:  states,
	}, nil
}

func (s *store) Retain() { s.refCount.Retain() }
func (s *store) Release() {
	if s.refCount.Release() {
		closeStore(s)
	}
}

func (s *store) close() {
	if err := s.persistentStore.Close(); err != nil {
		s.log.Errorf("Closing registry store did report an error: %+v", err)
	}
}

// Get returns the resource for the key.
// A new shared resource is generated if the key is not known. The generated
// resource is not synced to disk yet.
func (s *store) Get(key string) *resource {
	return s.ephemeralStore.Find(key, true)
}

// UpdateTTL updates the time-to-live of a resource. Inactive resources with expired TTL are subject to removal.
// The TTL value is part of the internal state, and will be written immediately to the persistent store.
// On update the resource its `cursor` state is used, to keep the cursor state in sync with the current known
// on disk store state.
func (s *store) UpdateTTL(resource *resource, ttl time.Duration) {
	resource.stateMutex.Lock()
	defer resource.stateMutex.Unlock()
	if resource.stored && resource.internalState.TTL == ttl {
		return
	}

	resource.internalState.TTL = ttl
	if resource.internalState.Updated.IsZero() {
		resource.internalState.Updated = time.Now()
	}

	err := s.persistentStore.Set(resource.key, state{
		TTL:     resource.internalState.TTL,
		Updated: resource.internalState.Updated,
		Cursor:  resource.cursor,
	})
	if err != nil {
		s.log.Errorf("Failed to update resource management fields for '%v'", resource.key)
		resource.internalInSync = false
	} else {
		resource.stored = true
		resource.internalInSync = true
	}
}

// Find returns the resource for a given key. If the key is unknown and create is set to false nil will be returned.
// The resource returned by Find is marked as active. (*resource).Release must be called to mark the resource as inactive again.
func (s *states) Find(key string, create bool) *resource {
	s.mu.Lock()
	defer s.mu.Unlock()

	if resource := s.table[key]; resource != nil {
		resource.Retain()
		return resource
	}

	if !create {
		return nil
	}

	// resource is owned by table(session) and input that uses the resource.
	resource := &resource{
		stored: false,
		key:    key,
		lock:   unison.MakeMutex(),
	}
	s.table[key] = resource
	resource.Retain()
	return resource
}

// IsNew returns true if we have no state recorded for the current resource.
func (r *resource) IsNew() bool {
	r.stateMutex.Lock()
	defer r.stateMutex.Unlock()
	return r.pendingCursor == nil && r.cursor == nil
}

// Retain is used to indicate that 'resource' gets an additional 'owner'.
// Owners of an resource can be active inputs or pending update operations
// not yet written to disk.
func (r *resource) Retain() { r.pending.Inc() }

// Release reduced the owner ship counter of the resource.
func (r *resource) Release() { r.pending.Dec() }

// UpdatesReleaseN is used to release ownership of N pending update operations.
func (r *resource) UpdatesReleaseN(n uint) {
	r.pending.Sub(uint64(n))
}

// Finished returns true if the resource is not in use and if there are no pending updates
// that still need to be written to the registry.
func (r *resource) Finished() bool { return r.pending.Load() == 0 }

// UnpackCursor deserializes the in memory state.
func (r *resource) UnpackCursor(to interface{}) error {
	r.stateMutex.Lock()
	defer r.stateMutex.Unlock()
	if r.activeCursorOperations == 0 {
		return typeconv.Convert(to, r.cursor)
	}
	return typeconv.Convert(to, r.pendingCursor)
}

// syncStateSnapshot returns the current insync state based on already ACKed update operations.
func (r *resource) inSyncStateSnapshot() state {
	return state{
		TTL:     r.internalState.TTL,
		Updated: r.internalState.Updated,
		Cursor:  r.cursor,
	}
}

// stateSnapshot returns the current in memory state, that already contains state updates
// not yet ACKed.
func (r *resource) stateSnapshot() state {
	cursor := r.pendingCursor
	if r.activeCursorOperations == 0 {
		cursor = r.cursor
	}

	return state{
		TTL:     r.internalState.TTL,
		Updated: r.internalState.Updated,
		Cursor:  cursor,
	}
}

func readStates(log *logp.Logger, store *statestore.Store, prefix string) (*states, error) {
	keyPrefix := prefix + "::"
	states := &states{
		table: map[string]*resource{},
	}

	err := store.Each(func(key string, dec statestore.ValueDecoder) (bool, error) {
		if !strings.HasPrefix(string(key), keyPrefix) {
			return true, nil
		}

		var st state
		if err := dec.Decode(&st); err != nil {
			log.Errorf("Failed to read regisry state for '%v', cursor state will be ignored. Error was: %+v",
				key, err)
			return true, nil
		}

		resource := &resource{
			key:            key,
			stored:         true,
			lock:           unison.MakeMutex(),
			internalInSync: true,
			internalState: stateInternal{
				TTL:     st.TTL,
				Updated: st.Updated,
			},
			cursor: st.Cursor,
		}
		states.table[resource.key] = resource

		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return states, nil
}
