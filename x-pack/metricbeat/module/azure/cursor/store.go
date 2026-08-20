// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package cursor

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/elastic-agent-libs/logp"
)

// StateVersion is the current version of the State struct.
// Increment this when making breaking changes to State.
const StateVersion = 1

// stateStoreName is the statestore bucket name for azure cursor entries.
const stateStoreName = "azure-cursor"

// State is the persisted cursor entry for one metricset.
type State struct {
	Version           int       `json:"version"`
	LastCollectionEnd time.Time `json:"last_collection_end"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Store persists cursor state using libbeat/statestore.
// The Store does not own the registry — the caller (Module) manages its lifecycle.
type Store struct {
	store  *statestore.Store
	logger *logp.Logger
}

// NewStoreFromRegistry creates a Store using a shared statestore.Registry.
// The registry is NOT owned by this Store — call Close() when done to release
// the ref-counted handle.
func NewStoreFromRegistry(registry *statestore.Registry, logger *logp.Logger) (*Store, error) {
	store, err := registry.Get(stateStoreName)
	if err != nil {
		return nil, fmt.Errorf("failed to open azure cursor store: %w", err)
	}
	return &Store{store: store, logger: logger}, nil
}

// Load retrieves cursor state for the given key.
// Returns (nil, nil) when the key does not exist.
func (s *Store) Load(key string) (*State, error) {
	if s.store == nil {
		return nil, errors.New("store is closed")
	}
	exists, err := s.store.Has(key)
	if err != nil {
		return nil, fmt.Errorf("failed to check cursor state existence: %w", err)
	}
	if !exists {
		return nil, nil
	}
	var state State
	if err := s.store.Get(key, &state); err != nil {
		return nil, fmt.Errorf("failed to load cursor state: %w", err)
	}
	return &state, nil
}

// Save persists cursor state for the given key.
func (s *Store) Save(key string, state *State) error {
	if s.store == nil {
		return errors.New("store is closed")
	}
	if err := s.store.Set(key, state); err != nil {
		return fmt.Errorf("failed to save cursor state: %w", err)
	}
	return nil
}

// Close releases the store handle. Idempotent — safe to call multiple times.
func (s *Store) Close() error {
	if s.store != nil {
		err := s.store.Close()
		s.store = nil
		if err != nil {
			return fmt.Errorf("failed to close azure cursor store: %w", err)
		}
	}
	return nil
}
