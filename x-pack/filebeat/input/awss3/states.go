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
	head       *state // oldest (front) in lexicographical ordering
	tail       *state // newest (back) in lexicographical ordering

	// The store used to persist state changes to the registry.
	// storeLock must be held to access store.
	store     *statestore.Store
	storeLock sync.Mutex

	// Accepted prefixes of state keys of this registry
	keyPrefix string
}

// newStates generates a new states registry.
func newStates(log *logp.Logger, stateStore statestore.States, listPrefix string, lexicographicalOrdering bool) (*states, error) {
	store, err := stateStore.StoreFor("")
	if err != nil {
		return nil, fmt.Errorf("can't access persistent store: %w", err)
	}

	stateTable, err := loadS3StatesFromRegistry(log, store, listPrefix, lexicographicalOrdering)
	if err != nil {
		return nil, fmt.Errorf("loading S3 input state: %w", err)
	}

	return &states{
		store:     store,
		states:    stateTable,
		keyPrefix: listPrefix,
	}, nil
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

	if lexicographicalOrdering {
		// maintain a doubly linked list structure for lexicographical ordering
		// with lexicographicalLookbackKeys as capacity
		id = st.IDWithLexicographicalOrdering()
		// If state already exists, update it and move to back (newest position)
		if existing, exists := s.states[id]; exists {
			// Update the existing state's fields
			existing.Stored = st.Stored
			existing.Failed = st.Failed
			s.moveToBack(existing)
		} else {
			// New state: check capacity and add to back
			// If at capacity, remove the oldest (front) state
			if len(s.states) >= lexicographicalLookbackKeys {
				oldest = s.head
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
	if oldest != nil {
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
			// remove from sate & store as ID is no longer seen in known ID set
			delete(s.states, id)
			err := s.store.Remove(getStoreKey(id))
			if err != nil {
				return fmt.Errorf("error while removing the state for ID %s: %w", id, err)
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

func (s *states) GetKeys() []string {
	return slices.Collect(maps.Keys(s.states))
}

func (s *states) SortStatesByLexicographicalOrdering(log *logp.Logger) {
	// s.statesLock.Lock()
	// defer s.statesLock.Unlock()

	if len(s.states) == 0 {
		return
	}

	log.Debugf("Before sorting states by lexicographical ordering - len(s.states): %d, s.head: %v, s.tail: %v, s.states.keys: %v", len(s.states), s.head, s.tail, s.GetKeys())
	sortedKeys := slices.Sorted(maps.Keys(s.states))

	// Rebuild the doubly linked list in sorted order
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

	log.Debugf("Sorted states by lexicographical ordering - len(s.states): %d, s.head: %v, s.tail: %v, s.states.keys: %v", len(s.states), s.head, s.tail, sortedKeys)
}
