// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/paths"
)

func setupTestManager(t *testing.T, cfg Config) (*Manager, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home:   tmpDir,
		Config: tmpDir,
		Data:   tmpDir,
		Logs:   tmpDir,
	}

	logger := logp.NewLogger("test-cursor-manager")

	store, err := NewStore(beatPaths, logger)
	require.NoError(t, err)

	mgr, err := NewManager(
		cfg,
		store,
		"localhost:5432",
		"SELECT * FROM logs WHERE id > :cursor ORDER BY id",
		logger,
	)
	require.NoError(t, err)

	cleanup := func() {
		mgr.Close()
	}

	return mgr, cleanup
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		query   string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: Config{
				Enabled: true,
				Column:  "id",
				Type:    CursorTypeInteger,
				Default: "0",
			},
			query:   "SELECT * FROM logs WHERE id > :cursor",
			wantErr: false,
		},
		{
			name: "invalid config - missing column",
			cfg: Config{
				Enabled: true,
				Column:  "",
				Type:    CursorTypeInteger,
				Default: "0",
			},
			query:   "SELECT * FROM logs WHERE id > :cursor",
			wantErr: true,
			errMsg:  "cursor.column is required",
		},
		{
			name: "missing cursor placeholder",
			cfg: Config{
				Enabled: true,
				Column:  "id",
				Type:    CursorTypeInteger,
				Default: "0",
			},
			query:   "SELECT * FROM logs WHERE id > 0",
			wantErr: true,
			errMsg:  "query must contain :cursor placeholder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if testing.Short() {
				t.Skip("skipping manager test in short mode")
			}

			tmpDir := t.TempDir()
			beatPaths := &paths.Path{
				Home:   tmpDir,
				Config: tmpDir,
				Data:   tmpDir,
				Logs:   tmpDir,
			}

			logger := logp.NewLogger("test")
			store, err := NewStore(beatPaths, logger)
			require.NoError(t, err)

			mgr, err := NewManager(tt.cfg, store, "host", tt.query, logger)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				store.Close()
				return
			}
			require.NoError(t, err)
			require.NotNil(t, mgr)
			mgr.Close()
		})
	}
}

func TestManagerCursorValueForQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	cfg := Config{
		Enabled: true,
		Column:  "id",
		Type:    CursorTypeInteger,
		Default: "100",
	}

	mgr, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	// Should return the default value initially
	val := mgr.CursorValueForQuery()
	assert.Equal(t, int64(100), val)

	// String version
	assert.Equal(t, "100", mgr.CursorValueString())
}

func TestManagerUpdateFromResults_Integer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	cfg := Config{
		Enabled: true,
		Column:  "id",
		Type:    CursorTypeInteger,
		Default: "0",
	}

	mgr, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	// Initial value
	assert.Equal(t, "0", mgr.CursorValueString())

	// Update with results
	rows := []mapstr.M{
		{"id": int64(100), "data": "row1"},
		{"id": int64(200), "data": "row2"},
		{"id": int64(150), "data": "row3"},
	}

	err := mgr.UpdateFromResults(rows)
	require.NoError(t, err)

	// Should have the max value (200)
	assert.Equal(t, "200", mgr.CursorValueString())
}

func TestManagerUpdateFromResults_Timestamp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home:   tmpDir,
		Config: tmpDir,
		Data:   tmpDir,
		Logs:   tmpDir,
	}

	logger := logp.NewLogger("test")
	store, err := NewStore(beatPaths, logger)
	require.NoError(t, err)

	cfg := Config{
		Enabled: true,
		Column:  "created_at",
		Type:    CursorTypeTimestamp,
		Default: "2024-01-01T00:00:00Z",
	}

	mgr, err := NewManager(
		cfg,
		store,
		"localhost",
		"SELECT * FROM logs WHERE created_at > :cursor ORDER BY created_at",
		logger,
	)
	require.NoError(t, err)
	defer mgr.Close()

	// Initial value
	assert.Equal(t, "2024-01-01T00:00:00Z", mgr.CursorValueString())

	// Update with results
	t1 := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC) // max
	t3 := time.Date(2024, 6, 15, 11, 0, 0, 0, time.UTC)

	rows := []mapstr.M{
		{"created_at": t1, "data": "row1"},
		{"created_at": t2, "data": "row2"},
		{"created_at": t3, "data": "row3"},
	}

	err = mgr.UpdateFromResults(rows)
	require.NoError(t, err)

	// Should have the max timestamp
	assert.Equal(t, "2024-06-15T12:00:00Z", mgr.CursorValueString())
}

func TestManagerUpdateFromResults_EmptyResults(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	cfg := Config{
		Enabled: true,
		Column:  "id",
		Type:    CursorTypeInteger,
		Default: "100",
	}

	mgr, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	// Update with empty results
	err := mgr.UpdateFromResults([]mapstr.M{})
	require.NoError(t, err)

	// Cursor should be unchanged
	assert.Equal(t, "100", mgr.CursorValueString())
}

func TestManagerUpdateFromResults_NullValues(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	cfg := Config{
		Enabled: true,
		Column:  "id",
		Type:    CursorTypeInteger,
		Default: "0",
	}

	mgr, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	// Update with some NULL values
	rows := []mapstr.M{
		{"id": nil, "data": "row1"},        // NULL
		{"id": int64(100), "data": "row2"}, // valid
		{"id": nil, "data": "row3"},        // NULL
	}

	err := mgr.UpdateFromResults(rows)
	require.NoError(t, err)

	// Should use the valid value (100)
	assert.Equal(t, "100", mgr.CursorValueString())
}

func TestManagerUpdateFromResults_AllNullValues(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	cfg := Config{
		Enabled: true,
		Column:  "id",
		Type:    CursorTypeInteger,
		Default: "50",
	}

	mgr, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	// Update with all NULL values
	rows := []mapstr.M{
		{"id": nil, "data": "row1"},
		{"id": nil, "data": "row2"},
	}

	err := mgr.UpdateFromResults(rows)
	require.NoError(t, err)

	// Cursor should be unchanged
	assert.Equal(t, "50", mgr.CursorValueString())
}

func TestManagerUpdateFromResults_MissingColumn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	cfg := Config{
		Enabled: true,
		Column:  "id",
		Type:    CursorTypeInteger,
		Default: "0",
	}

	mgr, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	// Update with rows missing the cursor column
	rows := []mapstr.M{
		{"other_column": int64(100), "data": "row1"},
		{"id": int64(200), "data": "row2"}, // this one has it
	}

	err := mgr.UpdateFromResults(rows)
	require.NoError(t, err)

	// Should use the value from the row that has the column
	assert.Equal(t, "200", mgr.CursorValueString())
}

func TestManagerUpdateFromResults_CaseInsensitiveColumn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	cfg := Config{
		Enabled: true,
		Column:  "ID", // uppercase in config
		Type:    CursorTypeInteger,
		Default: "0",
	}

	mgr, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	// Update with lowercase column name in result
	rows := []mapstr.M{
		{"id": int64(100), "data": "row1"}, // lowercase in result
	}

	err := mgr.UpdateFromResults(rows)
	require.NoError(t, err)

	// Should match case-insensitively
	assert.Equal(t, "100", mgr.CursorValueString())
}

func TestManagerClose(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	cfg := Config{
		Enabled: true,
		Column:  "id",
		Type:    CursorTypeInteger,
		Default: "0",
	}

	mgr, _ := setupTestManager(t, cfg)

	// First close
	err := mgr.Close()
	require.NoError(t, err)

	// Second close should also succeed (idempotent)
	err = mgr.Close()
	require.NoError(t, err)
}

func TestManagerGetStateKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	cfg := Config{
		Enabled: true,
		Column:  "id",
		Type:    CursorTypeInteger,
		Default: "0",
	}

	mgr, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	key := mgr.GetStateKey()
	assert.NotEmpty(t, key)
	assert.Contains(t, key, "sql-cursor::")
}

func TestManagerGetColumn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	cfg := Config{
		Enabled: true,
		Column:  "my_column",
		Type:    CursorTypeInteger,
		Default: "0",
	}

	mgr, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	assert.Equal(t, "my_column", mgr.GetColumn())
}

func TestManagerStatePersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home:   tmpDir,
		Config: tmpDir,
		Data:   tmpDir,
		Logs:   tmpDir,
	}

	cfg := Config{
		Enabled: true,
		Column:  "id",
		Type:    CursorTypeInteger,
		Default: "0",
	}
	host := "localhost:5432"
	query := "SELECT * FROM logs WHERE id > :cursor ORDER BY id"

	logger := logp.NewLogger("test")

	// Create first manager and update cursor
	store1, err := NewStore(beatPaths, logger)
	require.NoError(t, err)

	mgr1, err := NewManager(cfg, store1, host, query, logger)
	require.NoError(t, err)

	rows := []mapstr.M{
		{"id": int64(500), "data": "row1"},
	}
	err = mgr1.UpdateFromResults(rows)
	require.NoError(t, err)
	assert.Equal(t, "500", mgr1.CursorValueString())

	mgr1.Close()

	// Create second manager - should load persisted state
	store2, err := NewStore(beatPaths, logger)
	require.NoError(t, err)

	mgr2, err := NewManager(cfg, store2, host, query, logger)
	require.NoError(t, err)
	defer mgr2.Close()

	// Should have the persisted value
	assert.Equal(t, "500", mgr2.CursorValueString())
}

// ============================================================================
// Descending scan tests
// ============================================================================

func setupTestManagerWithDirection(t *testing.T, cfg Config) (*Manager, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home:   tmpDir,
		Config: tmpDir,
		Data:   tmpDir,
		Logs:   tmpDir,
	}

	logger := logp.NewLogger("test-cursor-manager-desc")

	store, err := NewStore(beatPaths, logger)
	require.NoError(t, err)

	mgr, err := NewManager(
		cfg,
		store,
		"localhost:5432",
		"SELECT * FROM logs WHERE id < :cursor ORDER BY id DESC",
		logger,
	)
	require.NoError(t, err)

	cleanup := func() {
		mgr.Close()
	}

	return mgr, cleanup
}

func TestManagerUpdateFromResults_Descending(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	cfg := Config{
		Enabled:   true,
		Column:    "id",
		Type:      CursorTypeInteger,
		Default:   "99999",
		Direction: CursorDirectionDesc,
	}

	mgr, cleanup := setupTestManagerWithDirection(t, cfg)
	defer cleanup()

	// Initial value
	assert.Equal(t, "99999", mgr.CursorValueString())

	// Update with results - should track MINIMUM value (200)
	rows := []mapstr.M{
		{"id": int64(500), "data": "row1"},
		{"id": int64(200), "data": "row2"},
		{"id": int64(350), "data": "row3"},
	}

	err := mgr.UpdateFromResults(rows)
	require.NoError(t, err)

	// Should have the min value (200) for descending scan
	assert.Equal(t, "200", mgr.CursorValueString())
}

func TestManagerUpdateFromResults_DescendingWithNulls(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	cfg := Config{
		Enabled:   true,
		Column:    "id",
		Type:      CursorTypeInteger,
		Default:   "99999",
		Direction: CursorDirectionDesc,
	}

	mgr, cleanup := setupTestManagerWithDirection(t, cfg)
	defer cleanup()

	// Update with some NULL values - should find min among valid values
	rows := []mapstr.M{
		{"id": nil, "data": "row1"},
		{"id": int64(300), "data": "row2"},
		{"id": int64(100), "data": "row3"},
		{"id": nil, "data": "row4"},
	}

	err := mgr.UpdateFromResults(rows)
	require.NoError(t, err)

	// Should have the min valid value (100)
	assert.Equal(t, "100", mgr.CursorValueString())
}

func TestManagerUpdateFromResults_AscendingDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	// Direction defaults to "asc" when empty
	cfg := Config{
		Enabled:   true,
		Column:    "id",
		Type:      CursorTypeInteger,
		Default:   "0",
		Direction: "", // should default to asc
	}

	// Validate sets default
	err := cfg.Validate()
	require.NoError(t, err)
	assert.Equal(t, CursorDirectionAsc, cfg.Direction)

	mgr, cleanup := setupTestManagerWithDirection(t, cfg)
	defer cleanup()

	rows := []mapstr.M{
		{"id": int64(100), "data": "row1"},
		{"id": int64(500), "data": "row2"},
		{"id": int64(300), "data": "row3"},
	}

	err = mgr.UpdateFromResults(rows)
	require.NoError(t, err)

	// Ascending: should have the max value (500)
	assert.Equal(t, "500", mgr.CursorValueString())
}

// ============================================================================
// State resilience tests (loadState coverage)
// ============================================================================

func TestManagerLoadState_VersionMismatch(t *testing.T) {
	// When stored state has a different version, Manager should fall back to default
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home: tmpDir, Config: tmpDir, Data: tmpDir, Logs: tmpDir,
	}
	logger := logp.NewLogger("test")

	cfg := Config{
		Enabled: true,
		Column:  "id",
		Type:    CursorTypeInteger,
		Default: "0",
	}
	host := "localhost"
	query := "SELECT * FROM t WHERE id > :cursor"

	// First: create a manager, update cursor to 500, close
	store1, err := NewStore(beatPaths, logger)
	require.NoError(t, err)
	mgr1, err := NewManager(cfg, store1, host, query, logger)
	require.NoError(t, err)

	err = mgr1.UpdateFromResults([]mapstr.M{{"id": int64(500)}})
	require.NoError(t, err)
	assert.Equal(t, "500", mgr1.CursorValueString())
	mgr1.Close()

	// Now tamper with the state: save a state with version=99
	store2, err := NewStore(beatPaths, logger)
	require.NoError(t, err)
	key := GenerateStateKey("sql", host, query, "id", CursorDirectionAsc)
	err = store2.Save(key, &State{
		Version:     99, // Wrong version
		CursorType:  CursorTypeInteger,
		CursorValue: "500",
		UpdatedAt:   time.Now().UTC(),
	})
	require.NoError(t, err)
	store2.Close()

	// Create a new manager — should detect version mismatch and use default
	store3, err := NewStore(beatPaths, logger)
	require.NoError(t, err)
	mgr3, err := NewManager(cfg, store3, host, query, logger)
	require.NoError(t, err)
	defer mgr3.Close()

	assert.Equal(t, "0", mgr3.CursorValueString(), "Should fall back to default on version mismatch")
}

func TestManagerLoadState_TypeMismatch(t *testing.T) {
	// When stored state has a different cursor type, Manager should fall back to default
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home: tmpDir, Config: tmpDir, Data: tmpDir, Logs: tmpDir,
	}
	logger := logp.NewLogger("test")

	host := "localhost"
	query := "SELECT * FROM t WHERE ts > :cursor"

	// Save state as integer type — we need to compute the key that the *new* config
	// would produce. The new config uses CursorTypeTimestamp + default direction "asc".
	// However, the stored state has CursorTypeInteger. The manager detects the type
	// mismatch at load time and falls back to default.
	cfg := Config{
		Enabled: true,
		Column:  "ts",
		Type:    CursorTypeTimestamp, // Different from stored state
		Default: "2024-01-01T00:00:00Z",
	}
	// Validate to set direction default
	require.NoError(t, cfg.Validate())

	store1, err := NewStore(beatPaths, logger)
	require.NoError(t, err)
	key := GenerateStateKey("sql", host, query, "ts", cfg.Direction)
	err = store1.Save(key, &State{
		Version:     StateVersion,
		CursorType:  CursorTypeInteger, // Mismatch with config
		CursorValue: "500",
		UpdatedAt:   time.Now().UTC(),
	})
	require.NoError(t, err)
	store1.Close()

	// Create manager with timestamp config — should detect type mismatch and use default
	store2, err := NewStore(beatPaths, logger)
	require.NoError(t, err)
	mgr, err := NewManager(cfg, store2, host, query, logger)
	require.NoError(t, err)
	defer mgr.Close()

	assert.Equal(t, "2024-01-01T00:00:00Z", mgr.CursorValueString(), "Should fall back to default on type mismatch")
}

func TestManagerLoadState_CorruptedValue(t *testing.T) {
	// When stored cursor value can't be parsed, Manager should fall back to default
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home: tmpDir, Config: tmpDir, Data: tmpDir, Logs: tmpDir,
	}
	logger := logp.NewLogger("test")

	host := "localhost"
	query := "SELECT * FROM t WHERE id > :cursor"

	// Create config first so we can compute the correct state key
	cfg := Config{
		Enabled: true,
		Column:  "id",
		Type:    CursorTypeInteger,
		Default: "0",
	}
	// Validate to set direction default
	require.NoError(t, cfg.Validate())

	// Save state with an unparseable integer value
	store1, err := NewStore(beatPaths, logger)
	require.NoError(t, err)
	key := GenerateStateKey("sql", host, query, "id", cfg.Direction)
	err = store1.Save(key, &State{
		Version:     StateVersion,
		CursorType:  CursorTypeInteger,
		CursorValue: "not-a-number", // Corrupted
		UpdatedAt:   time.Now().UTC(),
	})
	require.NoError(t, err)
	store1.Close()

	// Create manager — should detect corrupt value and use default
	store2, err := NewStore(beatPaths, logger)
	require.NoError(t, err)
	mgr, err := NewManager(cfg, store2, host, query, logger)
	require.NoError(t, err)
	defer mgr.Close()

	assert.Equal(t, "0", mgr.CursorValueString(), "Should fall back to default on corrupted state value")
}

// ============================================================================
// UpdateFromResults edge case tests
// ============================================================================

func TestManagerUpdateFromResults_AllColumnsMissing(t *testing.T) {
	// When ALL rows are missing the cursor column, cursor should remain unchanged
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	cfg := Config{
		Enabled: true,
		Column:  "id",
		Type:    CursorTypeInteger,
		Default: "42",
	}

	mgr, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	rows := []mapstr.M{
		{"other_col": int64(100)},
		{"other_col": int64(200)},
	}

	err := mgr.UpdateFromResults(rows)
	require.NoError(t, err)

	// Cursor unchanged — no valid column found
	assert.Equal(t, "42", mgr.CursorValueString())
}

func TestManagerUpdateFromResults_ParseErrors(t *testing.T) {
	// When cursor column values can't be parsed, they should be skipped
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	cfg := Config{
		Enabled: true,
		Column:  "id",
		Type:    CursorTypeInteger,
		Default: "0",
	}

	mgr, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	rows := []mapstr.M{
		{"id": "not-a-number", "data": "bad1"},      // parse error
		{"id": int64(100), "data": "good"},          // valid
		{"id": "also-not-a-number", "data": "bad2"}, // parse error
	}

	err := mgr.UpdateFromResults(rows)
	require.NoError(t, err)

	// Should use the one valid value
	assert.Equal(t, "100", mgr.CursorValueString())
}

func TestManagerUpdateFromResults_MixedNullAndMissingAndValid(t *testing.T) {
	// Mix of NULL values, missing columns, parse errors, and valid values
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	cfg := Config{
		Enabled: true,
		Column:  "id",
		Type:    CursorTypeInteger,
		Default: "0",
	}

	mgr, cleanup := setupTestManager(t, cfg)
	defer cleanup()

	rows := []mapstr.M{
		{"id": nil, "data": "null row"},      // NULL
		{"other_col": int64(999)},            // missing column
		{"id": "not-a-number"},               // parse error
		{"id": int64(50), "data": "valid 1"}, // valid
		{"id": nil},                          // NULL
		{"id": int64(75), "data": "valid 2"}, // valid — the max
		{"id": "garbage"},                    // parse error
	}

	err := mgr.UpdateFromResults(rows)
	require.NoError(t, err)

	// Should find max among valid values: max(50, 75) = 75
	assert.Equal(t, "75", mgr.CursorValueString())
}

func TestManagerUpdateFromResults_FloatCursor(t *testing.T) {
	// Verify float cursor works end-to-end through Manager
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home: tmpDir, Config: tmpDir, Data: tmpDir, Logs: tmpDir,
	}
	logger := logp.NewLogger("test")

	cfg := Config{
		Enabled: true,
		Column:  "score",
		Type:    CursorTypeFloat,
		Default: "0.0",
	}

	store, err := NewStore(beatPaths, logger)
	require.NoError(t, err)

	mgr, err := NewManager(cfg, store, "host",
		"SELECT * FROM t WHERE score > :cursor ORDER BY score", logger)
	require.NoError(t, err)
	defer mgr.Close()

	assert.Equal(t, "0", mgr.CursorValueString())

	rows := []mapstr.M{
		{"score": float64(1.5)},
		{"score": float64(3.14)},
		{"score": float64(2.7)},
	}
	err = mgr.UpdateFromResults(rows)
	require.NoError(t, err)
	assert.Equal(t, "3.14", mgr.CursorValueString())

	// Verify driver arg is float64
	val := mgr.CursorValueForQuery()
	_, ok := val.(float64)
	assert.True(t, ok, "float cursor should return float64 driver arg")
}

func TestManagerUpdateFromResults_DecimalCursor(t *testing.T) {
	// Verify decimal cursor works end-to-end through Manager
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home: tmpDir, Config: tmpDir, Data: tmpDir, Logs: tmpDir,
	}
	logger := logp.NewLogger("test")

	cfg := Config{
		Enabled: true,
		Column:  "price",
		Type:    CursorTypeDecimal,
		Default: "0.00",
	}

	store, err := NewStore(beatPaths, logger)
	require.NoError(t, err)

	mgr, err := NewManager(cfg, store, "host",
		"SELECT * FROM t WHERE price > :cursor ORDER BY price", logger)
	require.NoError(t, err)
	defer mgr.Close()

	assert.Equal(t, "0", mgr.CursorValueString())

	// Simulate DB returning strings (common for DECIMAL columns via []byte -> string)
	rows := []mapstr.M{
		{"price": "10.25"},
		{"price": "99.99"},
		{"price": "50.50"},
	}
	err = mgr.UpdateFromResults(rows)
	require.NoError(t, err)
	assert.Equal(t, "99.99", mgr.CursorValueString())

	// Verify driver arg is string (for decimal)
	val := mgr.CursorValueForQuery()
	_, ok := val.(string)
	assert.True(t, ok, "decimal cursor should return string driver arg")
}

func TestManagerUpdateFromResults_DecimalPersistenceRoundTrip(t *testing.T) {
	// Verify decimal cursor survives store -> load with exact precision
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home: tmpDir, Config: tmpDir, Data: tmpDir, Logs: tmpDir,
	}
	logger := logp.NewLogger("test")

	cfg := Config{
		Enabled: true,
		Column:  "price",
		Type:    CursorTypeDecimal,
		Default: "0.00",
	}
	host := "localhost"
	query := "SELECT * FROM t WHERE price > :cursor ORDER BY price"

	// First manager: update to a precise decimal value
	store1, err := NewStore(beatPaths, logger)
	require.NoError(t, err)
	mgr1, err := NewManager(cfg, store1, host, query, logger)
	require.NoError(t, err)

	err = mgr1.UpdateFromResults([]mapstr.M{{"price": "123456.789012"}})
	require.NoError(t, err)
	assert.Equal(t, "123456.789012", mgr1.CursorValueString())
	mgr1.Close()

	// Second manager: should load exact same value from store
	store2, err := NewStore(beatPaths, logger)
	require.NoError(t, err)
	mgr2, err := NewManager(cfg, store2, host, query, logger)
	require.NoError(t, err)
	defer mgr2.Close()

	assert.Equal(t, "123456.789012", mgr2.CursorValueString(),
		"Decimal value must survive store->load round trip with exact precision")
}

// ============================================================================
// Timestamp coverage hardening (Manager-level)
// ============================================================================

func TestManagerUpdateFromResults_TimestampDescending(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home: tmpDir, Config: tmpDir, Data: tmpDir, Logs: tmpDir,
	}
	logger := logp.NewLogger("test")

	store, err := NewStore(beatPaths, logger)
	require.NoError(t, err)

	cfg := Config{
		Enabled:   true,
		Column:    "created_at",
		Type:      CursorTypeTimestamp,
		Default:   "2099-12-31T23:59:59Z",
		Direction: CursorDirectionDesc,
	}

	mgr, err := NewManager(
		cfg,
		store,
		"localhost",
		"SELECT * FROM logs WHERE created_at < :cursor ORDER BY created_at DESC",
		logger,
	)
	require.NoError(t, err)
	defer mgr.Close()

	// Initial value should be the high default
	assert.Equal(t, "2099-12-31T23:59:59Z", mgr.CursorValueString())

	// Update with results — descending should track MINIMUM timestamp
	t1 := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC) // min
	t2 := time.Date(2024, 6, 15, 14, 0, 0, 0, time.UTC)
	t3 := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	rows := []mapstr.M{
		{"created_at": t2, "data": "row1"},
		{"created_at": t1, "data": "row2"},
		{"created_at": t3, "data": "row3"},
	}

	err = mgr.UpdateFromResults(rows)
	require.NoError(t, err)
	assert.Equal(t, "2024-06-15T10:00:00Z", mgr.CursorValueString(),
		"descending cursor should track the minimum timestamp")

	// Second batch — cursor should advance (decrease) further
	t4 := time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC) // new min
	t5 := time.Date(2024, 6, 15, 9, 0, 0, 0, time.UTC)

	rows2 := []mapstr.M{
		{"created_at": t4, "data": "row4"},
		{"created_at": t5, "data": "row5"},
	}

	err = mgr.UpdateFromResults(rows2)
	require.NoError(t, err)
	assert.Equal(t, "2024-06-15T08:00:00Z", mgr.CursorValueString(),
		"descending cursor should advance to the new minimum")
}

func TestManagerTimestampPersistenceRoundTrip(t *testing.T) {
	// Critical test: timestamp with nanosecond precision must survive
	// Manager1.Update -> Close -> Manager2.Load without losing precision.
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home: tmpDir, Config: tmpDir, Data: tmpDir, Logs: tmpDir,
	}
	logger := logp.NewLogger("test")

	cfg := Config{
		Enabled: true,
		Column:  "created_at",
		Type:    CursorTypeTimestamp,
		Default: "2024-01-01T00:00:00Z",
	}
	host := "localhost:5432"
	query := "SELECT * FROM logs WHERE created_at > :cursor ORDER BY created_at"

	// --- First manager: update to a timestamp with nanosecond precision ---
	store1, err := NewStore(beatPaths, logger)
	require.NoError(t, err)
	mgr1, err := NewManager(cfg, store1, host, query, logger)
	require.NoError(t, err)

	tsWithNanos := time.Date(2024, 6, 15, 10, 30, 0, 123456789, time.UTC)
	err = mgr1.UpdateFromResults([]mapstr.M{
		{"created_at": tsWithNanos, "data": "row1"},
	})
	require.NoError(t, err)
	assert.Equal(t, "2024-06-15T10:30:00.123456789Z", mgr1.CursorValueString())

	// Verify ToDriverArg preserves nanoseconds
	arg1 := mgr1.CursorValueForQuery()
	tm1, ok := arg1.(time.Time)
	require.True(t, ok)
	assert.Equal(t, 123456789, tm1.Nanosecond())

	mgr1.Close()

	// --- Second manager: should load exact same timestamp from store ---
	store2, err := NewStore(beatPaths, logger)
	require.NoError(t, err)
	mgr2, err := NewManager(cfg, store2, host, query, logger)
	require.NoError(t, err)
	defer mgr2.Close()

	assert.Equal(t, "2024-06-15T10:30:00.123456789Z", mgr2.CursorValueString(),
		"Timestamp with nanoseconds must survive store->load round trip")

	// Verify ToDriverArg still produces the correct time.Time after reload
	arg2 := mgr2.CursorValueForQuery()
	tm2, ok := arg2.(time.Time)
	require.True(t, ok)
	assert.Equal(t, 123456789, tm2.Nanosecond(),
		"Nanosecond component must survive store->close->load->ToDriverArg cycle")
	assert.True(t, tm1.Equal(tm2),
		"time.Time from first and second manager must be exactly equal")
}
