// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/beats/v7/filebeat/beater"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/logp"
)

const awsS3ObjectStatePrefix = "filebeat::aws-s3::state::"

// states handles list of s3 object state. One must use newStates to instantiate a
// file states registry. Using the zero-value is not safe.
type states struct {
	// Completed S3 object states, indexed by state ID.
	// statesLock must be held to access states.
	states     map[string]state
	statesLock sync.Mutex

	// The store used to persist state changes to the registry.
	// storeLock must be held to access store.
	store     *statestore.Store
	storeLock sync.Mutex
}

// newStates generates a new states registry.
func newStates(log *logp.Logger, stateStore beater.StateStore) (*states, error) {
	store, err := stateStore.Access()
	if err != nil {
		return nil, fmt.Errorf("can't access persistent store: %w", err)
	}

	stateTable, err := loadS3StatesFromRegistry(log, store)
	if err != nil {
		return nil, fmt.Errorf("loading S3 input state: %w", err)
	}

	return &states{
		store:  store,
		states: stateTable,
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
	id := state.ID()
	// Update in-memory copy
	s.statesLock.Lock()
	s.states[id] = state
	s.statesLock.Unlock()

	// Persist to the registry
	s.storeLock.Lock()
	defer s.storeLock.Unlock()
	key := awsS3ObjectStatePrefix + id
	if err := s.store.Set(key, state); err != nil {
		return err
	}
	return nil
}

func (s *states) Close() {
	s.storeLock.Lock()
	s.store.Close()
	s.storeLock.Unlock()
}

func loadS3StatesFromRegistry(log *logp.Logger, store *statestore.Store) (map[string]state, error) {
	stateTable := map[string]state{}
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

		stateTable[st.ID()] = st
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return stateTable, nil
}
