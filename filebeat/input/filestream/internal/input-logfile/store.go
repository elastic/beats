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

package input_logfile

import (
	"fmt"
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

// sourceStore is a store which can access resources using the Source
// from an input.
type sourceStore struct {
	identifier *sourceIdentifier
	store      *store
}

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

	// current identity version when updated stateMutex must be locked.
	// Pending updates will be discarded if it is increased.
	version, lockedVersion uint

	// lock guarantees only one input create updates for this entry
	lock unison.Mutex

	// key of the resource as used for the registry.
	key string

	// stateMutex is used to lock the resource when it is update/read from
	// multiple go-routines like the ACK handler or the input publishing an
	// event.
	// stateMutex is used to access the fields 'stored', 'state', 'internalInSync' and 'version'.
	stateMutex sync.Mutex

	// stored indicates that the state is available in the registry file. It is false for new entries.
	stored bool
	// invalid indicates if the resource has been marked for deletion, if yes, it cannot be overwritten
	// in the persistent state.
	invalid bool

	activeCursorOperations uint
	internalState          stateInternal

	// cursor states. The cursor holds the state as it is currently known to the
	// persistent store, while pendingCursor contains the most recent update
	// (in-memory state), that still needs to be synced to the persistent store.
	// The pendingCursor is nil if there are no pending updates.
	// When processing update operations on ACKs, the state is applied to cursor
	// first, which is finally written to the persistent store. This ensures that
	// we always write the complete state of the key/value pair.
	cursor             interface{}
	pendingCursorValue interface{}
	pendingUpdate      interface{} // delta value of most recent pending updateOp
	cursorMeta         interface{}
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
		Meta    interface{}
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

func newSourceStore(s *store, identifier *sourceIdentifier) *sourceStore {
	return &sourceStore{
		store:      s,
		identifier: identifier,
	}
}

func (s *sourceStore) FindCursorMeta(src Source, v interface{}) error {
	key := s.identifier.ID(src)
	return s.store.findCursorMeta(key, v)
}

func (s *sourceStore) UpdateMetadata(src Source, v interface{}) error {
	key := s.identifier.ID(src)
	return s.store.updateMetadata(key, v)
}

func (s *sourceStore) Remove(src Source) error {
	key := s.identifier.ID(src)
	return s.store.remove(key)
}

func (s *sourceStore) ResetCursor(src Source, cur interface{}) error {
	key := s.identifier.ID(src)
	return s.store.resetCursor(key, cur)
}

// CleanIf sets the TTL of a resource if the predicate return true.
func (s *sourceStore) CleanIf(pred func(v Value) bool) {
	s.store.ephemeralStore.mu.Lock()
	defer s.store.ephemeralStore.mu.Unlock()

	for key, res := range s.store.ephemeralStore.table {
		if !s.identifier.MatchesInput(key) {
			continue
		}

		if !res.lock.TryLock() {
			continue
		}

		remove := pred(res)
		if remove {
			s.store.UpdateTTL(res, 0)
		}
		res.lock.Unlock()
	}
}

// FixUpIdentifiers copies an existing resource to a new ID and marks the previous one
// for removal.
func (s *sourceStore) FixUpIdentifiers(getNewID func(v Value) (string, interface{})) {
	s.store.ephemeralStore.mu.Lock()
	defer s.store.ephemeralStore.mu.Unlock()

	for key, res := range s.store.ephemeralStore.table {
		if !s.identifier.MatchesInput(key) {
			continue
		}

		res.lock.Lock()

		newKey, updatedMeta := getNewID(res)
		if len(newKey) > 0 && res.internalState.TTL > 0 {
			if _, ok := s.store.ephemeralStore.table[newKey]; ok {
				res.lock.Unlock()
				continue
			}

			// Pending updates due to events that have not yet been ACKed
			// are not included in the copy. Collection on
			// the copy start from the last known ACKed position.
			// This might lead to data duplication because the harvester
			// will pickup from the last ACKed postion using the new key
			// and the pending updates will affect the entry with the oldKey.
			r := res.copyWithNewKey(newKey)
			r.cursorMeta = updatedMeta
			r.stored = false
			s.store.writeState(r)

			// Add the new resource to the ephemeralStore so the rest of the
			// codebase can have access to the new value
			s.store.ephemeralStore.table[newKey] = r

			// Remove the old key from the store
			s.store.UpdateTTL(res, 0) // aka delete. See store.remove for details
			s.store.log.Infof("migrated entry in registry from '%s' to '%s'", key, newKey)
		}

		res.lock.Unlock()
	}
}

// UpdateIdentifiers copies an existing resource to a new ID and marks the previous one
// for removal.
func (s *sourceStore) UpdateIdentifiers(getNewID func(v Value) (string, interface{})) {
	s.store.ephemeralStore.mu.Lock()
	defer s.store.ephemeralStore.mu.Unlock()

	for key, res := range s.store.ephemeralStore.table {
		if !s.identifier.MatchesInput(key) {
			continue
		}

		if !res.lock.TryLock() {
			continue
		}

		newKey, updatedMeta := getNewID(res)
		if len(newKey) > 0 && res.internalState.TTL > 0 {
			if _, ok := s.store.ephemeralStore.table[newKey]; ok {
				res.lock.Unlock()
				continue
			}

			// Pending updates due to events that have not yet been ACKed
			// are not included in the copy. Collection on
			// the copy start from the last known ACKed position.
			// This might lead to data duplication because the harvester
			// will pickup from the last ACKed postion using the new key
			// and the pending updates will affect the entry with the oldKey.
			r := res.copyWithNewKey(newKey)
			r.cursorMeta = updatedMeta
			r.stored = false
			s.store.writeState(r)
		}

		res.lock.Unlock()
	}
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

func (s *store) findCursorMeta(key string, to interface{}) error {
	resource := s.ephemeralStore.Find(key, false)
	if resource == nil {
		return fmt.Errorf("resource '%s' not found", key)
	}
	return typeconv.Convert(to, resource.cursorMeta)
}

// updateMetadata updates the cursor metadata in the persistent store.
func (s *store) updateMetadata(key string, meta interface{}) error {
	resource := s.ephemeralStore.Find(key, true)
	if resource == nil {
		return fmt.Errorf("resource '%s' not found", key)
	}

	resource.cursorMeta = meta

	s.writeState(resource)
	return nil
}

// writeState writes the state to the persistent store.
// WARNING! it does not lock the store or the resource.
func (s *store) writeState(r *resource) {
	if r.invalid {
		return
	}

	err := s.persistentStore.Set(r.key, r.inSyncStateSnapshot())
	if err != nil {
		s.log.Errorf("Failed to update resource fields for '%v'", r.key)
	} else {
		r.stored = true
	}
}

// resetCursor sets the cursor to the value in cur in the persistent store and
// drops all pending cursor operations.
func (s *store) resetCursor(key string, cur interface{}) error {
	r := s.ephemeralStore.Find(key, false)
	if r == nil {
		return fmt.Errorf("resource '%s' not found", key)
	}
	defer r.Release()

	r.stateMutex.Lock()
	defer r.stateMutex.Unlock()

	r.version++
	r.UpdatesReleaseN(r.activeCursorOperations)
	r.activeCursorOperations = 0
	r.pendingCursorValue = nil
	r.pendingUpdate = nil
	typeconv.Convert(&r.cursor, cur)

	s.writeState(r)

	return nil
}

// Removes marks an entry for removal by setting its TTL to zero.
func (s *store) remove(key string) error {
	resource := s.ephemeralStore.Find(key, false)
	if resource == nil {
		return fmt.Errorf("resource '%s' not found", key)
	}
	s.UpdateTTL(resource, 0)
	return nil
}

// UpdateTTL updates the time-to-live of a resource. Inactive resources with expired TTL are subject to removal.
// The TTL value is part of the internal state, and will be written immediately to the persistent store.
// On update the resource its `cursor` state is used, to keep the cursor state in sync with the current known
// on disk store state.
//
// If the TTL of the resource is set to 0, once it is persisted, it is going to be removed from the
// store in the next cleaner run. The resource also gets invalidated to make sure new updates are not
// saved to the registry.
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

	s.writeState(resource)

	if resource.isDeleted() {
		// version must be incremented to make sure existing resource
		// instances do not overwrite the removal of the entry
		resource.version++
		// invalidate it after it has been persisted to make sure it cannot
		// be overwritten in the persistent store
		resource.invalid = true
	}
}

// Find returns the resource for a given key. If the key is unknown and create is set to false nil will be returned.
// The resource returned by Find is marked as active. (*resource).Release must be called to mark the resource as inactive again.
func (s *states) Find(key string, create bool) *resource {
	s.mu.Lock()
	defer s.mu.Unlock()

	if resource := s.table[key]; resource != nil && !resource.isDeleted() {
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
	return r.pendingCursorValue == nil && r.pendingUpdate == nil && r.cursor == nil
}

func (r *resource) isDeleted() bool {
	return !r.internalState.Updated.IsZero() && r.internalState.TTL == 0
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
	return typeconv.Convert(to, r.activeCursor())
}

func (r *resource) UnpackCursorMeta(to interface{}) error {
	return typeconv.Convert(to, r.cursorMeta)
}

// syncStateSnapshot returns the current insync state based on already ACKed update operations.
func (r *resource) inSyncStateSnapshot() state {
	return state{
		TTL:     r.internalState.TTL,
		Updated: r.internalState.Updated,
		Cursor:  r.cursor,
		Meta:    r.cursorMeta,
	}
}

func (r *resource) copyInto(dst *resource) {
	r.stateMutex.Lock()
	defer r.stateMutex.Unlock()

	internalState := r.internalState

	// This is required to prevent the cleaner from removing the
	// entry from the registry immediately.
	// It still might be removed if the output is blocked for a long
	// time. If removed the whole file is resent to the output when found/updated.
	internalState.Updated = time.Now()
	dst.stored = r.stored
	dst.internalState = internalState
	dst.activeCursorOperations = r.activeCursorOperations
	dst.cursor = r.cursor
	dst.pendingCursorValue = nil
	dst.pendingUpdate = nil
	dst.cursorMeta = r.cursorMeta
	dst.lock = unison.MakeMutex()
}

func (r *resource) copyWithNewKey(key string) *resource {
	internalState := r.internalState

	// This is required to prevent the cleaner from removing the
	// entry from the registry immediately.
	// It still might be removed if the output is blocked for a long
	// time. If removed the whole file is resent to the output when found/updated.
	internalState.Updated = time.Now()
	return &resource{
		key:                    key,
		stored:                 r.stored,
		internalState:          internalState,
		activeCursorOperations: r.activeCursorOperations,
		cursor:                 r.cursor,
		pendingCursorValue:     nil,
		pendingUpdate:          nil,
		cursorMeta:             r.cursorMeta,
		lock:                   unison.MakeMutex(),
	}
}

// pendingCursor returns the current published cursor state not yet ACKed.
//
// Note: The stateMutex must be locked when calling pendingCursor.
func (r *resource) pendingCursor() interface{} {
	if r.pendingUpdate != nil {
		var tmp interface{}
		typeconv.Convert(&tmp, &r.cursor)
		typeconv.Convert(&tmp, r.pendingUpdate)
		r.pendingCursorValue = tmp
		r.pendingUpdate = nil
	}
	return r.pendingCursorValue
}

// activeCursor
func (r *resource) activeCursor() interface{} {
	if r.activeCursorOperations != 0 {
		return r.pendingCursor()
	}
	return r.cursor
}

// stateSnapshot returns the current in memory state, that already contains state updates
// not yet ACKed.
func (r *resource) stateSnapshot() state {
	return state{
		TTL:     r.internalState.TTL,
		Updated: r.internalState.Updated,
		Cursor:  r.activeCursor(),
		Meta:    r.cursorMeta,
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
			key:    key,
			stored: true,
			lock:   unison.MakeMutex(),
			internalState: stateInternal{
				TTL:     st.TTL,
				Updated: st.Updated,
			},
			cursor:     st.Cursor,
			cursorMeta: st.Meta,
		}
		states.table[resource.key] = resource

		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return states, nil
}
