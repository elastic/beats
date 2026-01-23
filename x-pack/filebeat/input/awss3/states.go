// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"container/heap"
	"fmt"
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
// - States are maintained in a min-heap ordered by lexicographical key
// - The least/smallest key state is used as startAfterKey for S3 listing
// - State ID includes a lexicographical suffix for isolation
type lexicographicalStateRegistry struct {
	baseStateRegistry

	// Min-heap for efficient access to the least key
	heap *stateHeap

	// Maximum number of states to keep
	capacity int
}

// stateHeap implements heap.Interface for states ordered by lexicographical key.
// It also maintains an index map for O(1) lookup.
type stateHeap struct {
	items []*state
	index map[string]int
}

func newStateHeap() *stateHeap {
	return &stateHeap{
		items: make([]*state, 0),
		index: make(map[string]int),
	}
}

func (h *stateHeap) Len() int { return len(h.items) }

func (h *stateHeap) Less(i, j int) bool {
	return h.items[i].Key < h.items[j].Key
}

// Swap swaps the items and updates the index map.
func (h *stateHeap) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.index[h.items[i].IDWithLexicographicalOrdering()] = i
	h.index[h.items[j].IDWithLexicographicalOrdering()] = j
}

func (h *stateHeap) Push(x any) {
	st := x.(*state)
	h.index[st.IDWithLexicographicalOrdering()] = len(h.items)
	h.items = append(h.items, st)
}

func (h *stateHeap) Pop() any {
	old := h.items
	n := len(old)
	st := old[n-1]
	old[n-1] = nil // avoid memory leak
	h.items = old[0 : n-1]
	delete(h.index, st.IDWithLexicographicalOrdering())
	return st
}

// Contains returns true if the heap contains a state with the given ID.
func (h *stateHeap) Contains(id string) bool {
	_, ok := h.index[id]
	return ok
}

// Remove removes a state by ID and returns it, or nil if not found.
func (h *stateHeap) Remove(id string) *state {
	idx, ok := h.index[id]
	if !ok {
		return nil
	}
	return heap.Remove(h, idx).(*state)
}

func (h *stateHeap) Peek() *state {
	if len(h.items) == 0 {
		return nil
	}
	return h.items[0]
}

// newLexicographicalStateRegistry creates a new lexicographical state registry.
func newLexicographicalStateRegistry(log *logp.Logger, store *statestore.Store, keyPrefix string, capacity int) (*lexicographicalStateRegistry, error) {
	stateTable, err := loadS3StatesFromRegistry(log, store, keyPrefix, true)
	if err != nil {
		return nil, fmt.Errorf("loading S3 input state: %w", err)
	}

	h := newStateHeap()

	r := &lexicographicalStateRegistry{
		baseStateRegistry: baseStateRegistry{
			store:     store,
			states:    stateTable,
			keyPrefix: keyPrefix,
		},
		heap:     h,
		capacity: capacity,
	}

	// Build heap from loaded states and trim to capacity
	if len(stateTable) > 0 {
		r.initHeapFromStates(log)
	}

	return r, nil
}

// initHeapFromStates builds the heap from the states map and trims to capacity.
func (r *lexicographicalStateRegistry) initHeapFromStates(log *logp.Logger) {
	r.storeLock.Lock()
	defer r.storeLock.Unlock()
	r.statesLock.Lock()
	defer r.statesLock.Unlock()

	for _, st := range r.states {
		r.heap.items = append(r.heap.items, st)
		r.heap.index[st.IDWithLexicographicalOrdering()] = len(r.heap.items) - 1
	}
	heap.Init(r.heap)

	// Trim to capacity
	for r.heap.Len() > r.capacity {
		smallestState := heap.Pop(r.heap).(*state)
		id := smallestState.IDWithLexicographicalOrdering()
		delete(r.states, id)
		if err := r.removeFromStore(id); err != nil && log != nil {
			log.Warnf("failed to evict least state from store: %v", err)
		}
	}
}

func (r *lexicographicalStateRegistry) AddState(st state) error {
	if err := r.validateKeyPrefix(st.Key); err != nil {
		return err
	}

	id := st.IDWithLexicographicalOrdering()
	var evictedID string
	var shouldPersist bool

	// Update in-memory state
	func() {
		r.statesLock.Lock()
		defer r.statesLock.Unlock()

		if r.heap.Len() >= r.capacity {
			// Only add if the new key is larger than the current minimum.
			// This ensures we keep the N largest keys.
			minState := r.heap.Peek()
			if minState != nil && st.Key <= minState.Key {
				return
			}
			// Evict the smallest key
			evicted := heap.Pop(r.heap).(*state)
			evictedID = evicted.IDWithLexicographicalOrdering()
			delete(r.states, evictedID)
		}

		// Add new state to states map and heap
		r.states[id] = &st
		heap.Push(r.heap, &st)
		shouldPersist = true
	}()

	if !shouldPersist {
		return nil
	}

	// Persist changes to store
	r.storeLock.Lock()
	defer r.storeLock.Unlock()

	if evictedID != "" {
		if err := r.removeFromStore(evictedID); err != nil {
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

	// If removing all states, preserve at least one (the greatest key for startAfterKey)
	if len(r.states)-len(idsToRemove) < 1 && len(idsToRemove) > 0 {
		// Find the greatest key to preserve
		var greatestID string
		for _, id := range idsToRemove {
			if greatestID == "" || id > greatestID {
				greatestID = id
			}
		}
		// Remove greatestID from idsToRemove
		filtered := make([]string, 0, len(idsToRemove)-1)
		for _, id := range idsToRemove {
			if id != greatestID {
				filtered = append(filtered, id)
			}
		}
		idsToRemove = filtered
	}

	// Remove the states from heap, map, and store
	for _, id := range idsToRemove {
		r.heap.Remove(id)
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
	return r.heap.Peek()
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
