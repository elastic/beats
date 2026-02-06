// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

// StateVersion is the current version of the state format.
// Increment this when making breaking changes to the State struct.
const StateVersion = 1

// State represents the persisted cursor state (versioned for future migrations)
type State struct {
	Version     int       `json:"version"`
	CursorType  string    `json:"cursor_type"`
	CursorValue string    `json:"cursor_value"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Store persists cursor state using libbeat/statestore with memlog backend.
type Store struct {
	registry *statestore.Registry
	store    *statestore.Store
	logger   *logp.Logger
}

// NewStore creates a memlog-backed store for cursor persistence.
// The store is created at {data.path}/sql-cursor/
// The caller is responsible for calling Close() when done.
func NewStore(beatPaths *paths.Path, logger *logp.Logger) (*Store, error) {
	if beatPaths == nil {
		beatPaths = paths.Paths
	}

	dataPath := beatPaths.Resolve(paths.Data, "sql-cursor")

	reg, err := memlog.New(logger.Named("memlog"), memlog.Settings{
		Root:     dataPath,
		FileMode: 0600,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create memlog registry: %w", err)
	}

	registry := statestore.NewRegistry(reg)
	store, err := registry.Get("cursor-state")
	if err != nil {
		if closeErr := registry.Close(); closeErr != nil {
			logger.Warnf("Failed to close registry after store creation error: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to open cursor store: %w", err)
	}

	return &Store{
		registry: registry,
		store:    store,
		logger:   logger,
	}, nil
}

// Load retrieves cursor state for the given key.
// Returns nil if the key doesn't exist (not an error).
func (s *Store) Load(key string) (*State, error) {
	if s.store == nil {
		return nil, errors.New("store is closed")
	}
	var state State
	err := s.store.Get(key, &state)
	if err != nil {
		// Check if key doesn't exist (not a real error)
		if isKeyNotFoundError(err) {
			return nil, nil
		}
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

// Close releases store resources. Must be called when done.
// Close is idempotent - calling it multiple times is safe.
func (s *Store) Close() error {
	var errs []error

	if s.store != nil {
		if err := s.store.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close store: %w", err))
		}
		s.store = nil
	}

	if s.registry != nil {
		if err := s.registry.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close registry: %w", err))
		}
		s.registry = nil
	}

	return errors.Join(errs...)
}

// isKeyNotFoundError checks if the error indicates a missing key.
// The memlog backend returns "key unknown" when a key is not found,
// which gets wrapped by statestore's ErrorOperation.
func isKeyNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// memlog backend returns errKeyUnknown ("key unknown") for missing keys,
	// wrapped by statestore's ErrorOperation as:
	// "failed in get operation on store 'cursor-state': key unknown"
	errStr := err.Error()
	return strings.Contains(errStr, "key unknown")
}

// GenerateStateKey creates a unique key for cursor state persistence.
// The key is based on module configuration to ensure separate state per unique config.
//
// Components:
//   - inputType: "sql" (for namespacing)
//   - moduleID: Optional module ID from config (for multi-instance support)
//   - dsn: Full database URI/DSN (includes database name for proper isolation)
//   - query: Full query string (no normalization - any change resets cursor)
//   - cursorColumn: The column being tracked
//
// Any change to these components will result in a different key, effectively
// resetting the cursor to its default value. The combined string is hashed
// via xxhash, so no secrets are stored in the key itself.
func GenerateStateKey(inputType, moduleID, dsn, query, cursorColumn string) string {
	var keyParts []string
	keyParts = append(keyParts, inputType) // "sql"
	if moduleID != "" {
		keyParts = append(keyParts, moduleID)
	}
	keyParts = append(keyParts, dsn, query, cursorColumn)

	combined := strings.Join(keyParts, "|")
	hash := xxhash.Sum64String(combined)
	return fmt.Sprintf("sql-cursor::%x", hash)
}
