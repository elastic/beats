// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/logp"
)

const awsS3ObjectStatePrefix = "filebeat::aws-s3::state::"

// stateRegistry defines the interface for managing S3 object states.
// This allows different implementations for normal mode vs lexicographical ordering mode.
type stateRegistry interface {
	// IsProcessed returns true if the object with the given ID has been processed.
	IsProcessed(id string) bool

	// AddState adds or updates a state in the registry.
	AddState(st state) error

	// CleanUp removes states that are not in the provided knownIDs list.
	CleanUp(knownIDs []string) error

	// GetLeastState returns the least/smallest key state in the registry (used for startAfterKey).
	// Returns nil if no states exist.
	GetLeastState() *state

	// Close closes the underlying store.
	Close()
}

// baseStateRegistry contains shared functionality between registry implementations.
type baseStateRegistry struct {
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

func (b *baseStateRegistry) IsProcessed(id string) bool {
	b.statesLock.Lock()
	defer b.statesLock.Unlock()
	_, ok := b.states[id]
	return ok
}

func (b *baseStateRegistry) Close() {
	b.storeLock.Lock()
	b.store.Close()
	b.storeLock.Unlock()
}

func (b *baseStateRegistry) validateKeyPrefix(key string) error {
	if !strings.HasPrefix(key, b.keyPrefix) {
		return fmt.Errorf("expected prefix %s in key %s, skipping state registering", b.keyPrefix, key)
	}
	return nil
}

func (b *baseStateRegistry) persistState(id string, st state) error {
	return b.store.Set(getStoreKey(id), st)
}

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
func newNormalStateRegistry(log *logp.Logger, store *statestore.Store, keyPrefix string) (*normalStateRegistry, error) {
	stateTable, err := loadS3StatesFromRegistry(log, store, keyPrefix, false)
	if err != nil {
		return nil, fmt.Errorf("loading S3 input state: %w", err)
	}

	return &normalStateRegistry{
		baseStateRegistry: baseStateRegistry{
			store:     store,
			states:    stateTable,
			keyPrefix: keyPrefix,
		},
	}, nil
}

func (r *normalStateRegistry) AddState(st state) error {
	if err := r.validateKeyPrefix(st.Key); err != nil {
		return err
	}

	id := st.ID()

	// Update in-memory copy
	r.statesLock.Lock()
	r.states[id] = &st
	r.statesLock.Unlock()

	// Persist to the registry
	r.storeLock.Lock()
	defer r.storeLock.Unlock()
	return r.persistState(id, st)
}

func (r *normalStateRegistry) CleanUp(knownIDs []string) error {
	knownIDHashSet := make(map[string]struct{}, len(knownIDs))
	for _, id := range knownIDs {
		knownIDHashSet[id] = struct{}{}
	}

	r.storeLock.Lock()
	defer r.storeLock.Unlock()
	r.statesLock.Lock()
	defer r.statesLock.Unlock()

	// Collect IDs to remove
	var idsToRemove []string
	for id := range r.states {
		if _, contains := knownIDHashSet[id]; !contains {
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

func (r *normalStateRegistry) GetLeastState() *state {
	// Normal mode doesn't track ordering, return nil
	return nil
}

// lexicographicalStateRegistry implements lexicographical ordering state management.
// In this mode:
// - States are limited to a configurable capacity (lookbackKeys)
// - States are maintained in a doubly linked list ordered by key
// - The least/smallest key state (head) is used as startAfterKey for S3 listing
// - State ID includes a lexicographical suffix for isolation
type lexicographicalStateRegistry struct {
	baseStateRegistry

	// Linked list pointers for ordering
	head *state // Least/smallest key state (lexicographically smallest)
	tail *state // Greatest/largest key state (lexicographically largest)

	// Maximum number of states to keep
	capacity int
}

// newLexicographicalStateRegistry creates a new lexicographical state registry.
func newLexicographicalStateRegistry(log *logp.Logger, store *statestore.Store, keyPrefix string, capacity int) (*lexicographicalStateRegistry, error) {
	stateTable, err := loadS3StatesFromRegistry(log, store, keyPrefix, true)
	if err != nil {
		return nil, fmt.Errorf("loading S3 input state: %w", err)
	}

	r := &lexicographicalStateRegistry{
		baseStateRegistry: baseStateRegistry{
			store:     store,
			states:    stateTable,
			keyPrefix: keyPrefix,
		},
		capacity: capacity,
	}

	// Trim loaded states to capacity and build linked list
	if len(stateTable) > 0 {
		r.trimAndBuildLinkedList(log)
	}

	return r, nil
}

func (r *lexicographicalStateRegistry) AddState(st state) error {
	if err := r.validateKeyPrefix(st.Key); err != nil {
		return err
	}

	id := st.IDWithLexicographicalOrdering()
	var leastState *state

	// Update in-memory state
	func() {
		r.statesLock.Lock()
		defer r.statesLock.Unlock()

		// Check capacity and evict if necessary
		if len(r.states) >= r.capacity {
			leastState = r.findLexicographicallyLeast()
			if leastState != nil {
				delete(r.states, leastState.IDWithLexicographicalOrdering())
				r.remove(leastState)
			}
		}
		// Add new state to the back (newest position)
		r.states[id] = &st
		r.addToBack(&st)
	}()

	// Persist changes to store
	r.storeLock.Lock()
	defer r.storeLock.Unlock()

	if leastState != nil {
		if err := r.removeFromStore(leastState.IDWithLexicographicalOrdering()); err != nil {
			return fmt.Errorf("error while removing evicted state: %w", err)
		}
	}

	return r.persistState(id, st)
}

func (r *lexicographicalStateRegistry) CleanUp(knownIDs []string) error {
	knownIDHashSet := make(map[string]struct{}, len(knownIDs))
	for _, id := range knownIDs {
		knownIDHashSet[id] = struct{}{}
	}

	r.storeLock.Lock()
	defer r.storeLock.Unlock()
	r.statesLock.Lock()
	defer r.statesLock.Unlock()

	// Collect IDs to remove
	var idsToRemove []string
	for id := range r.states {
		if _, contains := knownIDHashSet[id]; !contains {
			idsToRemove = append(idsToRemove, id)
		}
	}

	if len(idsToRemove) == 0 {
		return nil
	}

	// Preserve the newest state for startAfterKey
	slices.Sort(idsToRemove)
	newestIdx := len(idsToRemove) - 1
	if len(r.states)-len(idsToRemove) < 1 {
		// Would remove all states, so keep the newest one
		idsToRemove = idsToRemove[:newestIdx]
	}

	// Remove the states
	for _, id := range idsToRemove {
		st := r.states[id]
		r.remove(st) // Remove from linked list
		delete(r.states, id)
		if err := r.removeFromStore(id); err != nil {
			return fmt.Errorf("error while removing the state for ID %s: %w", id, err)
		}
	}

	return nil
}

func (r *lexicographicalStateRegistry) GetLeastState() *state {
	r.statesLock.Lock()
	defer r.statesLock.Unlock()
	return r.head
}

// SortStates rebuilds the linked list in lexicographical order.
// This should be called before each polling run.
func (r *lexicographicalStateRegistry) SortStates(log *logp.Logger) {
	r.statesLock.Lock()
	defer r.statesLock.Unlock()

	if len(r.states) == 0 {
		return
	}

	r.trimAndBuildLinkedList(log)
	if r.head != nil && r.tail != nil && log != nil {
		log.Debugw("Sorted states by lexicographical ordering.", "state_count", len(r.states), "oldest_state", r.head.IDWithLexicographicalOrdering(), "newest_state", r.tail.IDWithLexicographicalOrdering())
	}
}

// addToBack adds a state to the tail (newest position)
func (r *lexicographicalStateRegistry) addToBack(st *state) {
	st.prev = r.tail
	st.next = nil
	if r.tail != nil {
		r.tail.next = st
	}
	r.tail = st
	if r.head == nil {
		r.head = st
	}
}

// remove removes a state from the linked list
func (r *lexicographicalStateRegistry) remove(st *state) {
	if st.prev != nil {
		st.prev.next = st.next
	} else {
		r.head = st.next // removing head
	}
	if st.next != nil {
		st.next.prev = st.prev
	} else {
		r.tail = st.prev // removing tail
	}
	st.prev = nil
	st.next = nil
}

// findLexicographicallyLeast finds the state with the smallest key.
// Assumes statesLock is held by the caller.
func (r *lexicographicalStateRegistry) findLexicographicallyLeast() *state {
	if len(r.states) == 0 {
		return nil
	}
	var leastState *state
	for _, st := range r.states {
		if leastState == nil || st.IDWithLexicographicalOrdering() < leastState.IDWithLexicographicalOrdering() {
			leastState = st
		}
	}
	return leastState
}

// trimAndBuildLinkedList trims states to capacity and rebuilds the linked list.
// Assumes statesLock is held by the caller.
func (r *lexicographicalStateRegistry) trimAndBuildLinkedList(log *logp.Logger) {
	sortedKeys := slices.Sorted(maps.Keys(r.states))

	// If over capacity, trim to capacity and remove least states
	if len(sortedKeys) > r.capacity {
		keysToRemove := sortedKeys[:len(sortedKeys)-r.capacity]
		r.storeLock.Lock()
		for _, key := range keysToRemove {
			delete(r.states, key)
			if err := r.removeFromStore(key); err != nil && log != nil {
				log.Warnf("failed to remove least state from store: %v", err)
			}
		}
		r.storeLock.Unlock()
		sortedKeys = sortedKeys[len(sortedKeys)-r.capacity:]
	}

	// Build the doubly linked list in sorted order
	r.buildLinkedListFromKeys(sortedKeys)
}

// buildLinkedListFromKeys rebuilds the doubly linked list from sorted keys.
func (r *lexicographicalStateRegistry) buildLinkedListFromKeys(sortedKeys []string) {
	r.head = nil
	r.tail = nil
	var prev *state
	for i, key := range sortedKeys {
		st := r.states[key]
		st.prev = prev
		st.next = nil
		if prev != nil {
			prev.next = st
		}
		if i == 0 {
			r.head = st
		}
		prev = st
	}
	r.tail = prev
}

// newStateRegistry creates the appropriate state registry based on configuration.
func newStateRegistry(log *logp.Logger, stateStore statestore.States, keyPrefix string, lexicographicalOrdering bool, lexicographicalLookbackKeys int) (stateRegistry, error) {
	store, err := stateStore.StoreFor("")
	if err != nil {
		return nil, fmt.Errorf("can't access persistent store: %w", err)
	}

	if lexicographicalOrdering {
		return newLexicographicalStateRegistry(log, store, keyPrefix, lexicographicalLookbackKeys)
	}
	return newNormalStateRegistry(log, store, keyPrefix)
}

// getStoreKey generates the key used by underlying persistent storage
func getStoreKey(stateID string) string {
	return awsS3ObjectStatePrefix + stateID
}

// loadS3StatesFromRegistry loads a copy of the registry states.
// If prefix is set, entries will match the provided prefix(including empty prefix)
func loadS3StatesFromRegistry(log *logp.Logger, store *statestore.Store, prefix string, lexicographicalOrdering bool) (map[string]*state, error) {
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

		// filter based on prefix and add entry to local copy
		if strings.HasPrefix(st.Key, prefix) {
			if lexicographicalOrdering {
				stateTable[st.IDWithLexicographicalOrdering()] = &st
			} else {
				stateTable[st.ID()] = &st
			}
		}
		return true, nil
	})
	return stateTable, err
}
