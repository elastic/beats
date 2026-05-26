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

// StateVersion is the current version of the state format.
// Increment this when making breaking changes to the State struct.
const StateVersion = 1

// stateStoreName is the statestore bucket name used for cursor entries.
const stateStoreName = "cursor-state"

// State represents the persisted cursor state (versioned for future migrations)
type State struct {
	Version     int       `json:"version"`
	CursorType  string    `json:"cursor_type"`
	CursorValue string    `json:"cursor_value"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Store persists cursor state using libbeat/statestore with memlog backend.
// The Store does not own the registry — the caller (Module) manages its lifecycle.
type Store struct {
	store  *statestore.Store
	logger *logp.Logger
}

// NewStoreFromRegistry creates a Store using a shared statestore.Registry.
// The registry is NOT owned by this Store — the caller (Module) manages its lifecycle.
// Each call obtains a ref-counted Store handle from the shared registry.
func NewStoreFromRegistry(registry *statestore.Registry, logger *logp.Logger) (*Store, error) {
	store, err := registry.Get(stateStoreName)
	if err != nil {
		return nil, fmt.Errorf("failed to open cursor store: %w", err)
	}

	return &Store{
		store:  store,
		logger: logger,
	}, nil
}

// Load retrieves cursor state for the given key.
// Returns nil if the key doesn't exist (not an error).
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

// Close releases the store handle. Must be called when done.
// Close is idempotent — calling it multiple times is safe.
// Only the store handle is closed (decrementing the ref count).
// The shared registry is managed by the caller (Module).
func (s *Store) Close() error {
	if s.store != nil {
		err := s.store.Close()
		s.store = nil
		if err != nil {
			return fmt.Errorf("failed to close store: %w", err)
		}
	}
	return nil
}

// GenerateStateKey creates a unique key for cursor state persistence.
// The key is based on module configuration to ensure separate state per unique config.
//
// IMPORTANT: Any change to the components below will cause the cursor to reset to
// its default value and re-ingest all data from scratch. This includes:
//
// Components that trigger cursor reset:
//   - inputType: "sql" (for namespacing) - hardcoded, never changes
//   - stateIdentity — one of: (a) full database URI/DSN (NOT normalized) when
//     cursor.state_id is unset, or (b) cursor.state_id when set (stable across
//     DSN changes). For DSN-based identity: changing host, password, or connection
//     params resets cursor; includes database name for isolation (prod_db vs
//     test_db on same server). Changing cursor.state_id also resets cursor.
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
//   - Operability: Optional cursor.state_id allows stable cursor continuity
//     across credential/DSN changes when operators explicitly opt in.
//   - Direction safety: A cursor value tracked as a maximum (asc) is semantically
//     incompatible with minimum tracking (desc). Changing direction must reset.
//
// The combined string is hashed via xxhash, so no secrets appear in the stored key.
// Each part is length-prefixed to avoid ambiguity when parts contain the delimiter.
func GenerateStateKey(inputType, stateIdentity, query, cursorColumn, direction string) string {
	keyParts := []string{inputType, stateIdentity, query, cursorColumn, direction}

	var b strings.Builder
	for _, p := range keyParts {
		fmt.Fprintf(&b, "%d:%s|", len(p), p)
	}
	hash := xxhash.Sum64String(b.String())
	return fmt.Sprintf("sql-cursor::%x", hash)
}
