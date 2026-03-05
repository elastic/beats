// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const nilValuePlaceholder = "<nil>"

// Manager handles cursor state lifecycle including loading, updating, and persisting cursor values.
type Manager struct {
	config      Config
	store       *Store
	stateKey    string
	cursorValue *Value
	typeLocked  bool
	mu          sync.Mutex
	logger      *logp.Logger
}

// NewManager creates a new cursor manager.
// It validates the configuration, initializes the store, and loads any existing state.
//
// Parameters:
//   - cfg: Cursor configuration from metricbeat.yml
//   - store: State persistence store (memlog-backed)
//   - dsn: Full database URI/DSN for state key generation (hashed, not stored in cleartext)
//   - query: Original SQL query (before placeholder translation)
//   - logger: Logger instance for this cursor
//
// The manager takes ownership of the store and will close it when Close() is called.
func NewManager(cfg Config, store *Store, dsn, query string, logger *logp.Logger) (*Manager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("cursor config validation failed: %w", err)
	}

	if err := ValidateQueryHasCursor(query); err != nil {
		return nil, err
	}

	m := &Manager{
		config:   cfg,
		store:    store,
		stateKey: GenerateStateKey("sql", dsn, query, cfg.Column, cfg.Direction),
		// Explicit type from config is authoritative and cannot be auto-adjusted.
		typeLocked: cfg.Type != "",
		logger:     logger,
	}

	if err := m.loadState(); err != nil {
		return nil, fmt.Errorf("failed to initialize cursor state: %w", err)
	}

	return m, nil
}

// Close releases resources held by the manager.
// This must be called when the MetricSet is closed to release statestore resources.
// Close is idempotent - calling it multiple times is safe.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.store != nil {
		err := m.store.Close()
		m.store = nil
		return err
	}
	return nil
}

// loadState loads cursor state from the store or initializes with the default value.
func (m *Manager) loadState() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.store.Load(m.stateKey)
	if err != nil {
		m.logger.Warnf("Failed to load cursor state, will use default: %v", err)
		state = nil
	}

	if state == nil || !m.isStateValid(state) {
		return m.initDefault()
	}

	// In auto mode (cursor.type omitted), state becomes authoritative once loaded.
	if m.config.Type == "" {
		m.config.Type = state.CursorType
		m.typeLocked = true
	}

	// Parse the stored value
	val, err := ParseValue(state.CursorValue, state.CursorType)
	if err != nil {
		m.logger.Warnf("Invalid cursor state value, using default: %v", err)
		return m.initDefault()
	}

	m.cursorValue = val
	m.logger.Infof("Cursor loaded: type=%s value=%s", state.CursorType, val.Raw)
	return nil
}

// isStateValid checks whether the loaded state is compatible with the current
// configuration. It returns false (and logs the reason) when the state is nil,
// has a version mismatch, or a cursor-type mismatch.
func (m *Manager) isStateValid(state *State) bool {
	if state == nil {
		return false
	}
	if state.Version != StateVersion {
		m.logger.Warnf("Unsupported cursor state version %d (expected %d), using default",
			state.Version, StateVersion)
		return false
	}
	if !isValidCursorType(state.CursorType) {
		m.logger.Warnf("Unsupported cursor type in state=%s, using default", state.CursorType)
		return false
	}
	if state.CursorType != m.config.Type {
		if m.config.Type != "" {
			m.logger.Warnf("Cursor type mismatch (state=%s, config=%s), using default",
				state.CursorType, m.config.Type)
			return false
		}
	}
	return true
}

// initDefault initializes the cursor with the default value from config.
// Caller must hold m.mu.
func (m *Manager) initDefault() error {
	if m.config.Type == "" {
		inferredType, err := InferTypeFromDefaultValue(m.config.Default)
		if err != nil {
			return fmt.Errorf("failed to infer cursor type from default value %q: %w", m.config.Default, err)
		}
		m.config.Type = inferredType
		m.logger.Infof("Cursor type inferred from default value: %s", inferredType)
	}

	defaultVal, err := ParseValue(m.config.Default, m.config.Type)
	if err != nil {
		return fmt.Errorf("invalid default cursor value: %w", err)
	}

	m.cursorValue = defaultVal
	m.logger.Infof("Cursor initialized: column=%s, type=%s, default=%s",
		m.config.Column, m.config.Type, defaultVal.Raw)
	return nil
}

// CursorValueForQuery returns the cursor value converted to a driver-compatible argument.
// The returned value is ready to be passed to db.QueryContext().
func (m *Manager) CursorValueForQuery() interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cursorValue == nil {
		return nil
	}
	return m.cursorValue.ToDriverArg()
}

// CursorValueString returns the cursor value as a string (for logging).
func (m *Manager) CursorValueString() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cursorValue == nil {
		return nilValuePlaceholder
	}
	return m.cursorValue.Raw
}

// UpdateFromResults processes query results and updates the cursor.
// For ascending direction (default), it finds the maximum cursor value.
// For descending direction, it finds the minimum cursor value.
// The selected value is persisted as the new cursor state.
//
// The function is resilient to errors:
//   - Missing cursor column: logs error, skips that row
//   - NULL cursor value: logs warning, skips that row
//   - Parse error: logs error, skips that row
//   - If all rows have issues: cursor remains unchanged
//
// Returns an error only if state persistence fails (events are already emitted at this point).
func (m *Manager) UpdateFromResults(rows []mapstr.M) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(rows) == 0 {
		m.logger.Debug("No rows returned, cursor unchanged")
		return nil
	}

	descending := m.config.Direction == CursorDirectionDesc
	columnLower := strings.ToLower(m.config.Column)
	var bestValue *Value
	var processed int
	var foundCount int

	// In auto mode, infer the best type from the returned rows before parsing.
	if !m.typeLocked {
		inferredType, err := inferTypeFromRows(rows, columnLower)
		if err != nil {
			m.logger.Debugf("Could not infer cursor type from result rows, using current type=%s: %v",
				m.config.Type, err)
		} else if inferredType != "" && inferredType != m.config.Type {
			m.logger.Infof("Cursor type inferred from query results: %s", inferredType)
			m.config.Type = inferredType
		}
	}

	for idx, row := range rows {
		// Find the cursor column (case-insensitive)
		var rawVal interface{}
		var found bool

		for key, val := range row {
			if strings.ToLower(key) == columnLower {
				rawVal = val
				found = true
				break
			}
		}

		if !found {
			// Don't spam logs per-row; a single prominent error is emitted below
			// if the column is missing from all rows.
			continue
		}
		foundCount++

		if rawVal == nil {
			m.logger.Warnf("NULL value in cursor column, row %d", idx+1)
			continue
		}

		val, err := FromDatabaseValue(rawVal, m.config.Type)
		if err != nil && !m.typeLocked {
			inferredType, inferErr := InferTypeFromDatabaseValue(rawVal)
			if inferErr == nil && inferredType != "" && inferredType != m.config.Type {
				fallbackVal, fallbackErr := FromDatabaseValue(rawVal, inferredType)
				if fallbackErr == nil {
					if bestValue != nil && bestValue.Type != inferredType {
						convertedBest, convErr := ParseValue(bestValue.Raw, inferredType)
						if convErr != nil {
							m.logger.Errorf("Failed to convert existing cursor candidate from %s to %s in row %d: %v",
								bestValue.Type, inferredType, idx+1, convErr)
							continue
						}
						bestValue = convertedBest
					}
					m.logger.Infof("Cursor type adjusted from %s to %s based on row %d",
						m.config.Type, inferredType, idx+1)
					m.config.Type = inferredType
					val = fallbackVal
					err = nil
				}
			}
		}
		if err != nil {
			m.logger.Errorf("Failed to parse cursor value in row %d: %v", idx+1, err)
			continue
		}

		processed++

		// Track the best value (max for ascending, min for descending)
		if bestValue == nil {
			bestValue = val
			continue
		}

		cmp, err := val.Compare(bestValue)
		if err != nil {
			m.logger.Errorf("Failed to compare cursor values (a=%s, b=%s): %v", val.Raw, bestValue.Raw, err)
			continue
		}

		if descending {
			if cmp < 0 {
				bestValue = val
			}
		} else {
			if cmp > 0 {
				bestValue = val
			}
		}
	}

	if bestValue == nil {
		// If cursor column was not found in any row, emit a single prominent error
		// explaining the likely misconfiguration. Otherwise, the column exists but
		// all values were NULL or invalid.
		if foundCount == 0 {
			m.logger.Errorf("Cursor column %q was not found in any of the %d result rows. "+
				"The cursor column must be included in the SELECT clause of your SQL query. "+
				"The cursor will not advance until the column appears in results.",
				m.config.Column, len(rows))
		} else {
			m.logger.Warn("All cursor column values were NULL or invalid, cursor unchanged")
		}
		return nil
	}

	previousValue := m.cursorValue
	m.cursorValue = bestValue

	// Persist the new state
	state := &State{
		Version:     StateVersion,
		CursorType:  m.config.Type,
		CursorValue: bestValue.Raw,
		UpdatedAt:   time.Now().UTC(),
	}

	if m.store == nil {
		return errors.New("cursor store is closed")
	}
	if err := m.store.Save(m.stateKey, state); err != nil {
		// Revert in-memory state on save failure to keep consistency.
		// We restore the exact previous *Value rather than re-parsing from string
		// to avoid any edge-case parse issues.
		m.cursorValue = previousValue
		return fmt.Errorf("failed to save cursor state: %w", err)
	}

	prevRaw := nilValuePlaceholder
	if previousValue != nil {
		prevRaw = previousValue.Raw
	}
	m.logger.Infof("Cursor updated: %s → %s (%d rows processed)", prevRaw, bestValue.Raw, processed)
	return nil
}

// GetStateKey returns the state key (for testing/debugging).
func (m *Manager) GetStateKey() string {
	return m.stateKey
}

// GetColumn returns the cursor column name.
func (m *Manager) GetColumn() string {
	return m.config.Column
}

func inferTypeFromRows(rows []mapstr.M, columnLower string) (string, error) {
	inferredType := ""
	for _, row := range rows {
		var (
			rawVal interface{}
			found  bool
		)

		for key, val := range row {
			if strings.ToLower(key) == columnLower {
				rawVal = val
				found = true
				break
			}
		}
		if !found || rawVal == nil {
			continue
		}

		rowType, err := InferTypeFromDatabaseValue(rawVal)
		if err != nil {
			continue
		}

		mergedType, err := mergeInferredTypes(inferredType, rowType)
		if err != nil {
			return "", err
		}
		inferredType = mergedType
	}

	if inferredType == "" {
		return "", errors.New("no inferable cursor values found in rows")
	}
	return inferredType, nil
}

func mergeInferredTypes(current, candidate string) (string, error) {
	if current == "" {
		return candidate, nil
	}
	if current == candidate {
		return current, nil
	}

	// date + timestamp should converge to timestamp.
	if (current == CursorTypeDate && candidate == CursorTypeTimestamp) ||
		(current == CursorTypeTimestamp && candidate == CursorTypeDate) {
		return CursorTypeTimestamp, nil
	}

	if isNumericCursorType(current) && isNumericCursorType(candidate) {
		return mergeNumericTypes(current, candidate), nil
	}

	return "", fmt.Errorf("conflicting inferred cursor types: %s vs %s", current, candidate)
}

func mergeNumericTypes(a, b string) string {
	if a == CursorTypeDecimal || b == CursorTypeDecimal {
		return CursorTypeDecimal
	}
	if a == CursorTypeFloat || b == CursorTypeFloat {
		return CursorTypeFloat
	}
	return CursorTypeInteger
}

func isNumericCursorType(t string) bool {
	switch t {
	case CursorTypeInteger, CursorTypeFloat, CursorTypeDecimal:
		return true
	default:
		return false
	}
}
