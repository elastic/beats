// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"container/heap"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/logp"
)

const awsS3ObjectStatePrefix = "filebeat::aws-s3::state::"
const awsS3TailKey = "filebeat::aws-s3::tail"

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

// baseStateRegistry contains shared functionality between registry implementations.
type baseStateRegistry struct {
	// Completed S3 object states, indexed by state ID.
	states map[string]*state
	// statesLock protects access to states map
	statesLock sync.Mutex

	// The store used to persist state changes to the registry.
	store *statestore.Store
	// storeLock protects access to store.
	// Callers of unexported stateRegistry methods that read or write store must hold this lock.
	storeLock sync.Mutex

	// Accepted prefixes of state keys of this registry
	keyPrefix string
}

func (b *baseStateRegistry) IsProcessed(id string) bool {
	b.statesLock.Lock()
	_, ok := b.states[id]
	b.statesLock.Unlock()
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
	// inFlightLock protects access to inFlight map
	inFlightLock sync.Mutex

	// persistedTail is the tail key stored in the persistent store.
	// This survives crashes and is used as startAfterKey on restart.
	persistedTail string
}

// newLexicographicalStateRegistry creates a new lexicographical state registry.
func newLexicographicalStateRegistry(log *logp.Logger, store *statestore.Store, keyPrefix string, capacity int) (*lexicographicalStateRegistry, error) {
	stateTable, err := loadS3StatesFromRegistry(log, store, keyPrefix, true)
	if err != nil {
		return nil, fmt.Errorf("loading S3 input state: %w", err)
	}

	var persistedTail string
	if err := store.Get(awsS3TailKey, &persistedTail); err != nil {
		// Key doesn't exist or can't be decoded - start fresh
		persistedTail = ""
	}

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
	if len(stateTable) > 0 {
		r.initHeapFromStates(log)
	}

	// If no persisted tail but we have states, compute initial tail from heap minimum
	if r.persistedTail == "" && r.heap.Len() > 0 {
		if minState := r.heap.peek(); minState != nil {
			r.persistedTail = minState.Key
			if err := store.Set(awsS3TailKey, r.persistedTail); err != nil {
				return nil, fmt.Errorf("persisting initial tail: %w", err)
			}
		}
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
		smallestState := r.heap.pop()
		id := smallestState.IDWithLexicographicalOrdering()
		delete(r.states, id)
		if err := r.removeFromStore(id); err != nil && log != nil {
			log.Warnf("failed to evict least state from store: %v", err)
		}
	}
}

// AddState removes the object from in-flight tracking and adds it
// to completed states, then recomputes and persists the tail once.
func (r *lexicographicalStateRegistry) AddState(st state) error {
	if err := r.validateKeyPrefix(st.Key); err != nil {
		return err
	}

	id := st.IDWithLexicographicalOrdering()
	var evictedID string
	var shouldPersist bool

	// Update in-memory state: remove from in-flight and add to completed atomically
	func() {
		r.inFlightLock.Lock()
		defer r.inFlightLock.Unlock()
		r.statesLock.Lock()
		defer r.statesLock.Unlock()

		// Remove from in-flight
		delete(r.inFlight, st.Key)

		if r.heap.Len() >= r.capacity {
			// Only add if the new key is larger than the current minimum.
			// This ensures we keep the N largest keys.
			minState := r.heap.peek()
			if minState != nil && st.Key <= minState.Key {
				return
			}
			// Evict the smallest key
			evicted := r.heap.pop()
			evictedID = evicted.IDWithLexicographicalOrdering()
			delete(r.states, evictedID)
		}

		// Add new state to states map and heap
		r.states[id] = &st
		r.heap.push(&st)
		shouldPersist = true
	}()

	if !shouldPersist {
		return nil
	}

	// Persist state changes to store
	r.storeLock.Lock()
	if evictedID != "" {
		if err := r.removeFromStore(evictedID); err != nil {
			r.storeLock.Unlock()
			return fmt.Errorf("error while removing evicted state: %w", err)
		}
	}
	if err := r.persistState(id, st); err != nil {
		r.storeLock.Unlock()
		return err
	}
	r.storeLock.Unlock()

	// Recompute and persist tail
	r.inFlightLock.Lock()
	err := r.recomputeAndPersistTail()
	r.inFlightLock.Unlock()

	return err
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
		greatestID := slices.Max(idsToRemove)
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
		r.heap.remove(id)
		delete(r.states, id)
		if err := r.removeFromStore(id); err != nil {
			return fmt.Errorf("error while removing the state for ID %s: %w", id, err)
		}
	}

	return nil
}

func (r *lexicographicalStateRegistry) GetStartAfterKey() string {
	r.inFlightLock.Lock()
	defer r.inFlightLock.Unlock()
	return r.persistedTail
}

// MarkObjectInFlight marks an object as in-flight and updates the persisted tail
// to ensure we don't skip this object if a crash occurs during processing.
func (r *lexicographicalStateRegistry) MarkObjectInFlight(key string) error {
	r.inFlightLock.Lock()
	defer r.inFlightLock.Unlock()

	r.inFlight[key] = struct{}{}

	// Update tail if smaller than current tail
	if r.persistedTail == "" || key < r.persistedTail {
		r.persistedTail = key
		r.storeLock.Lock()
		err := r.store.Set(awsS3TailKey, key)
		r.storeLock.Unlock()
		if err != nil {
			return fmt.Errorf("failed to persist tail key: %w", err)
		}
	}

	return nil
}

func (r *lexicographicalStateRegistry) UnmarkObjectInFlight(key string) error {
	r.inFlightLock.Lock()
	defer r.inFlightLock.Unlock()

	delete(r.inFlight, key)

	return r.recomputeAndPersistTail()
}

// computeTail returns the tail key, which is the minimum(in-flight keys, completed keys).
// Called by recomputeAndPersistTail; inFlightLock must be held by the caller chain.
func (r *lexicographicalStateRegistry) computeTail() string {
	var minInFlight string
	for key := range r.inFlight {
		if minInFlight == "" || key < minInFlight {
			minInFlight = key
		}
	}

	r.statesLock.Lock()
	heapMin := r.heap.peek()
	r.statesLock.Unlock()

	var minCompleted string
	if heapMin != nil {
		minCompleted = heapMin.Key
	}

	// Return smaller of in-flight and completed keys.
	switch {
	case minInFlight == "":
		return minCompleted
	case minCompleted == "" || minInFlight < minCompleted:
		return minInFlight
	default:
		return minCompleted
	}
}

// recomputeAndPersistTail recomputes the tail from in-flight and completed states,
// and persists it if changed.
// Caller must hold inFlightLock.
func (r *lexicographicalStateRegistry) recomputeAndPersistTail() error {
	newTail := r.computeTail()

	if newTail == r.persistedTail {
		return nil
	}

	r.persistedTail = newTail
	r.storeLock.Lock()
	var err error
	if newTail == "" {
		err = r.store.Remove(awsS3TailKey)
	} else {
		err = r.store.Set(awsS3TailKey, newTail)
	}
	r.storeLock.Unlock()

	if err != nil {
		return fmt.Errorf("failed to persist tail key: %w", err)
	}
	return nil
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

// pop removes and returns the smallest state from the heap.
func (h *stateHeap) pop() *state {
	if h.Len() == 0 {
		return nil
	}
	return heap.Pop(h).(*state)
}

// push adds a state to the heap.
func (h *stateHeap) push(st *state) {
	heap.Push(h, st)
}

// remove removes a state by ID and returns it, or nil if not found.
func (h *stateHeap) remove(id string) *state {
	idx, ok := h.index[id]
	if !ok {
		return nil
	}
	return heap.Remove(h, idx).(*state)
}

// peek returns the smallest state without removing it.
func (h *stateHeap) peek() *state {
	if len(h.items) == 0 {
		return nil
	}
	return h.items[0]
}

func (h *stateHeap) Len() int { return len(h.items) }

func (h *stateHeap) Less(i, j int) bool {
	return h.items[i].Key < h.items[j].Key
}

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
