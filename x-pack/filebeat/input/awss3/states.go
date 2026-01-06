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

// states handles list of s3 object state. One must use newStates to instantiate a
// file states registry. Using the zero-value is not safe.
type states struct {
	// Completed S3 object states, indexed by state ID.
	// statesLock must be held to access states.
	states     map[string]*state
	statesLock sync.Mutex
	head       *state // In lexicographical ordering mode, this is the oldest (front) state.
	tail       *state // In lexicographical ordering mode, this is the newest (back) state.

	// The store used to persist state changes to the registry.
	// storeLock must be held to access store.
	store     *statestore.Store
	storeLock sync.Mutex

	// Accepted prefixes of state keys of this registry
	keyPrefix string
}

// newStates generates a new states registry.
func newStates(log *logp.Logger, stateStore statestore.States, listPrefix string, lexicographicalOrdering bool, lexicographicalLookbackKeys int) (*states, error) {
	store, err := stateStore.StoreFor("")
	if err != nil {
		return nil, fmt.Errorf("can't access persistent store: %w", err)
	}

	stateTable, err := loadS3StatesFromRegistry(log, store, listPrefix, lexicographicalOrdering)
	if err != nil {
		return nil, fmt.Errorf("loading S3 input state: %w", err)
	}

	s := &states{
		store:     store,
		states:    stateTable,
		keyPrefix: listPrefix,
	}

	// If lexicographical ordering is enabled, trim the loaded states to capacity
	// and build the linked list structure. Otherwise, the linked list structure is not built.
	if lexicographicalOrdering && len(stateTable) > 0 {
		s.trimAndBuildLinkedList(log, lexicographicalLookbackKeys)
	}

	return s, nil
}

func (s *states) IsProcessed(id string) bool {
	s.statesLock.Lock()
	defer s.statesLock.Unlock()
	// Our in-memory table only stores completed objects
	var ok bool
	_, ok = s.states[id]
	return ok
}

// addToBack adds a state to the tail (newest position)
func (s *states) addToBack(st *state) {
	st.prev = s.tail
	st.next = nil
	if s.tail != nil {
		s.tail.next = st
	}
	s.tail = st
	if s.head == nil {
		s.head = st
	}
}

// remove removes a state from the linked list
func (s *states) remove(st *state) {
	if st.prev != nil {
		st.prev.next = st.next
	} else {
		s.head = st.next // removing head
	}
	if st.next != nil {
		st.next.prev = st.prev
	} else {
		s.tail = st.prev // removing tail
	}
	st.prev = nil
	st.next = nil
}

// moveToBack moves an existing state to the tail (newest position)
func (s *states) moveToBack(st *state) {
	if st == s.tail {
		return // already at back
	}
	s.remove(st)
	s.addToBack(st)
}

// findLexicographicallyOldest finds the state with the smallest key lexicographically.
// This is used during eviction to ensure we always remove the oldest key, not just
// the first inserted one. Assumes statesLock is held by the caller.
func (s *states) findLexicographicallyOldest() *state {
	if len(s.states) == 0 {
		return nil
	}
	var oldest *state
	for _, st := range s.states {
		if oldest == nil || st.IDWithLexicographicalOrdering() < oldest.IDWithLexicographicalOrdering() {
			oldest = st
		}
	}
	return oldest
}

func (s *states) AddState(st state, lexicographicalOrdering bool, lexicographicalLookbackKeys int) error {
	if !strings.HasPrefix(st.Key, s.keyPrefix) {
		// Note - This failure should not happen since we create a dedicated state instance per input.
		// Yet, this is here to avoid any wiring errors within the component.
		return fmt.Errorf("expected prefix %s in key %s, skipping state registering", s.keyPrefix, st.Key)
	}

	var id string

	// Update in-memory copy
	s.statesLock.Lock()

	var oldest *state

	// Maintain a doubly linked list structure for lexicographical ordering
	// with lexicographicalLookbackKeys as capacity
	if lexicographicalOrdering {
		id = st.IDWithLexicographicalOrdering()
		// If state already exists, update it and move to back (newest position)
		if existing, exists := s.states[id]; exists {
			// Update the existing state's fields
			existing.Stored = st.Stored
			existing.Failed = st.Failed
			s.moveToBack(existing)
		} else {
			// New state: check capacity and add to back
			// If at capacity, find and remove the lexicographically oldest state
			if len(s.states) >= lexicographicalLookbackKeys {
				oldest = s.findLexicographicallyOldest()
				if oldest != nil {
					delete(s.states, oldest.IDWithLexicographicalOrdering())
					s.remove(oldest)
				}
			}
			// Add new state to the back (newest position)
			s.states[id] = &st
			s.addToBack(&st)
		}
	} else {
		id = st.ID()
		// Add new state to the in-memory copy of the states
		s.states[id] = &st
	}

	s.statesLock.Unlock()

	// Persist to the registry
	s.storeLock.Lock()
	defer s.storeLock.Unlock()
	if lexicographicalOrdering && oldest != nil {
		if err := s.store.Remove(getStoreKey(oldest.IDWithLexicographicalOrdering())); err != nil {
			return fmt.Errorf("error while removing the oldest state: %w", err)
		}
	}

	if err := s.store.Set(getStoreKey(id), st); err != nil {
		return err
	}
	return nil
}

func (s *states) GetOldestState() *state {
	return s.head
}

// CleanUp performs state and store cleanup based on provided knownIDs.
// knownIDs must contain valid currently tracked state IDs that must be known by this state registry.
// State and underlying storage will be cleaned if ID is no longer present in knownIDs set.
// When lexicographicalOrdering is enabled, at least one state (the lexicographically newest)
// is always preserved to ensure startAfterKey can be used for subsequent S3 listings.
func (s *states) CleanUp(knownIDs []string, lexicographicalOrdering bool) error {
	knownIDHashSet := map[string]struct{}{}
	for _, id := range knownIDs {
		knownIDHashSet[id] = struct{}{}
	}

	s.storeLock.Lock()
	defer s.storeLock.Unlock()
	s.statesLock.Lock()
	defer s.statesLock.Unlock()

	// Collect IDs to remove (can't modify map while iterating)
	var idsToRemove []string
	for id := range s.states {
		if _, contains := knownIDHashSet[id]; !contains {
			idsToRemove = append(idsToRemove, id)
		}
	}

	if len(idsToRemove) == 0 {
		return nil
	}

	// For lexicographical ordering, preserve the newest state for startAfterKey
	if lexicographicalOrdering {
		slices.Sort(idsToRemove)
		// Keep the lexicographically newest, and ensure at least one state remains
		newestIdx := len(idsToRemove) - 1
		if len(s.states)-len(idsToRemove) < 1 {
			// Would remove all states, so keep the newest one
			idsToRemove = idsToRemove[:newestIdx]
		}
	}

	// Remove the states
	for _, id := range idsToRemove {
		st := s.states[id]
		if lexicographicalOrdering {
			s.remove(st) // Remove from linked list
		}
		delete(s.states, id)
		if err := s.store.Remove(getStoreKey(id)); err != nil {
			return fmt.Errorf("error while removing the state for ID %s: %w", id, err)
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

func (s *states) SortStatesByLexicographicalOrdering(log *logp.Logger, lexicographicalLookbackKeys int) {
	s.statesLock.Lock()
	defer s.statesLock.Unlock()

	if len(s.states) == 0 {
		return
	}

	s.trimAndBuildLinkedList(log, lexicographicalLookbackKeys)
	log.Debugf("Sorted states by lexicographical ordering: state_count=%d, oldest_state=%s, newest_state=%s", len(s.states), s.head.IDWithLexicographicalOrdering(), s.tail.IDWithLexicographicalOrdering())
}

// trimAndBuildLinkedList trims states to capacity and rebuilds the linked list.
// It acquires storeLock internally for store operations.
func (s *states) trimAndBuildLinkedList(log *logp.Logger, capacity int) {
	sortedKeys := slices.Sorted(maps.Keys(s.states))

	// If over capacity, remove oldest states from both map and store
	if len(sortedKeys) > capacity {
		keysToRemove := sortedKeys[:len(sortedKeys)-capacity]
		s.storeLock.Lock()
		for _, key := range keysToRemove {
			delete(s.states, key)
			// Also remove from persistent store
			if err := s.store.Remove(getStoreKey(key)); err != nil {
				log.Warnf("failed to remove old state from store: %v", err)
			}
		}
		s.storeLock.Unlock()
		// Update sortedKeys to only include remaining keys
		sortedKeys = sortedKeys[len(sortedKeys)-capacity:]
	}

	// Build the doubly linked list in sorted order
	s.buildLinkedListFromKeys(sortedKeys)
}

// buildLinkedListFromKeys rebuilds the doubly linked list from sorted keys.
func (s *states) buildLinkedListFromKeys(sortedKeys []string) {
	var prev *state
	for i, key := range sortedKeys {
		st := s.states[key]
		st.prev = prev
		st.next = nil
		if prev != nil {
			prev.next = st
		}
		if i == 0 {
			s.head = st
		}
		prev = st
	}
	s.tail = prev
}
