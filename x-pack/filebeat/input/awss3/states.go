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

// states handles list of s3 object state. One must use newStates to instantiate a
// file states registry. Using the zero-value is not safe.
type states struct {
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

	stateTable, err := loadS3StatesFromRegistry(log, store, listPrefix)
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
func loadS3StatesFromRegistry(log *logp.Logger, store *statestore.Store, prefix string) (map[string]*state, error) {
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
			stateTable[st.ID()] = &st
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return stateTable, nil
}
