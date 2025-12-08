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
	"sync/atomic"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/cleanup"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/statestore"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-concert"
	"github.com/elastic/go-concert/unison"
)

// sourceStore is a store which can access resources using the Source
// from an input.
type sourceStore struct {
	// identifier is the sourceIdentifier used to generate IDs fro this store.
	identifier *SourceIdentifier
	// identifiersToTakeOver are sourceIdentifier from previous input instances
	// that this sourceStore will take states over.
	identifiersToTakeOver []*SourceIdentifier
	// store is the underlying store that encapsulates
	// the in-memory and persistent store.
	store *store
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

func openStore(log *logp.Logger, statestore statestore.States, prefix string) (*store, error) {
	ok := false

	persistentStore, err := statestore.StoreFor("")
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

// newSourceStore store returns a souceStore that will operate on the provided
// store. identifier is required and is used to generate the ID for the
// resources stored on store. identifiersToTakeOver is used by the TakeOver
// method when taking over states from other Filestream inputs.
// identifiersToTakeOver is optional and can be nil.
func newSourceStore(
	s *store,
	identifier *SourceIdentifier,
	identifiersToTakeOver []*SourceIdentifier,
) *sourceStore {

	return &sourceStore{
		store:                 s,
		identifier:            identifier,
		identifiersToTakeOver: identifiersToTakeOver,
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

// UpdateIdentifiers copies an existing resource to a new ID and marks the previous one
// for removal.
func (s *sourceStore) UpdateIdentifiers(getNewID func(v Value) (string, any)) {
	s.store.ephemeralStore.mu.Lock()
	defer s.store.ephemeralStore.mu.Unlock()

	for key, res := range s.store.ephemeralStore.table {
		// Entries in the registry are soft deleted, once the gcStore runs,
		// they're actually removed from the in-memory registry (ephemeralStore)
		// and marked as removed in the registry operations log. So we need
		// to skip all entries that were soft deleted.
		if res.isDeleted() {
			continue
		}

		if !s.identifier.MatchesInput(key) {
			continue
		}

		if !res.lock.TryLock() {
			s.store.log.Infof("cannot lock '%s', will not update registry for it", key)
			continue
		}

		newKey, updatedMeta := getNewID(res)
		if len(newKey) > 0 {
			if _, ok := s.store.ephemeralStore.table[newKey]; ok {
				res.lock.Unlock()
				continue
			}

			r := res.copyWithNewKey(newKey)
			r.cursorMeta = updatedMeta
			r.stored = false
			// writeState only writes to the log file (disk)
			// the write is synchronous
			s.store.writeState(r)

			// Add the new resource to the ephemeralStore so the rest of the
			// codebase can have access to the new value
			s.store.ephemeralStore.table[newKey] = r

			// Remove the old key from the store aka delete. This is also
			// synchronously written to the disk.
			// We cannot use store.remove because it will
			// acquire the same lock we hold, causing a deadlock.
			// See store.remove for details.
			// Fully remove the old resource from all stores.
			//  - 1. Update the TLL, which soft-deletes it. This is the
			//    mechanism used by store.remove. We cannot call store.remove
			//    because it will acquire a lock we're holding.
			//  - 2. Remove the resource from the in-memory store
			//  - 3. Finally, synchronously remove it from the disk store.
			s.store.UpdateTTL(res, 0)
			delete(s.store.ephemeralStore.table, res.key)
			_ = s.store.persistentStore.Remove(res.key)
			s.store.log.Infof("migrated entry in registry from '%s' to '%s'. Cursor: %v", key, newKey, r.cursor)
		}

		res.lock.Unlock()
	}
}

// TakeOver allows one Filestream input to take over states from other
// Filestream inputs or the Log input. fn should return the new registry ID
// and new CursorMeta. If fn returns an empty string, the entry is skipped.
//
// When fn returns a valid ID, the old resource is removed from both,
// the in-memory and persistent store. The operations are synchronous.
//
// If the resource migrated was from the Log input, `TakeOver` will
// remove it from the persistent store, however the Log input reigstrar
// will write it back when Filebeat is shutting down. However,
// there is a mechanism in place to detect this situation and avoid
// migrating the same state over and over again.
// See the comments on this method for more details.
func (s *sourceStore) TakeOver(fn func(Value) (string, any)) {
	matchPreviousFilestreamIDs := func(key string) bool {
		for _, identifier := range s.identifiersToTakeOver {
			if identifier.MatchesInput(key) {
				return true
			}
		}

		return false
	}

	// Iterate through the states from any Filestream input
	fromFilestreamInput := map[string]struct{}{}
	for key, res := range s.store.ephemeralStore.table {
		// Entries in the registry are soft deleted, once the gcStore runs,
		// they're actually removed from the in-memory registry (ephemeralStore)
		// and marked as removed in the registry operations log. So we need
		// to skip all entries that were soft deleted.
		if res.isDeleted() {
			continue
		}

		if !matchPreviousFilestreamIDs(key) {
			continue
		}

		fromFilestreamInput[key] = struct{}{}
	}

	// Iterate through the whole store, no matter input type or input ID.
	// That's the only way to access the log input states.
	// We only iterate through the whole store if we're not migrating from
	// a Filestream input
	fromLogInput := map[string]logInputState{}
	if len(s.identifiersToTakeOver) == 0 {
		_ = s.store.persistentStore.Each(func(key string, value statestore.ValueDecoder) (bool, error) {
			if strings.HasPrefix(key, "filebeat::logs::") {
				m := mapstr.M{}
				if err := value.Decode(&m); err != nil {
					return true, err
				}
				st, err := logInputStateFromMapM(m)
				if err != nil {
					// Log the error and continue
					s.store.log.Errorf("cannot read Log input state: %s", err)
					return true, nil
				}
				// That is a workaround for the problems with the
				// Log input Registrar (`filebeat/registrar`) and the way it
				// handles states.
				// There are two problems:
				//  - 1. The Log input store/registrar does not have an API for
				//       removing states
				//  - 2. When `registrar.Registrar` starts, it copies all states
				//       belonging to the Log input from the disk store into
				//       memory and when the Registrar is shutting down, it
				//       writes all states to the disk. This all happens even
				//       if no Log input was ever started.
				// This means that no matter what we do here, the states from
				// the Log input are always re-written to disk.
				// See: filebeat/registrar/registrar.go, deferred statement on
				// `Registrar.Run`.
				//
				// However, there is a "reset state" code, that runs
				// during the Registrar initialisation and sets the
				// TTL to -2, once the Log input havesting that file starts
				// the TTL is set to -1 (never expires) or the configured
				// value.
				// See: filebeat/registrar/registrar.go (readStatesFrom) and
				// filebeat/beater/filebeat.go (registrar.Start())
				//
				// This means that while the Log input is running and the file
				// has been active at any moment during the Filebeat's execution
				// the TTL is never going to be -2 during the shutdown.
				//
				// So, if TTL == -2, then in the previous run of Filebeat, there
				// was no Log input using this state, which likely means, it is
				// a state that has already been migrated to Filestream.
				//
				// The worst case that can happen is that we re-ingest the file
				// once, which is still better than copying an old state with
				// an incorrect offset every time Filebeat starts.
				if st.TTL == -2 {
					return true, nil
				}
				st.key = key
				fromLogInput[key] = st
			}

			return true, nil
		})
	}

	// Lock the ephemeral store so we can migrate the states in one go
	s.store.ephemeralStore.mu.Lock()
	defer s.store.ephemeralStore.mu.Unlock()

	// Migrate all states from the Filestream input
	for k := range fromFilestreamInput {
		res := s.store.ephemeralStore.unsafeFind(k, false)
		if res == nil {
			// The resource does not exist or has been deleted.
			// This should never happen, but better safe than sorry
			continue
		}

		if !res.lock.TryLock() {
			res.Release()
			s.store.log.Infof("cannot lock '%s', will not migrate its state", k)
			continue
		}

		newKey, updatedMeta := fn(res)
		if len(newKey) > 0 {
			// If the new key already exists in the store, do nothing.
			// Unlock the resource and return
			if res := s.store.ephemeralStore.unsafeFind(newKey, false); res != nil {
				res.Release()
				res.lock.Unlock()
				continue
			}

			r := res.copyWithNewKey(newKey)
			r.cursorMeta = updatedMeta
			r.stored = false
			// writeState only writes to the log file (disk)
			// the write is synchronous
			s.store.writeState(r)

			// Add the new resource to the ephemeralStore so the rest of the
			// codebase can have access to the new value
			s.store.ephemeralStore.table[newKey] = r

			// Remove the old key from the store aka delete. This is also
			// synchronously written to the disk.
			// We cannot use store.remove because it will
			// acquire the same lock we hold, causing a deadlock.
			// See store.remove for details.
			// Fully remove the old resource from all stores.
			//  - 1. Update the TTL, which soft-deletes it.
			//  - 2. Remove the resource from the in-memory store
			//  - 3. Finally, synchronously remove it from the disk store.
			s.store.UpdateTTL(res, 0)
			delete(s.store.ephemeralStore.table, res.key)
			_ = s.store.persistentStore.Remove(res.key)
			s.store.log.Infof("migrated entry in registry from '%s' to '%s'. Cursor: %v", k, newKey, r.cursor)
		}

		res.Release()
		res.lock.Unlock()
	}

	// Migrate all states from the Log input
	for k, v := range fromLogInput {
		newKey, updatedMeta := fn(v)

		// Find or create a resource. It should always create a new one.
		res := s.store.ephemeralStore.unsafeFind(newKey, true)
		res.cursorMeta = updatedMeta
		// Convert the offset to the correct type
		res.cursor = struct {
			Offset int64 `json:"offset" struct:"offset"`
		}{
			Offset: v.Offset,
		}

		// Write to disk
		s.store.writeState(res)

		// Update in-memory store
		s.store.ephemeralStore.table[newKey] = res

		// "remove" from the disk store.
		// It will add a remove entry in the log file for this key, however
		// the Registrar used by the Log input will write to disk all states
		// it read when Filebeat was starting, thus "overriding" this delete.
		// We keep it here because when we remove the Log input we will ensure
		// the entry is actually remove from the disk store.
		_ = s.store.persistentStore.Remove(k)
		res.Release()
		s.store.log.Infof("migrated entry in registry from '%s' to '%s'. Cursor: %v", k, newKey, res.cursor)
	}
}

type logInputState struct {
	ID     string        `json:"id"`
	Offset int64         `json:"offset"`
	TTL    time.Duration `json:"ttl" struct:"ttl"`
	key    string        `json:"-"`

	// This matches the filestream.fileMeta struct
	// and are used by UnpackCursorMeta
	Source         string `json:"source" struct:"source"`
	IdentifierName string `json:"identifier_name" struct:"identifier_name"`
}

func logInputStateFromMapM(m mapstr.M) (logInputState, error) {
	state := logInputState{}

	// typeconf.Convert kept failing with an "unsupported" error because
	// FileStateOS was present, we don't need it, so just delete it.
	m.Delete("FileStateOS")
	if err := typeconv.Convert(&state, m); err != nil {
		return logInputState{}, fmt.Errorf("cannot convert Log input state: %w", err)
	}

	return state, nil
}

// UnpackCursorMeta unpacks the cursor metadata's into the provided struct. TBD
func (l logInputState) UnpackCursorMeta(to any) error {
	return typeconv.Convert(to, l)
}

// Key returns the resource's key
func (l logInputState) Key() string {
	return l.key
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
	resource.Release()
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
	typeconv.Convert(&r.cursor, cur) //nolint:errcheck // not changing behaviour on this commit

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
	resource.Release()
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

	if resource.unsafeIsDeleted() {
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

	return s.unsafeFind(key, create)
}

// unsafeFind DOES NOT LOCK THE STORE!!! Only call unsafeFind if you're
// currently holding the lock from states.mu.
//
// unsafeFind returns the resource for a given key. If the key is unknown and
// create is set to false nil will be returned.
// The resource returned by unsafeFind is marked as active. (*resource).Release
// must be called to mark the resource as inactive again.
func (s *states) unsafeFind(key string, create bool) *resource {
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
	// -1 means this resource will not be cleaned up due to a timeout.
	// The zero-value for internalState.TTL means this resource is
	// soft-deleted.
	resource.internalState.TTL = -1
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

// isDeleted locks stateMutex then checks whether [resource] is deleted.
func (r *resource) isDeleted() bool {
	r.stateMutex.Lock()
	defer r.stateMutex.Unlock()
	return r.unsafeIsDeleted()
}

// unsafeIsDeleted DOES NOT LOCK THE RESOURCE!!!
// Only call unsafeIsDeleted if you're currently holding the
// lock from resource.stateMutex
func (r *resource) unsafeIsDeleted() bool {
	return !r.internalState.Updated.IsZero() && r.internalState.TTL == 0
}

// Retain is used to indicate that 'resource' gets an additional 'owner'.
// Owners of an resource can be active inputs or pending update operations
// not yet written to disk.
func (r *resource) Retain() { r.pending.Add(1) }

// Release reduced the owner ship counter of the resource.
func (r *resource) Release() { r.pending.Add(^uint64(0)) }

// UpdatesReleaseN is used to release ownership of N pending update operations.
func (r *resource) UpdatesReleaseN(n uint) {
	r.pending.Add(^uint64(n - 1))
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

// UnpackCursorMeta unpacks the cursor metadata's into the provided struct.
func (r *resource) UnpackCursorMeta(to interface{}) error {
	return typeconv.Convert(to, r.cursorMeta)
}

// Key returns the resource's key
func (r *resource) Key() string {
	return r.key
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
	// dst.lock should not be overwritten here because it's supposed to be locked
	// before this function call and it's important to preserve the previous value.
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
//
//nolint:errcheck // not changing behaviour on this commit
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
		if !strings.HasPrefix(key, keyPrefix) {
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
