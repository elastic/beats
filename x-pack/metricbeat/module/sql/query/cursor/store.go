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
	"github.com/elastic/elastic-agent-libs/logp"
)

// registryOwnership indicates whether the Store owns the registry lifecycle.
type registryOwnership bool

const (
	ownsRegistry       registryOwnership = true
	doesNotOwnRegistry registryOwnership = false
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
	registry     *statestore.Registry
	ownsRegistry registryOwnership
	store        *statestore.Store
	logger       *logp.Logger
}

// NewStoreFromRegistry creates a Store using a shared statestore.Registry.
// The registry is NOT owned by this Store — the caller (Module) manages its lifecycle.
// Each call obtains a ref-counted Store handle from the shared registry.
func NewStoreFromRegistry(registry *statestore.Registry, logger *logp.Logger) (*Store, error) {
	store, err := registry.Get("cursor-state")
	if err != nil {
		return nil, fmt.Errorf("failed to open cursor store: %w", err)
	}

	return &Store{
		registry:     nil, // not owned
		ownsRegistry: doesNotOwnRegistry,
		store:        store,
		logger:       logger,
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
// If the Store was created via NewStoreFromRegistry, only the store handle is
// closed (decrementing the ref count). The shared registry is not closed.
func (s *Store) Close() error {
	var errs []error

	if s.store != nil {
		if err := s.store.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close store: %w", err))
		}
		s.store = nil
	}

	// Only close the registry if this Store owns it (created via NewStore).
	// Stores created via NewStoreFromRegistry share a Module-level registry.
	if s.ownsRegistry && s.registry != nil {
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
// IMPORTANT: Any change to the components below will cause the cursor to reset to
// its default value and re-ingest all data from scratch. This includes:
//
// Components that trigger cursor reset:
//   - inputType: "sql" (for namespacing) - hardcoded, never changes
//   - dsn: Full database URI/DSN (NOT normalized)
//   - Changing host (localhost → 127.0.0.1) resets cursor
//   - Changing password in DSN resets cursor
//   - Adding connection params (?sslmode=require) resets cursor
//   - Includes database name for isolation (prod_db vs test_db on same server)
//   - query: Full query string (NOT normalized - exact byte match)
//   - Adding/removing whitespace resets cursor
//   - Changing SQL capitalization (SELECT → select) resets cursor
//   - Changing LIMIT value resets cursor
//   - Modifying WHERE clause resets cursor
//   - cursorColumn: The column name being tracked
//   - Renaming cursor column resets cursor
//   - direction: The cursor scan direction ("asc" or "desc")
//   - Changing direction resets cursor (prevents using a max-tracked value
//     as a min-tracking starting point, or vice versa)
//
// Design rationale:
//   - Safety: Query changes could affect result set semantics. Better to start
//     fresh than risk missing data or duplicates from incompatible queries.
//   - Simplicity: SQL normalization is complex and database-specific. Avoiding
//     SQL parsing keeps implementation simple and reliable.
//   - Isolation: Different databases on same server (e.g., prod_db vs test_db)
//     must have separate cursor states. Including full DSN ensures this.
//   - Direction safety: A cursor value tracked as a maximum (asc) is semantically
//     incompatible with minimum tracking (desc). Changing direction must reset.
//
// The combined string is hashed via xxhash, so no secrets appear in the stored key.
func GenerateStateKey(inputType, dsn, query, cursorColumn, direction string) string {
	keyParts := []string{inputType, dsn, query, cursorColumn, direction}

	combined := strings.Join(keyParts, "|")
	hash := xxhash.Sum64String(combined)
	return fmt.Sprintf("sql-cursor::%x", hash)
}
