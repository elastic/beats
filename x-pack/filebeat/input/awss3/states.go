// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"strings"
	"sync"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/libbeat/statestore"
)

const awsS3ObjectStatePrefix = "filebeat::aws-s3::state::"

// states handles list of s3 object state. One must use newStates to instantiate a
// file states registry. Using the zero-value is not safe.
type states struct {
	log *logp.Logger

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
func newStates(ctx v2.Context, store *statestore.Store) (*states, error) {
	states := &states{
		log:    ctx.Logger.Named("states"),
		states: map[string]state{},
		store:  store,
	}
	return states, states.loadFromRegistry()
}

func (s *states) IsProcessed(state state) bool {
	s.statesLock.Lock()
	defer s.statesLock.Unlock()
	// Our in-memory table only stores completed objects
	_, ok := s.states[state.ID()]
	return ok
}

func (s *states) AddState(state state) {

	id := state.ID()
	// Update in-memory copy
	s.statesLock.Lock()
	s.states[id] = state
	s.statesLock.Unlock()

	// Persist to the registry
	s.storeLock.Lock()
	key := awsS3ObjectStatePrefix + id
	if err := s.store.Set(key, state); err != nil {
		s.log.Errorw("Failed to write states to the registry", "error", err)
	}
	s.storeLock.Unlock()
}

func (s *states) loadFromRegistry() error {
	states := map[string]state{}

	s.storeLock.Lock()
	err := s.store.Each(func(key string, dec statestore.ValueDecoder) (bool, error) {
		if !strings.HasPrefix(key, awsS3ObjectStatePrefix) {
			return true, nil
		}

		// try to decode. Ignore faulty/incompatible values.
		var st state
		if err := dec.Decode(&st); err != nil {
			// Skip this key but continue iteration
			s.log.Warnf("invalid S3 state loading object key %v", key)
			//nolint:nilerr // One bad object shouldn't stop iteration
			return true, nil
		}
		if !st.Stored && !st.Failed {
			// This is from an older version where state could be stored in the
			// registry even if the object wasn't processed, or if it encountered
			// ephemeral download errors. We don't add these to the in-memory cache,
			// so if we see them during a bucket scan we will still retry them.
			return true, nil
		}

		states[st.ID()] = st
		return true, nil
	})
	s.storeLock.Unlock()
	if err != nil {
		return err
	}

	s.statesLock.Lock()
	s.states = states
	s.statesLock.Unlock()

	return nil
}
