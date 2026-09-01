// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/logp"
)

const awsS3ObjectStatePrefix = "filebeat::aws-s3::state::"

<<<<<<< HEAD
// states handles list of s3 object state. One must use newStates to instantiate a
// file states registry. Using the zero-value is not safe.
type states struct {
=======
// stateRegistry defines the interface for managing S3 object states.
// This allows different implementations for normal mode vs lexicographical ordering mode.
type stateRegistry interface {
	// IsProcessed returns true if the object with the given ID has been processed.
	IsProcessed(id string) bool

	// AddState adds or updates a state in the registry.
	AddState(st state) error

	// CleanUp removes states that are not in the provided knownIDs list.
	CleanUp(knownIDs []string) error

	// GetStartAfterKey returns the key to use for S3 ListObjects StartAfter parameter.
	// For lexicographical mode, this returns the persisted tail key.
	// Returns empty string if no tail exists.
	GetStartAfterKey() string

	// MarkObjectInFlight marks an object key as currently being processed.
	// In lexicographical mode, this updates the in-memory tail tracking and
	// persists the new tail if it's smaller than the current tail.
	MarkObjectInFlight(key string) error

	// UnmarkObjectInFlight removes an object key from in-flight tracking.
	// Called when processing fails or is skipped (not when completing successfully).
	// Updates and persists the tail if needed.
	UnmarkObjectInFlight(key string) error

	// Close closes the underlying store.
	Close()
}

// newStateRegistry creates the appropriate state registry based on configuration.
// bucket is the name of the bucket the input polls, used to scope the loaded
// states to this input. The persistent store is shared by all aws-s3 inputs of
// the process, so without this scoping an input would load (and later clean up)
// states belonging to other inputs. An empty bucket disables the scoping.
func newStateRegistry(log *logp.Logger, stateStore statestore.States, bucket string, keyPrefix string, lexicographicalOrdering bool, lexicographicalLookbackKeys int) (stateRegistry, error) {
	// When lexicographical ordering is enabled, pass the input type to allow
	// ES state store routing for agentless deployments
	storeKey := ""
	if lexicographicalOrdering {
		storeKey = inputName
	}
	store, err := stateStore.StoreFor(storeKey)
	if err != nil {
		return nil, fmt.Errorf("can't access persistent store: %w", err)
	}

	if lexicographicalOrdering {
		return newLexicographicalStateRegistry(log, store, bucket, keyPrefix, lexicographicalLookbackKeys)
	}
	return newNormalStateRegistry(log, store, bucket, keyPrefix)
}

// baseStateRegistry contains shared functionality between registry implementations.
type baseStateRegistry struct {
>>>>>>> d5abb60 ([aws-s3]  Scope polling state registry to the input's bucket (#52728))
	// Completed S3 object states, indexed by state ID.
	// statesLock must be held to access states.
	states     map[string]*state
	statesLock sync.Mutex

	// The store used to persist state changes to the registry.
	// storeLock must be held to access store.
	store     *statestore.Store
	storeLock sync.Mutex

	// Accepted prefixes of state keys of this registry
	keyPrefix string
}

// newStates generates a new states registry.
func newStates(log *logp.Logger, stateStore statestore.States, listPrefix string) (*states, error) {
	store, err := stateStore.StoreFor("")
	if err != nil {
		return nil, fmt.Errorf("can't access persistent store: %w", err)
	}

<<<<<<< HEAD
	stateTable, err := loadS3StatesFromRegistry(log, store, listPrefix)
=======
// persistState saves the state to the persistent store.
// Caller must hold storeLock.
func (b *baseStateRegistry) persistState(id string, st state) error {
	return b.store.Set(getStoreKey(id), st)
}

// removeFromStore removes the state from the persistent store.
// Caller must hold storeLock.
func (b *baseStateRegistry) removeFromStore(id string) error {
	return b.store.Remove(getStoreKey(id))
}

// normalStateRegistry implements the default (non-lexicographical) state management.
// In this mode:
// - States are stored indefinitely (no capacity limit)
// - State ID includes etag and last modified for change detection
// - No ordering is maintained
type normalStateRegistry struct {
	baseStateRegistry
}

// newNormalStateRegistry creates a new normal state registry.
func newNormalStateRegistry(log *logp.Logger, store *statestore.Store, bucket string, keyPrefix string) (*normalStateRegistry, error) {
	stateTable, err := loadS3StatesFromRegistry(log, store, bucket, keyPrefix, false)
>>>>>>> d5abb60 ([aws-s3]  Scope polling state registry to the input's bucket (#52728))
	if err != nil {
		return nil, fmt.Errorf("loading S3 input state: %w", err)
	}

	return &states{
		store:     store,
		states:    stateTable,
		keyPrefix: listPrefix,
	}, nil
}

func (s *states) IsProcessed(state state) bool {
	s.statesLock.Lock()
	defer s.statesLock.Unlock()
	// Our in-memory table only stores completed objects
	_, ok := s.states[state.ID()]
	return ok
}

func (s *states) AddState(state state) error {
	if !strings.HasPrefix(state.Key, s.keyPrefix) {
		// Note - This failure should not happen since we create a dedicated state instance per input.
		// Yet, this is here to avoid any wiring errors within the component.
		return fmt.Errorf("expected prefix %s in key %s, skipping state registering", s.keyPrefix, state.Key)
	}

	id := state.ID()
	// Update in-memory copy
	s.statesLock.Lock()
	s.states[id] = &state
	s.statesLock.Unlock()

	// Persist to the registry
	s.storeLock.Lock()
	defer s.storeLock.Unlock()
	if err := s.store.Set(getStoreKey(id), state); err != nil {
		return err
	}
	return nil
}

// CleanUp performs state and store cleanup based on provided knownIDs.
// knownIDs must contain valid currently tracked state IDs that must be known by this state registry.
// State and underlying storage will be cleaned if ID is no longer present in knownIDs set.
func (s *states) CleanUp(knownIDs []string) error {
	knownIDHashSet := map[string]struct{}{}
	for _, id := range knownIDs {
		knownIDHashSet[id] = struct{}{}
	}

	s.storeLock.Lock()
	defer s.storeLock.Unlock()
	s.statesLock.Lock()
	defer s.statesLock.Unlock()

	for id := range s.states {
		if _, contains := knownIDHashSet[id]; !contains {
<<<<<<< HEAD
			// remove from sate & store as ID is no longer seen in known ID set
			delete(s.states, id)
			err := s.store.Remove(getStoreKey(id))
			if err != nil {
				return fmt.Errorf("error while removing the state for ID %s: %w", id, err)
=======
			idsToRemove = append(idsToRemove, id)
		}
	}

	// Remove the states
	for _, id := range idsToRemove {
		delete(r.states, id)
		if err := r.removeFromStore(id); err != nil {
			return fmt.Errorf("error while removing the state for ID %s: %w", id, err)
		}
	}

	return nil
}

func (r *normalStateRegistry) GetStartAfterKey() string {
	// Normal mode lists from beginning each poll cycle
	return ""
}

func (r *normalStateRegistry) MarkObjectInFlight(key string) error {
	// Normal mode doesn't use tail tracking
	return nil
}

func (r *normalStateRegistry) UnmarkObjectInFlight(key string) error {
	// Normal mode doesn't use tail tracking
	return nil
}

// lexicographicalStateRegistry implements lexicographical ordering state management.
// In this mode:
// - States are limited to a configurable capacity (lookbackKeys)
// - States are maintained in a min-heap ordered by lexicographical key
// - A "tail" (smallest key among in-flight + completed) is persisted for crash recovery
// - State ID includes a lexicographical suffix for isolation
type lexicographicalStateRegistry struct {
	baseStateRegistry

	// Min-heap for efficient access to the least key among completed states
	heap *stateHeap

	// Maximum number of states to keep
	capacity int

	// inFlight tracks keys currently being processed (dispatched but not completed).
	// This is used to compute the tail = min(inFlight keys, completed keys).
	inFlight map[string]struct{}
	// persistedTail is the tail key stored in the persistent store.
	// This survives crashes and is used as startAfterKey on restart.
	persistedTail string

	// inFlightLock protects access to inFlight map and persistedTail
	inFlightLock sync.Mutex
}

// newLexicographicalStateRegistry creates a new lexicographical state registry.
func newLexicographicalStateRegistry(log *logp.Logger, store *statestore.Store, bucket string, keyPrefix string, capacity int) (*lexicographicalStateRegistry, error) {
	stateTable, err := loadS3StatesFromRegistry(log, store, bucket, keyPrefix, true)
	if err != nil {
		return nil, fmt.Errorf("loading S3 input state: %w", err)
	}

	var persisted struct {
		Tail string `json:"tail"`
	}
	if err := store.Get(awsS3TailKey, &persisted); err != nil {
		// Key doesn't exist or can't be decoded - start fresh
		if log != nil {
			log.Infof("No valid persisted tail found (key=%s), starting fresh: %v", awsS3TailKey, err)
		}
	}
	persistedTail := persisted.Tail

	h := newStateHeap()

	r := &lexicographicalStateRegistry{
		baseStateRegistry: baseStateRegistry{
			store:     store,
			states:    stateTable,
			keyPrefix: keyPrefix,
		},
		heap:          h,
		capacity:      capacity,
		inFlight:      make(map[string]struct{}),
		persistedTail: persistedTail,
	}

	// Build heap from loaded states and trim to capacity
	r.initHeapFromStates(log)

	// If no persisted tail but we have states, compute initial tail from heap minimum
	if r.persistedTail == "" && r.heap.Len() > 0 {
		if minState := r.heap.peek(); minState != nil {
			r.persistedTail = minState.Key
			_ = store.Remove(awsS3TailKey)
			if err := store.Set(awsS3TailKey, struct {
				Tail string `json:"tail"`
			}{r.persistedTail}); err != nil {
				return nil, fmt.Errorf("failed to persist initial tail key to store (key=%q): %w", r.persistedTail, err)
>>>>>>> d5abb60 ([aws-s3]  Scope polling state registry to the input's bucket (#52728))
			}
		}
	}

	return nil
}

func (s *states) Close() {
	s.storeLock.Lock()
	s.store.Close()
	s.storeLock.Unlock()
}

// getStoreKey is a helper to generate the key used by underlying persistent storage
func getStoreKey(stateID string) string {
	return awsS3ObjectStatePrefix + stateID
}

// loadS3StatesFromRegistry loads a copy of the registry states.
<<<<<<< HEAD
// If prefix is set, entries will match the provided prefix(including empty prefix)
func loadS3StatesFromRegistry(log *logp.Logger, store *statestore.Store, prefix string) (map[string]*state, error) {
=======
// Only entries belonging to the given bucket and matching the given key prefix
// are loaded. The store is shared by all aws-s3 inputs of the process, and an
// input must only ever see its own states: CleanUp removes every store entry
// that is missing from the input's bucket listing, so loading another input's
// states here would delete them from the shared store (and each input would
// also hold every other input's states in memory).
// Passing an empty bucket argument disables the bucket filter; the pollers
// always pass their bucket name.
func loadS3StatesFromRegistry(log *logp.Logger, store *statestore.Store, bucket string, prefix string, lexicographicalOrdering bool) (map[string]*state, error) {
>>>>>>> d5abb60 ([aws-s3]  Scope polling state registry to the input's bucket (#52728))
	stateTable := map[string]*state{}
	err := store.Each(func(key string, dec statestore.ValueDecoder) (bool, error) {
		if !strings.HasPrefix(key, awsS3ObjectStatePrefix) {
			return true, nil
		}

		// try to decode. Ignore faulty/incompatible values.
		var st state
		if err := dec.Decode(&st); err != nil {
			// Skip this key but continue iteration
			if log != nil {
				log.Warnf("invalid S3 state loading object key %v", key)
			}
			return true, nil
		}
		if !st.Stored && !st.Failed {
			// This is from an older version where state could be stored in the
			// registry even if the object wasn't processed, or if it encountered
			// ephemeral download errors. We don't add these to the in-memory cache,
			// so if we see them during a bucket scan we will still retry them.
			return true, nil
		}

		// skip entries that belong to another input's bucket
		if bucket != "" && st.Bucket != bucket {
			return true, nil
		}

		// filter based on prefix and add entry to local copy
		if strings.HasPrefix(st.Key, prefix) {
			stateTable[st.ID()] = &st
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return stateTable, nil
}
