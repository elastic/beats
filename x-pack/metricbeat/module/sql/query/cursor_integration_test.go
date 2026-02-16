// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && !requirefips

package query

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/godror/godror"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/paths"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/mysql"
	"github.com/elastic/beats/v7/metricbeat/module/postgresql"
	sqlmod "github.com/elastic/beats/v7/x-pack/metricbeat/module/sql"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/sql/query/cursor"
)

const testTableName = "cursor_test_events"

// newMetricSetWithPaths creates a MetricSet with custom paths for cursor storage.
// It sets the global paths.Paths.Data to the test's data directory so that
// GetCursorRegistry resolves to the correct temp path, and uses t.Cleanup
// to restore the original value when the test completes.
func newMetricSetWithPaths(t *testing.T, config map[string]interface{}, p *paths.Path) mb.MetricSet {
	t.Helper()

	// Override the global data path so GetCursorRegistry creates its
	// registry under the per-test temp directory instead of the shared
	// process-level path.
	origData := paths.Paths.Data
	paths.Paths.Data = p.Data
	t.Cleanup(func() { paths.Paths.Data = origData })

	c, err := conf.NewConfigFrom(config)
	require.NoError(t, err)

	logger := logptest.NewTestingLogger(t, "")
	_, metricsets, err := mb.NewModule(c, mb.Registry, p, logger)
	require.NoError(t, err)
	require.Len(t, metricsets, 1)

	return metricsets[0]
}

// createTestPaths creates paths for testing with cursor
func createTestPaths(t *testing.T) *paths.Path {
	t.Helper()
	tmpDir := t.TempDir()
	return &paths.Path{
		Home:   tmpDir,
		Config: tmpDir,
		Data:   tmpDir,
		Logs:   tmpDir,
	}
}

// TestPostgreSQLCursor tests cursor functionality with PostgreSQL
func TestPostgreSQLCursor(t *testing.T) {
	service := compose.EnsureUp(t, "postgresql")
	host, port, err := net.SplitHostPort(service.Host())
	require.NoError(t, err)

	user := postgresql.GetEnvUsername()
	password := postgresql.GetEnvPassword()
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/?sslmode=disable", user, password, host, port)

	// Set up test table
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	setupPostgresTestTable(t, db)
	defer cleanupTestTable(t, db, "postgres")

	// Test integer cursor
	t.Run("integer cursor", func(t *testing.T) {
		testIntegerCursor(t, "postgres", dsn)
	})

	// Test timestamp cursor
	t.Run("timestamp cursor", func(t *testing.T) {
		testTimestampCursor(t, "postgres", dsn)
	})

	// Test float cursor
	t.Run("float cursor", func(t *testing.T) {
		testFloatCursor(t, "postgres", dsn)
	})

	// Test decimal cursor
	t.Run("decimal cursor", func(t *testing.T) {
		testDecimalCursor(t, "postgres", dsn)
	})

	// Test descending integer cursor
	t.Run("descending integer cursor", func(t *testing.T) {
		testDescendingIntegerCursor(t, "postgres", dsn)
	})

	// Test compound WHERE clause (cursor + additional filter)
	t.Run("compound where clause", func(t *testing.T) {
		testCompoundWhereCursor(t, "postgres", dsn)
	})
}

// TestMySQLCursor tests cursor functionality with MySQL
func TestMySQLCursor(t *testing.T) {
	service := compose.EnsureUp(t, "mysql")
	baseDSN := mysql.GetMySQLEnvDSN(service.Host())

	// First connect without database to create test database
	db0, err := sql.Open("mysql", baseDSN)
	require.NoError(t, err)
	_, err = db0.Exec("CREATE DATABASE IF NOT EXISTS cursor_test")
	require.NoError(t, err)
	db0.Close()

	// Now connect to the test database
	dsn := baseDSN + "cursor_test"

	// Set up test table
	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	defer db.Close()
	defer func() { _, _ = db.Exec("DROP DATABASE IF EXISTS cursor_test") }()

	setupMySQLTestTable(t, db)
	defer cleanupTestTable(t, db, "mysql")

	// Test integer cursor
	t.Run("integer cursor", func(t *testing.T) {
		testIntegerCursor(t, "mysql", dsn)
	})

	// Test timestamp cursor
	t.Run("timestamp cursor", func(t *testing.T) {
		testTimestampCursor(t, "mysql", dsn)
	})

	// Test decimal cursor (MySQL DECIMAL is very common)
	t.Run("decimal cursor", func(t *testing.T) {
		testDecimalCursor(t, "mysql", dsn)
	})

	// Test descending scan on MySQL
	t.Run("descending integer cursor", func(t *testing.T) {
		testDescendingIntegerCursor(t, "mysql", dsn)
	})
}

// insertTestData inserts n rows into the test table using the appropriate
// placeholder syntax for the given driver.
func insertTestData(t *testing.T, db *sql.DB, driver string, n int) {
	t.Helper()

	var insertSQL string
	switch driver {
	case "postgres":
		insertSQL = fmt.Sprintf(`INSERT INTO %s (event_data) VALUES ($1)`, testTableName)
	case "mysql":
		insertSQL = fmt.Sprintf(`INSERT INTO %s (event_data) VALUES (?)`, testTableName)
	default:
		t.Fatalf("unsupported driver for insertTestData: %s", driver)
	}

	for i := 0; i < n; i++ {
		_, err := db.Exec(insertSQL, fmt.Sprintf("event-%d", i))
		require.NoError(t, err)
	}
}

func setupPostgresTestTable(t *testing.T, db *sql.DB) {
	t.Helper()

	// Drop table if exists
	_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", testTableName))
	require.NoError(t, err)

	// Create table with columns for all cursor types
	createSQL := fmt.Sprintf(`
		CREATE TABLE %s (
			id SERIAL PRIMARY KEY,
			event_data TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			score DOUBLE PRECISION,
			price NUMERIC(10,2)
		)
	`, testTableName)
	_, err = db.Exec(createSQL)
	require.NoError(t, err)

	// Insert test data with values for all cursor types
	insertSQL := fmt.Sprintf(`INSERT INTO %s (event_data, created_at, score, price) VALUES ($1, $2, $3, $4)`, testTableName)
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		score := float64(i+1) * 1.5   // 1.5, 3.0, 4.5, 6.0, 7.5
		price := float64(i+1) * 10.25 // 10.25, 20.50, 30.75, 41.00, 51.25
		_, err := db.Exec(insertSQL, fmt.Sprintf("event-%d", i), now.Add(time.Duration(i)*time.Second), score, price)
		require.NoError(t, err)
	}
}

func setupMySQLTestTable(t *testing.T, db *sql.DB) {
	t.Helper()

	// Drop table if exists
	_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", testTableName))
	require.NoError(t, err)

	// Create table with columns for all cursor types
	createSQL := fmt.Sprintf(`
		CREATE TABLE %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			event_data TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			price DECIMAL(10,2)
		)
	`, testTableName)
	_, err = db.Exec(createSQL)
	require.NoError(t, err)

	// Insert test data with values for all cursor types
	insertSQL := fmt.Sprintf(`INSERT INTO %s (event_data, created_at, price) VALUES (?, ?, ?)`, testTableName)
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		price := float64(i+1) * 10.25 // 10.25, 20.50, 30.75, 41.00, 51.25
		_, err := db.Exec(insertSQL, fmt.Sprintf("event-%d", i), now.Add(time.Duration(i)*time.Second), price)
		require.NoError(t, err)
	}
}

func cleanupTestTable(t *testing.T, db *sql.DB, driver string) {
	t.Helper()
	_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", testTableName))
	if err != nil {
		t.Logf("Warning: failed to cleanup test table: %v", err)
	}
}

func testIntegerCursor(t *testing.T, driver, dsn string) {
	t.Helper()

	// Set up temp paths for cursor store
	testPaths := createTestPaths(t)

	query := fmt.Sprintf("SELECT id, event_data FROM %s WHERE id > :cursor ORDER BY id ASC LIMIT 3", testTableName)

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              driver,
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "id",
		"cursor.type":         cursor.CursorTypeInteger,
		"cursor.default":      "0",
	}

	// First fetch - should get first 3 rows (id 1-3)
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1)
	require.Len(t, events1, 3, "First fetch should return 3 events")

	// Log the IDs
	for _, event := range events1 {
		if id, ok := event.MetricSetFields["id"]; ok {
			t.Logf("First fetch: id=%v", id)
		}
	}

	// Close the first metricset to persist state
	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Second fetch - should get remaining rows (id 4-5)
	ms2 := newMetricSetWithPaths(t, cfg, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2)
	require.Len(t, events2, 2, "Second fetch should return 2 events")

	// Log the IDs
	for _, event := range events2 {
		if id, ok := event.MetricSetFields["id"]; ok {
			t.Logf("Second fetch: id=%v", id)
		}
	}

	// Close second metricset
	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Third fetch - should get no rows
	ms3 := newMetricSetWithPaths(t, cfg, testPaths)
	events3, errs3 := fetchEvents(t, ms3)
	require.Empty(t, errs3)
	require.Len(t, events3, 0, "Third fetch should return 0 events")

	if closer, ok := ms3.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

func testTimestampCursor(t *testing.T, driver, dsn string) {
	t.Helper()

	// Set up temp paths for cursor store
	testPaths := createTestPaths(t)

	// Set default to a time before our test data
	defaultTimestamp := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)

	query := fmt.Sprintf("SELECT id, event_data, created_at FROM %s WHERE created_at > :cursor ORDER BY created_at ASC LIMIT 3", testTableName)

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              driver,
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "created_at",
		"cursor.type":         cursor.CursorTypeTimestamp,
		"cursor.default":      defaultTimestamp,
	}

	// First fetch - should get first 3 rows
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1)
	require.Len(t, events1, 3, "First fetch should return 3 events")

	// Close first metricset to persist state
	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Second fetch - should get remaining rows
	ms2 := newMetricSetWithPaths(t, cfg, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2)
	require.Len(t, events2, 2, "Second fetch should return 2 events")

	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

func testFloatCursor(t *testing.T, driver, dsn string) {
	t.Helper()

	testPaths := createTestPaths(t)

	// Scores are: 1.5, 3.0, 4.5, 6.0, 7.5
	// With cursor default 0.0 and LIMIT 3, first fetch gets 1.5, 3.0, 4.5
	query := fmt.Sprintf("SELECT id, event_data, score FROM %s WHERE score > :cursor ORDER BY score ASC LIMIT 3", testTableName)

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              driver,
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "score",
		"cursor.type":         cursor.CursorTypeFloat,
		"cursor.default":      "0.0",
	}

	// First fetch - should get 3 rows (scores 1.5, 3.0, 4.5)
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1)
	require.Len(t, events1, 3, "First float fetch should return 3 events")

	for _, event := range events1 {
		if s, ok := event.MetricSetFields["score"]; ok {
			t.Logf("Float cursor - First fetch: score=%v", s)
		}
	}

	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Second fetch - should get remaining 2 rows (scores 6.0, 7.5)
	ms2 := newMetricSetWithPaths(t, cfg, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2)
	require.Len(t, events2, 2, "Second float fetch should return 2 events")

	for _, event := range events2 {
		if s, ok := event.MetricSetFields["score"]; ok {
			t.Logf("Float cursor - Second fetch: score=%v", s)
		}
	}

	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Third fetch - should get no rows
	ms3 := newMetricSetWithPaths(t, cfg, testPaths)
	events3, errs3 := fetchEvents(t, ms3)
	require.Empty(t, errs3)
	require.Len(t, events3, 0, "Third float fetch should return 0 events")

	if closer, ok := ms3.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

func testDecimalCursor(t *testing.T, driver, dsn string) {
	t.Helper()

	testPaths := createTestPaths(t)

	// Prices are: 10.25, 20.50, 30.75, 41.00, 51.25
	// With cursor default 0.00 and LIMIT 3, first fetch gets 10.25, 20.50, 30.75
	query := fmt.Sprintf("SELECT id, event_data, price FROM %s WHERE price > :cursor ORDER BY price ASC LIMIT 3", testTableName)

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              driver,
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "price",
		"cursor.type":         cursor.CursorTypeDecimal,
		"cursor.default":      "0.00",
	}

	// First fetch - should get 3 rows
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1)
	require.Len(t, events1, 3, "First decimal fetch should return 3 events")

	for _, event := range events1 {
		if p, ok := event.MetricSetFields["price"]; ok {
			t.Logf("Decimal cursor - First fetch: price=%v", p)
		}
	}

	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Second fetch - should get remaining 2 rows
	ms2 := newMetricSetWithPaths(t, cfg, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2)
	require.Len(t, events2, 2, "Second decimal fetch should return 2 events")

	for _, event := range events2 {
		if p, ok := event.MetricSetFields["price"]; ok {
			t.Logf("Decimal cursor - Second fetch: price=%v", p)
		}
	}

	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Third fetch - should get no rows
	ms3 := newMetricSetWithPaths(t, cfg, testPaths)
	events3, errs3 := fetchEvents(t, ms3)
	require.Empty(t, errs3)
	require.Len(t, events3, 0, "Third decimal fetch should return 0 events")

	if closer, ok := ms3.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

func testDescendingIntegerCursor(t *testing.T, driver, dsn string) {
	t.Helper()

	testPaths := createTestPaths(t)

	// IDs are 1-5. With descending scan, cursor starts high and works down.
	// Default 999999, first fetch gets ids 5, 4, 3 (ORDER BY id DESC LIMIT 3)
	query := fmt.Sprintf("SELECT id, event_data FROM %s WHERE id < :cursor ORDER BY id DESC LIMIT 3", testTableName)

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              driver,
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "id",
		"cursor.type":         cursor.CursorTypeInteger,
		"cursor.default":      "999999",
		"cursor.direction":    "desc",
	}

	// First fetch - should get 3 rows (ids 5, 4, 3)
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1)
	require.Len(t, events1, 3, "First descending fetch should return 3 events")

	for _, event := range events1 {
		if id, ok := event.MetricSetFields["id"]; ok {
			t.Logf("Descending cursor - First fetch: id=%v", id)
		}
	}

	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Second fetch - cursor should be at min(5,4,3)=3, so fetch ids < 3 → ids 2, 1
	ms2 := newMetricSetWithPaths(t, cfg, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2)
	require.Len(t, events2, 2, "Second descending fetch should return 2 events")

	for _, event := range events2 {
		if id, ok := event.MetricSetFields["id"]; ok {
			t.Logf("Descending cursor - Second fetch: id=%v", id)
		}
	}

	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Third fetch - cursor should be at min(2,1)=1, so fetch ids < 1 → empty
	ms3 := newMetricSetWithPaths(t, cfg, testPaths)
	events3, errs3 := fetchEvents(t, ms3)
	require.Empty(t, errs3)
	require.Len(t, events3, 0, "Third descending fetch should return 0 events")

	if closer, ok := ms3.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

func testCompoundWhereCursor(t *testing.T, driver, dsn string) {
	t.Helper()

	testPaths := createTestPaths(t)

	// Real-world pattern: cursor combined with an additional filter condition.
	// The table has 5 rows with event_data: event-0, event-1, event-2, event-3, event-4.
	// We filter for event_data LIKE 'event-%' (matches all) AND id > :cursor.
	// This verifies :cursor works correctly when it's not the only WHERE condition.
	query := fmt.Sprintf(
		"SELECT id, event_data FROM %s WHERE id > :cursor AND event_data LIKE 'event-%%' ORDER BY id ASC LIMIT 3",
		testTableName)

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              driver,
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "id",
		"cursor.type":         cursor.CursorTypeInteger,
		"cursor.default":      "0",
	}

	// First fetch - 3 rows
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1)
	require.Len(t, events1, 3, "First fetch with compound WHERE should return 3 events")

	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Second fetch - remaining 2 rows
	ms2 := newMetricSetWithPaths(t, cfg, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2)
	require.Len(t, events2, 2, "Second fetch with compound WHERE should return 2 events")

	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Third fetch - 0 rows
	ms3 := newMetricSetWithPaths(t, cfg, testPaths)
	events3, errs3 := fetchEvents(t, ms3)
	require.Empty(t, errs3)
	require.Len(t, events3, 0, "Third fetch with compound WHERE should return 0 events")

	if closer, ok := ms3.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

// TestCursorStatePersistence verifies cursor state survives restarts
func TestCursorStatePersistence(t *testing.T) {
	service := compose.EnsureUp(t, "postgresql")
	host, port, err := net.SplitHostPort(service.Host())
	require.NoError(t, err)

	user := postgresql.GetEnvUsername()
	password := postgresql.GetEnvPassword()
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/?sslmode=disable", user, password, host, port)

	// Set up test table
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	setupPostgresTestTable(t, db)
	defer cleanupTestTable(t, db, "postgres")

	// Set up temp paths - we need to track tmpDir to check file existence
	testPaths := createTestPaths(t)

	query := fmt.Sprintf("SELECT id, event_data FROM %s WHERE id > :cursor ORDER BY id ASC", testTableName)

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "postgres",
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "id",
		"cursor.type":         cursor.CursorTypeInteger,
		"cursor.default":      "0",
	}

	// First fetch - get all 5 rows
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1)
	require.Len(t, events1, 5, "Should get all 5 rows")

	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Verify cursor state file exists
	cursorDir := filepath.Join(testPaths.Data, "sql-cursor")
	_, statErr := os.Stat(cursorDir)
	assert.NoError(t, statErr, "Cursor state directory should exist")

	// Create new metricset - cursor should be loaded from state
	ms2 := newMetricSetWithPaths(t, cfg, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2)
	require.Len(t, events2, 0, "Should get 0 rows after cursor loaded from state")

	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

// TestCursorNullValues verifies handling of NULL values in cursor column
func TestCursorNullValues(t *testing.T) {
	service := compose.EnsureUp(t, "postgresql")
	host, port, err := net.SplitHostPort(service.Host())
	require.NoError(t, err)

	user := postgresql.GetEnvUsername()
	password := postgresql.GetEnvPassword()
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/?sslmode=disable", user, password, host, port)

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	// Create table with nullable timestamp
	tableName := "cursor_null_test"
	_, err = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	require.NoError(t, err)

	_, err = db.Exec(fmt.Sprintf(`
		CREATE TABLE %s (
			id SERIAL PRIMARY KEY,
			event_data TEXT,
			updated_at TIMESTAMP
		)
	`, tableName))
	require.NoError(t, err)
	defer func() { _, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)) }()

	// Insert data with NULL timestamps
	now := time.Now().UTC()
	_, err = db.Exec(fmt.Sprintf(`INSERT INTO %s (event_data, updated_at) VALUES ($1, $2)`, tableName), "event-1", now)
	require.NoError(t, err)
	_, err = db.Exec(fmt.Sprintf(`INSERT INTO %s (event_data, updated_at) VALUES ($1, NULL)`, tableName), "event-2-null")
	require.NoError(t, err)
	_, err = db.Exec(fmt.Sprintf(`INSERT INTO %s (event_data, updated_at) VALUES ($1, $2)`, tableName), "event-3", now.Add(time.Second))
	require.NoError(t, err)

	// Set up temp paths for cursor store
	testPaths := createTestPaths(t)

	defaultTimestamp := now.Add(-time.Hour).Format(time.RFC3339)
	query := fmt.Sprintf("SELECT id, event_data, updated_at FROM %s WHERE updated_at > :cursor OR updated_at IS NULL ORDER BY id ASC", tableName)

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "postgres",
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "updated_at",
		"cursor.type":         cursor.CursorTypeTimestamp,
		"cursor.default":      defaultTimestamp,
	}

	ms := newMetricSetWithPaths(t, cfg, testPaths)
	events, errs := fetchEvents(t, ms)
	require.Empty(t, errs)

	// Should get all 3 rows even though one has NULL
	require.Len(t, events, 3, "Should get all 3 events including NULL one")

	if closer, ok := ms.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

// fetchEvents is a helper to fetch events from a MetricSet
func fetchEvents(t *testing.T, ms mb.MetricSet) ([]mb.Event, []error) {
	t.Helper()

	switch v := ms.(type) {
	case mb.ReportingMetricSetV2WithContext:
		return mbtest.ReportingFetchV2WithContext(v)
	case mb.ReportingMetricSetV2Error:
		return mbtest.ReportingFetchV2Error(v)
	case mb.ReportingMetricSetV2:
		return mbtest.ReportingFetchV2(v)
	default:
		t.Fatalf("unknown metricset type: %T", ms)
		return nil, nil
	}
}

// ============================================================================
// ORACLE CURSOR TESTS
// ============================================================================

// TestOracleCursor tests cursor functionality with Oracle database.
//
// This is a comprehensive test covering:
//   - Integer cursor: first run, subsequent runs, restart (state persistence)
//   - Timestamp cursor: multi-batch incremental fetch, timezone handling, precision
//   - Date cursor: Oracle DATE column handling
//   - NULL handling: NULL values in cursor column
//   - Empty result set: no matching rows
//   - Query change resets cursor: different query → new cursor state
//   - Oracle driver type conversions: NUMBER, TIMESTAMP, DATE → Go types
//
// The timestamp multi-batch test is particularly important as it verifies the fix
// for the Oracle timestamp cursor bug where subsequent fetches returned 0 rows
// because the godror driver didn't properly bind time.Time values to the
// :cursor_val named parameter without sql.Named().
func TestOracleCursor(t *testing.T) {
	// Skip if Oracle Instant Client is not installed.
	// The godror driver requires the Oracle Instant Client library (libclntsh.dylib/so).
	// See: https://oracle.github.io/odpi/doc/installation.html
	testDB, err := sql.Open("godror", "user/pass@localhost:1521/test")
	if err == nil {
		err = testDB.Ping()
		testDB.Close()
	}
	if err != nil && containsOracleClientError(err.Error()) {
		t.Skip("Skipping Oracle cursor tests: Oracle Instant Client not installed. " +
			"See https://oracle.github.io/odpi/doc/installation.html")
	}

	service := compose.EnsureUp(t, "oracle")
	host, port, err := net.SplitHostPort(service.Host())
	require.NoError(t, err)

	waitForOracleConnection(t, host, port)

	dsn := GetOracleConnectionDetails(t, host, port)

	db, err := sql.Open("godror", dsn)
	require.NoError(t, err, "Failed to connect to Oracle")
	defer db.Close()

	setupOracleTestTable(t, db)
	defer cleanupOracleTestTable(t, db)

	// --- Integer cursor tests ---
	t.Run("integer_cursor_first_and_subsequent_runs", func(t *testing.T) {
		testOracleIntegerCursor(t, dsn)
	})

	t.Run("integer_cursor_restart_preserves_state", func(t *testing.T) {
		testOracleIntegerCursorRestart(t, dsn)
	})

	// --- Timestamp cursor tests ---
	t.Run("timestamp_cursor_multi_batch", func(t *testing.T) {
		testOracleTimestampCursorMultiBatch(t, dsn)
	})

	t.Run("timestamp_cursor_timezone_handling", func(t *testing.T) {
		testOracleTimestampTimezoneHandling(t, dsn)
	})

	t.Run("timestamp_cursor_precision", func(t *testing.T) {
		testOracleTimestampPrecision(t, dsn)
	})

	// --- Date cursor test ---
	t.Run("date_cursor", func(t *testing.T) {
		testOracleDateCursor(t, dsn)
	})

	// --- Edge case tests ---
	t.Run("null_handling", func(t *testing.T) {
		testOracleNullHandling(t, db, dsn)
	})

	t.Run("empty_result_set", func(t *testing.T) {
		testOracleEmptyResultSet(t, dsn)
	})

	t.Run("query_change_resets_cursor", func(t *testing.T) {
		testOracleQueryChangeResetsCursor(t, dsn)
	})

	// --- Timezone mismatch test (reproduces reported bug) ---
	t.Run("timestamp_cursor_timezone_mismatch", func(t *testing.T) {
		testOracleTimestampCursorTimezoneMismatch(t, host, port)
	})

	// --- Oracle-specific tests ---
	t.Run("driver_type_conversions", func(t *testing.T) {
		testOracleDriverTypeConversions(t, db)
	})
}

func setupOracleTestTable(t *testing.T, db *sql.DB) {
	t.Helper()

	// Drop table if exists (Oracle doesn't have IF EXISTS, so we ignore errors)
	_, _ = db.Exec("DROP TABLE cursor_test_events")

	// Create table with Oracle-specific syntax.
	// Uses TIMESTAMP (not TIMESTAMP WITH TIME ZONE) and DATE to test
	// Oracle-specific type handling through the godror driver.
	createSQL := `
		CREATE TABLE cursor_test_events (
			id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			event_data VARCHAR2(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			event_date DATE DEFAULT CURRENT_DATE
		)
	`
	_, err := db.Exec(createSQL)
	require.NoError(t, err, "Failed to create Oracle test table")

	// Insert 10 rows with timestamps 1 second apart.
	// This gives enough data to test multi-batch pagination (e.g., 3+3+3+1+0).
	insertSQL := `INSERT INTO cursor_test_events (event_data, created_at, event_date) VALUES (:1, :2, :3)`
	now := time.Now().UTC()
	for i := 0; i < 10; i++ {
		ts := now.Add(time.Duration(i) * time.Second)
		_, err := db.Exec(insertSQL, fmt.Sprintf("event-%d", i), ts, ts)
		require.NoError(t, err, "Failed to insert Oracle test data row %d", i)
	}
	t.Log("Oracle test table created with 10 rows")
}

func cleanupOracleTestTable(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("DROP TABLE cursor_test_events")
	if err != nil {
		t.Logf("Warning: failed to cleanup Oracle test table: %v", err)
	}
}

// testOracleIntegerCursor verifies integer cursor pagination with Oracle.
// With 10 rows and FETCH FIRST 3, the pattern is: 3 + 3 + 3 + 1 + 0.
// Uses :cursor named parameter binding via :cursor_val.
func testOracleIntegerCursor(t *testing.T, dsn string) {
	t.Helper()

	testPaths := createTestPaths(t)

	// :cursor is translated to :cursor_val for Oracle by cursor.TranslateQuery
	query := "SELECT id, event_data FROM cursor_test_events WHERE id > :cursor ORDER BY id ASC FETCH FIRST 3 ROWS ONLY"

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "oracle",
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "id",
		"cursor.type":         cursor.CursorTypeInteger,
		"cursor.default":      "0",
	}

	// First fetch - ids 1-3
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1, "First fetch should not have errors")
	require.Len(t, events1, 3, "First fetch should return 3 events")
	t.Logf("Oracle integer cursor - First fetch: %d events", len(events1))
	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Second fetch - ids 4-6
	ms2 := newMetricSetWithPaths(t, cfg, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2, "Second fetch should not have errors")
	require.Len(t, events2, 3, "Second fetch should return 3 events")
	t.Logf("Oracle integer cursor - Second fetch: %d events", len(events2))
	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Third fetch - ids 7-9
	ms3 := newMetricSetWithPaths(t, cfg, testPaths)
	events3, errs3 := fetchEvents(t, ms3)
	require.Empty(t, errs3, "Third fetch should not have errors")
	require.Len(t, events3, 3, "Third fetch should return 3 events")
	if closer, ok := ms3.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Fourth fetch - id 10 (last row)
	ms4 := newMetricSetWithPaths(t, cfg, testPaths)
	events4, errs4 := fetchEvents(t, ms4)
	require.Empty(t, errs4, "Fourth fetch should not have errors")
	require.Len(t, events4, 1, "Fourth fetch should return 1 event (last row)")
	if closer, ok := ms4.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Fifth fetch - all consumed, should return 0
	ms5 := newMetricSetWithPaths(t, cfg, testPaths)
	events5, errs5 := fetchEvents(t, ms5)
	require.Empty(t, errs5, "Fifth fetch should not have errors")
	require.Len(t, events5, 0, "Fifth fetch should return 0 events (all consumed)")
	if closer, ok := ms5.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

// testOracleIntegerCursorRestart verifies that cursor state persists across restarts.
// Fetches all rows, closes MetricSet, re-creates it with same paths, and verifies
// the cursor was loaded from persisted state (0 new rows returned).
func testOracleIntegerCursorRestart(t *testing.T, dsn string) {
	t.Helper()

	testPaths := createTestPaths(t)

	// No FETCH FIRST — get all rows in one go
	query := "SELECT id, event_data FROM cursor_test_events WHERE id > :cursor ORDER BY id ASC"

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "oracle",
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "id",
		"cursor.type":         cursor.CursorTypeInteger,
		"cursor.default":      "0",
	}

	// First fetch - get all 10 rows
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1)
	require.Len(t, events1, 10, "Should get all 10 rows")
	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Verify cursor state directory exists (state was persisted)
	cursorDir := filepath.Join(testPaths.Data, "sql-cursor")
	_, statErr := os.Stat(cursorDir)
	assert.NoError(t, statErr, "Cursor state directory should exist after close")

	// "Restart" — create new MetricSet with same paths (simulates process restart)
	ms2 := newMetricSetWithPaths(t, cfg, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2)
	require.Len(t, events2, 0, "After restart, should get 0 rows (cursor loaded from persisted state)")
	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

// testOracleTimestampCursorMultiBatch is the critical test for the Oracle timestamp
// cursor bug. Without the sql.Named() fix for the :cursor_val named parameter,
// the godror driver fails to properly bind time.Time values, causing all fetches
// after the first to silently return 0 rows.
//
// With 10 rows and FETCH FIRST 3, the expected pattern is: 3 + 3 + 3 + 1 + 0.
func testOracleTimestampCursorMultiBatch(t *testing.T, dsn string) {
	t.Helper()

	testPaths := createTestPaths(t)
	defaultTimestamp := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)

	query := "SELECT id, event_data, created_at FROM cursor_test_events WHERE created_at > :cursor ORDER BY created_at ASC FETCH FIRST 3 ROWS ONLY"

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "oracle",
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "created_at",
		"cursor.type":         cursor.CursorTypeTimestamp,
		"cursor.default":      defaultTimestamp,
	}

	// First fetch — should get 3 rows
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1, "First timestamp fetch should not have errors")
	require.Len(t, events1, 3, "First timestamp fetch should return 3 events")
	t.Logf("Oracle timestamp cursor - First fetch: %d events", len(events1))
	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Second fetch — CRITICAL: should get next 3 rows, NOT 0.
	// This is the assertion that catches the Oracle timestamp cursor bug.
	// Without sql.Named(), godror doesn't properly bind the cursor time.Time
	// to the :cursor_val named parameter, causing 0 rows to be returned.
	ms2 := newMetricSetWithPaths(t, cfg, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2, "Second timestamp fetch should not have errors")
	require.Len(t, events2, 3,
		"Second timestamp fetch should return 3 events; "+
			"got 0 indicates the Oracle timestamp :cursor_val bind parameter is broken — "+
			"the named parameter needs sql.Named() for godror driver")
	t.Logf("Oracle timestamp cursor - Second fetch: %d events", len(events2))
	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Third fetch — should get 3 more rows
	ms3 := newMetricSetWithPaths(t, cfg, testPaths)
	events3, errs3 := fetchEvents(t, ms3)
	require.Empty(t, errs3, "Third timestamp fetch should not have errors")
	require.Len(t, events3, 3, "Third timestamp fetch should return 3 events")
	if closer, ok := ms3.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Fourth fetch — last row
	ms4 := newMetricSetWithPaths(t, cfg, testPaths)
	events4, errs4 := fetchEvents(t, ms4)
	require.Empty(t, errs4, "Fourth timestamp fetch should not have errors")
	require.Len(t, events4, 1, "Fourth timestamp fetch should return 1 event (last row)")
	if closer, ok := ms4.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Fifth fetch — all consumed
	ms5 := newMetricSetWithPaths(t, cfg, testPaths)
	events5, errs5 := fetchEvents(t, ms5)
	require.Empty(t, errs5, "Fifth timestamp fetch should not have errors")
	require.Len(t, events5, 0, "Fifth timestamp fetch should return 0 events (all consumed)")
	if closer, ok := ms5.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

// testOracleTimestampTimezoneHandling verifies that Oracle timestamps are
// correctly handled in UTC regardless of the database timezone setting.
// The Oracle session is configured with TIME_ZONE='UTC' via GetOracleConnectionDetails.
func testOracleTimestampTimezoneHandling(t *testing.T, dsn string) {
	t.Helper()

	testPaths := createTestPaths(t)
	// Use a UTC timestamp as default — the session timezone is set to UTC
	defaultTimestamp := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)

	query := "SELECT id, event_data, created_at FROM cursor_test_events WHERE created_at > :cursor ORDER BY created_at ASC FETCH FIRST 5 ROWS ONLY"

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "oracle",
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "created_at",
		"cursor.type":         cursor.CursorTypeTimestamp,
		"cursor.default":      defaultTimestamp,
	}

	// First fetch — 5 rows
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1)
	require.Len(t, events1, 5, "First fetch with UTC timezone should return 5 events")
	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Second fetch — remaining 5 rows (verifies cursor updated correctly across timezone boundary)
	ms2 := newMetricSetWithPaths(t, cfg, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2)
	require.Len(t, events2, 5, "Second fetch with UTC timezone should return 5 events")
	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Third fetch — all consumed
	ms3 := newMetricSetWithPaths(t, cfg, testPaths)
	events3, errs3 := fetchEvents(t, ms3)
	require.Empty(t, errs3)
	require.Len(t, events3, 0, "Third fetch should return 0 events (all consumed)")
	if closer, ok := ms3.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

// testOracleTimestampPrecision verifies that Oracle TIMESTAMP sub-second
// precision is correctly preserved through the cursor round-trip.
// Fetches rows one-at-a-time to verify each cursor update is precise enough
// to skip exactly the current row and return the next.
func testOracleTimestampPrecision(t *testing.T, dsn string) {
	t.Helper()

	testPaths := createTestPaths(t)
	defaultTimestamp := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339Nano)

	// FETCH FIRST 1 ROW ONLY to test single-row pagination
	query := "SELECT id, event_data, created_at FROM cursor_test_events WHERE created_at > :cursor ORDER BY created_at ASC FETCH FIRST 1 ROWS ONLY"

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "oracle",
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "created_at",
		"cursor.type":         cursor.CursorTypeTimestamp,
		"cursor.default":      defaultTimestamp,
	}

	// Fetch rows one at a time — precision loss would cause rows to be
	// skipped (cursor jumps too far) or re-fetched (cursor doesn't advance).
	var totalFetched int
	for i := 0; i < 12; i++ { // 10 rows + 2 safety iterations
		ms := newMetricSetWithPaths(t, cfg, testPaths)
		events, errs := fetchEvents(t, ms)
		require.Empty(t, errs, "Fetch %d should not have errors", i+1)
		if closer, ok := ms.(mb.Closer); ok {
			require.NoError(t, closer.Close())
		}
		if len(events) == 0 {
			break
		}
		require.Len(t, events, 1, "Fetch %d should return exactly 1 event", i+1)
		totalFetched += len(events)
	}

	require.Equal(t, 10, totalFetched,
		"Should fetch all 10 rows one-at-a-time; precision loss would cause rows to be skipped or re-fetched")
}

// testOracleDateCursor verifies cursor operation on Oracle DATE columns.
// Oracle DATE includes time component (unlike SQL standard DATE), so the
// cursor uses TO_DATE(:cursor_val, 'YYYY-MM-DD') for proper comparison.
func testOracleDateCursor(t *testing.T, dsn string) {
	t.Helper()

	testPaths := createTestPaths(t)
	defaultDate := time.Now().Add(-24 * time.Hour).UTC().Format("2006-01-02")

	query := "SELECT id, event_data, event_date FROM cursor_test_events WHERE event_date > TO_DATE(:cursor, 'YYYY-MM-DD') ORDER BY event_date ASC, id ASC FETCH FIRST 3 ROWS ONLY"

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "oracle",
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "event_date",
		"cursor.type":         cursor.CursorTypeDate,
		"cursor.default":      defaultDate,
	}

	// First fetch — all 10 rows have today's date, which is > yesterday's default
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1, "First date fetch should not have errors")
	require.Len(t, events1, 3, "First date fetch should return 3 events")
	t.Logf("Oracle date cursor - First fetch: %d events", len(events1))
	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Second fetch — since all rows share the same date (today), the cursor
	// value is today's date. The query WHERE event_date > today returns 0 rows
	// because there are no future-dated rows. This is expected behavior for date
	// cursors when all data shares the same date — timestamp cursors handle
	// intra-day ordering better.
	ms2 := newMetricSetWithPaths(t, cfg, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2, "Second date fetch should not have errors")
	t.Logf("Oracle date cursor - Second fetch: %d events", len(events2))
	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

// testOracleNullHandling verifies that NULL values in the cursor column
// don't cause errors and are properly skipped during cursor updates.
func testOracleNullHandling(t *testing.T, db *sql.DB, dsn string) {
	t.Helper()

	// Create a separate table with NULL values
	tableName := "cursor_null_test_oracle"
	_, _ = db.Exec(fmt.Sprintf("DROP TABLE %s", tableName))

	_, err := db.Exec(fmt.Sprintf(`
		CREATE TABLE %s (
			id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			event_data VARCHAR2(255),
			updated_at TIMESTAMP
		)
	`, tableName))
	require.NoError(t, err, "Failed to create NULL test table")
	defer func() { _, _ = db.Exec(fmt.Sprintf("DROP TABLE %s", tableName)) }()

	// Insert rows: some with timestamps, some with NULL
	now := time.Now().UTC()
	_, err = db.Exec(fmt.Sprintf("INSERT INTO %s (event_data, updated_at) VALUES (:1, :2)", tableName), "event-1", now)
	require.NoError(t, err)
	_, err = db.Exec(fmt.Sprintf("INSERT INTO %s (event_data, updated_at) VALUES (:1, NULL)", tableName), "event-2-null")
	require.NoError(t, err)
	_, err = db.Exec(fmt.Sprintf("INSERT INTO %s (event_data, updated_at) VALUES (:1, :2)", tableName), "event-3", now.Add(time.Second))
	require.NoError(t, err)

	testPaths := createTestPaths(t)
	defaultTimestamp := now.Add(-time.Hour).Format(time.RFC3339)

	// Query includes OR updated_at IS NULL to also fetch NULL rows
	query := fmt.Sprintf(
		"SELECT id, event_data, updated_at FROM %s WHERE updated_at > :cursor OR updated_at IS NULL ORDER BY id ASC",
		tableName)

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "oracle",
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "updated_at",
		"cursor.type":         cursor.CursorTypeTimestamp,
		"cursor.default":      defaultTimestamp,
	}

	ms := newMetricSetWithPaths(t, cfg, testPaths)
	events, errs := fetchEvents(t, ms)
	require.Empty(t, errs)
	// Should get all 3 rows: 2 with timestamps + 1 with NULL
	require.Len(t, events, 3, "Should get all 3 events including NULL timestamp row")
	if closer, ok := ms.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

// testOracleEmptyResultSet verifies that an empty result set (no matching rows)
// is handled correctly: no errors, 0 events, cursor unchanged.
func testOracleEmptyResultSet(t *testing.T, dsn string) {
	t.Helper()

	testPaths := createTestPaths(t)

	// Use a far-future default so no rows match
	farFuture := time.Now().Add(100 * 365 * 24 * time.Hour).UTC().Format(time.RFC3339)

	query := "SELECT id, event_data, created_at FROM cursor_test_events WHERE created_at > :cursor ORDER BY created_at ASC FETCH FIRST 3 ROWS ONLY"

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "oracle",
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "created_at",
		"cursor.type":         cursor.CursorTypeTimestamp,
		"cursor.default":      farFuture,
	}

	ms := newMetricSetWithPaths(t, cfg, testPaths)
	events, errs := fetchEvents(t, ms)
	require.Empty(t, errs, "Empty result set should not cause errors")
	require.Len(t, events, 0, "Should return 0 events when cursor is in the far future")

	// Verify cursor remains unchanged after empty result
	queryMs, ok := ms.(*MetricSet)
	require.True(t, ok)

	// ParseValue normalizes the format, so compare normalized values
	normalizedFarFuture, err := cursor.ParseValue(farFuture, cursor.CursorTypeTimestamp)
	require.NoError(t, err)
	assert.Equal(t, normalizedFarFuture.String(), queryMs.cursorManager.CursorValueString(),
		"Cursor should remain at far-future default after empty result set")

	if closer, ok := ms.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

// testOracleQueryChangeResetsCursor verifies that changing the SQL query causes
// the cursor state key to change (because the query is part of the key hash),
// effectively resetting the cursor to the default value.
func testOracleQueryChangeResetsCursor(t *testing.T, dsn string) {
	t.Helper()

	testPaths := createTestPaths(t)

	// First query — fetch 5 rows
	query1 := "SELECT id, event_data FROM cursor_test_events WHERE id > :cursor ORDER BY id ASC FETCH FIRST 5 ROWS ONLY"

	cfg1 := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "oracle",
		"sql_query":           query1,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "id",
		"cursor.type":         cursor.CursorTypeInteger,
		"cursor.default":      "0",
	}

	ms1 := newMetricSetWithPaths(t, cfg1, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1)
	require.Len(t, events1, 5, "First query should return 5 events")
	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Change the query (different FETCH FIRST) — this generates a different
	// state key hash, so the cursor resets to default.
	query2 := "SELECT id, event_data FROM cursor_test_events WHERE id > :cursor ORDER BY id ASC FETCH FIRST 3 ROWS ONLY"

	cfg2 := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "oracle",
		"sql_query":           query2,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "id",
		"cursor.type":         cursor.CursorTypeInteger,
		"cursor.default":      "0",
	}

	// Should start from default (0) because the query changed
	ms2 := newMetricSetWithPaths(t, cfg2, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2)
	require.Len(t, events2, 3,
		"Changed query should start from default cursor (0), returning first 3 events")
	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

// testOracleDriverTypeConversions verifies that Oracle column types are correctly
// handled through the cursor pipeline. This tests the full round-trip:
// Oracle type → godror Go type → getValue() → FromDatabaseValue/ParseValue.
func testOracleDriverTypeConversions(t *testing.T, db *sql.DB) {
	t.Helper()

	// Query a row to inspect godror's type mapping for Oracle columns
	rows, err := db.Query("SELECT id, created_at, event_date FROM cursor_test_events WHERE ROWNUM <= 1")
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next(), "Should have at least one row")

	// Log Oracle column types for debugging
	cols, err := rows.ColumnTypes()
	require.NoError(t, err)
	for _, col := range cols {
		t.Logf("Oracle column mapping: %s → DatabaseTypeName=%s, ScanType=%v",
			col.Name(), col.DatabaseTypeName(), col.ScanType())
	}

	var id, createdAt, eventDate interface{}
	err = rows.Scan(&id, &createdAt, &eventDate)
	require.NoError(t, err)

	t.Logf("Oracle NUMBER (id) → Go type: %T, value: %v", id, id)
	t.Logf("Oracle TIMESTAMP (created_at) → Go type: %T, value: %v", createdAt, createdAt)
	t.Logf("Oracle DATE (event_date) → Go type: %T, value: %v", eventDate, eventDate)

	// Verify Oracle TIMESTAMP → time.Time (which godror returns natively).
	// This is important because the cursor pipeline relies on getValue()
	// converting time.Time to RFC3339Nano string for mapstr.M storage.
	if ts, ok := createdAt.(time.Time); ok {
		formatted := ts.Format(time.RFC3339Nano)
		val, err := cursor.ParseValue(formatted, cursor.CursorTypeTimestamp)
		assert.NoError(t, err,
			"Oracle TIMESTAMP → time.Time → RFC3339Nano should round-trip through cursor.ParseValue")
		if err == nil {
			// Verify ToDriverArg returns time.Time (which godror needs for TIMESTAMP binding)
			driverArg := val.ToDriverArg()
			_, isTime := driverArg.(time.Time)
			assert.True(t, isTime,
				"cursor.ToDriverArg() for timestamp should return time.Time, got %T", driverArg)
		}
	} else {
		t.Logf("Note: Oracle TIMESTAMP scanned as %T (not time.Time); cursor pipeline handles this via getValue()", createdAt)
	}

	// Verify Oracle DATE → time.Time round-trip
	if dt, ok := eventDate.(time.Time); ok {
		formatted := dt.Format("2006-01-02")
		_, err := cursor.ParseValue(formatted, cursor.CursorTypeDate)
		assert.NoError(t, err,
			"Oracle DATE → time.Time → date string should round-trip through cursor.ParseValue")
	} else {
		t.Logf("Note: Oracle DATE scanned as %T (not time.Time); cursor pipeline handles this via getValue()", eventDate)
	}

	// Verify Oracle NUMBER round-trip through fmt.Sprint (which getValue uses for unknown types)
	idStr := fmt.Sprint(id)
	_, err = cursor.ParseValue(idStr, cursor.CursorTypeInteger)
	assert.NoError(t, err,
		"Oracle NUMBER → fmt.Sprint(%T) → %q should be parseable as integer cursor value", id, idStr)
}

// getOracleDSNWithTimezone builds an Oracle DSN with explicit timezone settings.
// sessionTZ is the Oracle session TIME_ZONE (e.g., "+05:00", "US/Eastern").
// goTZ is the timezone godror uses to interpret TIMESTAMP values from Oracle.
func getOracleDSNWithTimezone(t *testing.T, host, port, sessionTZ string, goTZ *time.Location) string {
	t.Helper()
	connectString := GetOracleConnectString(host, port)
	params, err := godror.ParseDSN(connectString)
	require.NoError(t, err, "Failed to parse Oracle DSN: %s", connectString)
	params.AlterSession = append(params.AlterSession, [2]string{"TIME_ZONE", sessionTZ})
	params.Timezone = goTZ
	return params.StringWithPassword()
}

// testOracleTimestampCursorTimezoneMismatch reproduces the Oracle timestamp cursor
// bug that occurs when there is a timezone mismatch between the Oracle session
// timezone and the timezone embedded in godror's bind parameters.
//
// ## Root cause
//
// Oracle's TIMESTAMP column stores values without timezone info. The godror driver
// sends Go time.Time bind parameters as TIMESTAMP WITH TIME ZONE (OCI type
// SQLT_TIMESTAMP_TZ), carrying the UTC zone marker. When Oracle compares a stored
// TIMESTAMP against a TIMESTAMP WITH TIME ZONE bind parameter, it implicitly
// converts the stored TIMESTAMP to TIMESTAMP WITH TIME ZONE using the session
// timezone (per Oracle comparison rules). If the session timezone is not UTC,
// this conversion shifts the effective time, causing the comparison to fail.
//
// ## Reproduction
//
//  1. Set Oracle session TIME_ZONE = '+05:00', godror params.Timezone = UTC
//  2. Insert: Go sends time.Time{14:00 UTC}. godror writes to TIMESTAMP column;
//     Oracle stores the raw value 14:00 (no TZ conversion for plain TIMESTAMP).
//  3. Read: Oracle returns TIMESTAMP '14:00'. godror (Timezone=UTC) creates
//     time.Time{14:00 UTC}. Round-trip is consistent — difference is 0.
//  4. Cursor stores 14:00 UTC. ToDriverArg() returns time.Time{14:00 UTC}.
//  5. Second fetch bind: godror sends TIMESTAMP WITH TIME ZONE '14:00 UTC'.
//     Oracle compares stored TIMESTAMP '14:00' by converting it using session
//     TZ: 14:00 +05:00 = 09:00 UTC. Comparison: 09:00 UTC > 14:00 UTC → FALSE.
//     Returns 0 rows.
//
// The bug does NOT manifest when the Oracle session timezone is UTC (as in the
// other tests that use GetOracleConnectionDetails which sets TIME_ZONE='UTC').
//
// ## Expected behavior (after fix)
//
// The cursor should correctly paginate through all rows regardless of timezone
// configuration. The fix should ensure that the bind parameter type matches the
// column type (plain TIMESTAMP, not TIMESTAMP WITH TIME ZONE).
func testOracleTimestampCursorTimezoneMismatch(t *testing.T, host, port string) {
	t.Helper()

	// Build a DSN with timezone MISMATCH:
	// - Oracle session timezone = '+05:00' (simulates a non-UTC database)
	// - godror interprets timestamps as UTC (simulates Metricbeat running on UTC host)
	//
	// In production, users don't call ALTER SESSION SET TIME_ZONE and godror
	// uses the local timezone. If the Oracle DB defaults to a non-UTC timezone,
	// this exact mismatch occurs.
	mismatchDSN := getOracleDSNWithTimezone(t, host, port, "+05:00", time.UTC)

	// Create a separate table for this test to avoid interfering with other tests
	tableName := "cursor_tz_mismatch_test"
	db, err := sql.Open("godror", mismatchDSN)
	require.NoError(t, err, "Failed to connect with timezone-mismatch DSN")
	defer db.Close()

	// Setup table
	_, _ = db.Exec(fmt.Sprintf("DROP TABLE %s", tableName))
	_, err = db.Exec(fmt.Sprintf(`
		CREATE TABLE %s (
			id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			event_data VARCHAR2(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`, tableName))
	require.NoError(t, err, "Failed to create timezone test table")
	defer func() { _, _ = db.Exec(fmt.Sprintf("DROP TABLE %s", tableName)) }()

	// Insert 10 rows with timestamps 1 second apart.
	// With params.Timezone=UTC, godror sends the raw UTC value and Oracle stores
	// it as-is in the plain TIMESTAMP column (no timezone conversion on storage).
	now := time.Now().UTC()
	for i := 0; i < 10; i++ {
		ts := now.Add(time.Duration(i) * time.Second)
		_, err := db.Exec(
			fmt.Sprintf("INSERT INTO %s (event_data, created_at) VALUES (:1, :2)", tableName),
			fmt.Sprintf("event-%d", i), ts)
		require.NoError(t, err, "Failed to insert row %d", i)
	}

	// Diagnostic: verify what Oracle stored and how godror reads it back.
	// The round-trip should be consistent (difference ≈ 0) because both
	// insert and read use the same params.Timezone=UTC. The mismatch only
	// manifests during comparison (WHERE clause with > operator).
	var storedTS time.Time
	err = db.QueryRow(fmt.Sprintf(
		"SELECT created_at FROM %s WHERE ROWNUM = 1 ORDER BY id", tableName)).Scan(&storedTS)
	require.NoError(t, err)
	t.Logf("Inserted Go time (UTC):  %s", now.Format(time.RFC3339Nano))
	t.Logf("Read back via godror:    %s", storedTS.Format(time.RFC3339Nano))
	t.Logf("Round-trip difference:   %v (expected ≈ 0)", storedTS.Sub(now).Truncate(time.Millisecond))

	// Diagnostic: verify the timezone mismatch affects comparisons.
	// Use a raw SQL query with a TIMESTAMP WITH TIME ZONE literal to show
	// that Oracle converts stored TIMESTAMP using session TZ for comparison.
	var countDirect int
	err = db.QueryRow(fmt.Sprintf(
		"SELECT COUNT(*) FROM %s WHERE created_at > TO_TIMESTAMP(:1, 'YYYY-MM-DD\"T\"HH24:MI:SS.FF\"Z\"')",
		tableName),
		now.Add(-1*time.Hour).Format("2006-01-02T15:04:05.000000Z")).Scan(&countDirect)
	require.NoError(t, err)
	t.Logf("Direct query (TO_TIMESTAMP, no TZ info): %d rows (should be 10)", countDirect)

	// Now run the timestamp cursor test with this mismatched DSN.
	// Use a far-past default so the first fetch succeeds even with the TZ shift.
	// Oracle converts stored TIMESTAMP '14:48:04' using session TZ +05:00 to
	// '14:48:04 +05:00' = '09:48:04 UTC'. With default '2000-01-01T00:00:00Z',
	// '09:48:04 UTC > 2000-01-01 00:00:00 UTC' → TRUE, so first fetch works.
	testPaths := createTestPaths(t)
	defaultTimestamp := "2000-01-01T00:00:00Z"

	query := fmt.Sprintf(
		"SELECT id, event_data, created_at FROM %s WHERE created_at > :cursor ORDER BY created_at ASC FETCH FIRST 3 ROWS ONLY",
		tableName)

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{mismatchDSN},
		"driver":              "oracle",
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "created_at",
		"cursor.type":         cursor.CursorTypeTimestamp,
		"cursor.default":      defaultTimestamp,
	}

	// First fetch — should get 3 rows.
	// Default cursor is year 2000, so even with the +05:00 timezone shift making
	// stored values appear ~5h earlier in UTC, all rows still satisfy the condition.
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1, "First fetch should not error")
	require.Len(t, events1, 3, "First fetch should return 3 events")
	t.Logf("TZ mismatch test - First fetch: %d events", len(events1))
	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Second fetch — THIS IS WHERE THE BUG MANIFESTS.
	//
	// After the first fetch, the cursor stores a timestamp read back by godror
	// (e.g., 14:48:06 UTC). ToDriverArg() returns time.Time{14:48:06 UTC}.
	// godror sends this as TIMESTAMP WITH TIME ZONE '14:48:06 UTC'.
	//
	// Oracle converts stored TIMESTAMP '14:48:07' (next row) using session
	// TZ (+05:00): 14:48:07 +05:00 = 09:48:07 UTC.
	//
	// Comparison: 09:48:07 UTC > 14:48:06 UTC → FALSE → 0 rows returned.
	//
	// The stored value (09:48:07 UTC after TZ conversion) appears ~5 hours
	// BEHIND the cursor value (14:48:06 UTC), so no rows match.
	ms2 := newMetricSetWithPaths(t, cfg, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2, "Second fetch should not error")
	require.Len(t, events2, 3,
		"Second fetch with timezone mismatch should return 3 events; "+
			"got 0 indicates the Oracle timestamp cursor is affected by "+
			"timezone mismatch between session TZ (+05:00) and godror TZ (UTC) — "+
			"the bind parameter arrives as TIMESTAMP WITH TIME ZONE (UTC) but "+
			"Oracle converts stored TIMESTAMP using session TZ for comparison")
	t.Logf("TZ mismatch test - Second fetch: %d events", len(events2))
	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Third fetch — verify full pagination continues
	ms3 := newMetricSetWithPaths(t, cfg, testPaths)
	events3, errs3 := fetchEvents(t, ms3)
	require.Empty(t, errs3, "Third fetch should not error")
	require.Len(t, events3, 3, "Third fetch should return 3 events")
	if closer, ok := ms3.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Fourth fetch — last row
	ms4 := newMetricSetWithPaths(t, cfg, testPaths)
	events4, errs4 := fetchEvents(t, ms4)
	require.Empty(t, errs4, "Fourth fetch should not error")
	require.Len(t, events4, 1, "Fourth fetch should return 1 event")
	if closer, ok := ms4.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

// Note: Oracle helper functions (GetOracleConnectionDetails, GetOracleEnvServiceName,
// GetOracleEnvUsername, GetOracleEnvPassword, GetOracleConnectString, waitForOracleConnection)
// are defined in query_integration_test.go and shared with these tests.

// ============================================================================
// MSSQL CURSOR TESTS
// ============================================================================

// TestMSSQLCursor tests cursor functionality with Microsoft SQL Server
func TestMSSQLCursor(t *testing.T) {
	service := compose.EnsureUp(t, "mssql")
	host := service.Host()

	// Wait for MSSQL to be ready
	waitForMSSQLConnection(t, host)

	dsn := GetMSSQLConnectionDSN(host)

	// Set up test table
	db, err := sql.Open("sqlserver", dsn)
	require.NoError(t, err, "Failed to connect to MSSQL")
	defer db.Close()

	setupMSSQLTestTable(t, db)
	defer cleanupMSSQLTestTable(t, db)

	// Test integer cursor
	t.Run("integer_cursor", func(t *testing.T) {
		testMSSQLIntegerCursor(t, dsn)
	})

	// Test timestamp cursor
	t.Run("timestamp_cursor", func(t *testing.T) {
		testMSSQLTimestampCursor(t, dsn)
	})

	// Test date cursor
	t.Run("date_cursor", func(t *testing.T) {
		testMSSQLDateCursor(t, dsn)
	})
}

func setupMSSQLTestTable(t *testing.T, db *sql.DB) {
	t.Helper()

	// Drop table if exists
	_, _ = db.Exec("DROP TABLE IF EXISTS cursor_test_events")

	// Create table with MSSQL-specific syntax
	createSQL := `
		CREATE TABLE cursor_test_events (
			id INT IDENTITY(1,1) PRIMARY KEY,
			event_data NVARCHAR(255),
			created_at DATETIME2 DEFAULT GETUTCDATE(),
			event_date DATE DEFAULT CAST(GETUTCDATE() AS DATE)
		)
	`
	_, err := db.Exec(createSQL)
	require.NoError(t, err, "Failed to create MSSQL test table")

	// Insert test data
	insertSQL := `INSERT INTO cursor_test_events (event_data, created_at, event_date) VALUES (@p1, @p2, @p3)`
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		ts := now.Add(time.Duration(i) * time.Second)
		_, err := db.Exec(insertSQL, fmt.Sprintf("event-%d", i), ts, ts)
		require.NoError(t, err, "Failed to insert MSSQL test data")
	}
	t.Log("MSSQL test table created with 5 rows")
}

func cleanupMSSQLTestTable(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("DROP TABLE IF EXISTS cursor_test_events")
	if err != nil {
		t.Logf("Warning: failed to cleanup MSSQL test table: %v", err)
	}
}

func testMSSQLIntegerCursor(t *testing.T, dsn string) {
	t.Helper()

	testPaths := createTestPaths(t)

	query := "SELECT TOP 3 id, event_data FROM cursor_test_events WHERE id > @p1 ORDER BY id ASC"

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "mssql",
		"sql_query":           "SELECT TOP 3 id, event_data FROM cursor_test_events WHERE id > :cursor ORDER BY id ASC",
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "id",
		"cursor.type":         cursor.CursorTypeInteger,
		"cursor.default":      "0",
	}
	_ = query // query is translated internally

	// First fetch - should get first 3 rows
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1, "First fetch should not have errors")
	require.Len(t, events1, 3, "First fetch should return 3 events")

	for _, event := range events1 {
		if id, ok := event.MetricSetFields["id"]; ok {
			t.Logf("MSSQL integer cursor - First fetch: id=%v", id)
		}
	}

	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Second fetch - should get remaining 2 rows
	ms2 := newMetricSetWithPaths(t, cfg, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2, "Second fetch should not have errors")
	require.Len(t, events2, 2, "Second fetch should return 2 events")

	for _, event := range events2 {
		if id, ok := event.MetricSetFields["id"]; ok {
			t.Logf("MSSQL integer cursor - Second fetch: id=%v", id)
		}
	}

	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Third fetch - should get no rows
	ms3 := newMetricSetWithPaths(t, cfg, testPaths)
	events3, errs3 := fetchEvents(t, ms3)
	require.Empty(t, errs3, "Third fetch should not have errors")
	require.Len(t, events3, 0, "Third fetch should return 0 events")

	if closer, ok := ms3.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

func testMSSQLTimestampCursor(t *testing.T, dsn string) {
	t.Helper()

	testPaths := createTestPaths(t)
	defaultTimestamp := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "mssql",
		"sql_query":           "SELECT TOP 3 id, event_data, created_at FROM cursor_test_events WHERE created_at > :cursor ORDER BY created_at ASC",
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "created_at",
		"cursor.type":         cursor.CursorTypeTimestamp,
		"cursor.default":      defaultTimestamp,
	}

	// First fetch
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1, "First timestamp fetch should not have errors")
	require.Len(t, events1, 3, "First timestamp fetch should return 3 events")

	t.Logf("MSSQL timestamp cursor - First fetch returned %d events", len(events1))

	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Second fetch
	ms2 := newMetricSetWithPaths(t, cfg, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2, "Second timestamp fetch should not have errors")
	require.Len(t, events2, 2, "Second timestamp fetch should return 2 events")

	t.Logf("MSSQL timestamp cursor - Second fetch returned %d events", len(events2))

	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

func testMSSQLDateCursor(t *testing.T, dsn string) {
	t.Helper()

	testPaths := createTestPaths(t)
	defaultDate := time.Now().Add(-24 * time.Hour).UTC().Format("2006-01-02")

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "mssql",
		"sql_query":           "SELECT TOP 3 id, event_data, event_date FROM cursor_test_events WHERE event_date > :cursor ORDER BY event_date ASC, id ASC",
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "event_date",
		"cursor.type":         cursor.CursorTypeDate,
		"cursor.default":      defaultDate,
	}

	// First fetch
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1, "First date fetch should not have errors")
	require.GreaterOrEqual(t, len(events1), 1, "First date fetch should return events")

	t.Logf("MSSQL date cursor - First fetch returned %d events", len(events1))

	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

// MSSQL helper functions
func GetMSSQLConnectionDSN(host string) string {
	user := GetMSSQLEnvUser()
	password := GetMSSQLEnvPassword()
	// Disable TLS encryption to avoid certificate validation issues in testing
	return fmt.Sprintf("sqlserver://%s:%s@%s?encrypt=disable", user, password, host)
}

func GetMSSQLEnvUser() string {
	user := os.Getenv("MSSQL_USER")
	if user == "" {
		user = "SA"
	}
	return user
}

func GetMSSQLEnvPassword() string {
	password := os.Getenv("MSSQL_PASSWORD")
	if password == "" {
		password = "1234_asdf"
	}
	return password
}

func waitForMSSQLConnection(t *testing.T, host string) {
	maxRetries := 30
	baseDelay := 2 * time.Second

	dsn := GetMSSQLConnectionDSN(host)

	for i := 0; i < maxRetries; i++ {
		db, err := sql.Open("sqlserver", dsn)
		if err == nil {
			err = db.Ping()
			db.Close()
			if err == nil {
				t.Log("MSSQL is ready")
				// Give it a bit more time for stability
				time.Sleep(5 * time.Second)
				return
			}
		}

		delay := time.Duration(1<<uint(i)) * baseDelay
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}

		t.Logf("MSSQL not ready yet (attempt %d/%d), waiting %v: %v", i+1, maxRetries, delay, err)
		time.Sleep(delay)
	}

	t.Fatalf("MSSQL service did not become ready after %d attempts", maxRetries)
}

// ============================================================================
// MULTI-DATABASE CURSOR STATE ISOLATION TEST
// ============================================================================

// TestCursorStateIsolation verifies that cursor states are isolated per database/query
func TestCursorStateIsolation(t *testing.T) {
	service := compose.EnsureUp(t, "postgresql")
	host, port, err := net.SplitHostPort(service.Host())
	require.NoError(t, err)

	user := postgresql.GetEnvUsername()
	password := postgresql.GetEnvPassword()
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/?sslmode=disable", user, password, host, port)

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	setupPostgresTestTable(t, db)
	defer cleanupTestTable(t, db, "postgres")

	// Use shared paths so both MetricSets would share state if not properly isolated
	testPaths := createTestPaths(t)

	// Two different queries on same table should have separate cursor states
	query1 := fmt.Sprintf("SELECT id, event_data FROM %s WHERE id > :cursor ORDER BY id ASC LIMIT 2", testTableName)
	query2 := fmt.Sprintf("SELECT id, event_data FROM %s WHERE id > :cursor ORDER BY id ASC LIMIT 3", testTableName)

	cfg1 := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "postgres",
		"sql_query":           query1,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "id",
		"cursor.type":         cursor.CursorTypeInteger,
		"cursor.default":      "0",
	}

	cfg2 := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "postgres",
		"sql_query":           query2,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "id",
		"cursor.type":         cursor.CursorTypeInteger,
		"cursor.default":      "0",
	}

	// First query fetch - gets 2 rows (ids 1, 2)
	ms1 := newMetricSetWithPaths(t, cfg1, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1)
	require.Len(t, events1, 2, "First query should return 2 events")

	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Second query fetch - should get 3 rows (ids 1, 2, 3) because it has separate state
	ms2 := newMetricSetWithPaths(t, cfg2, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2)
	require.Len(t, events2, 3, "Second query should return 3 events (separate cursor state)")

	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// First query again - should continue from where it left off (gets ids 3, 4)
	ms1again := newMetricSetWithPaths(t, cfg1, testPaths)
	events1again, errs1again := fetchEvents(t, ms1again)
	require.Empty(t, errs1again)
	require.Len(t, events1again, 2, "First query (second fetch) should return 2 more events")

	if closer, ok := ms1again.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	t.Log("Cursor state isolation verified - different queries maintain separate cursor states")
}

// TestCursorRegistrySharing verifies that multiple SQL module instances share
// the same statestore.Registry pointer, preventing file lock conflicts.
//
// This test ensures the ModuleBuilder closure pattern correctly shares the
// registry across all module instances, which is critical for avoiding the
// original bug where multiple independent stores operated on the same files.
func TestCursorRegistrySharing(t *testing.T) {
	service := compose.EnsureUp(t, "postgresql")
	host, port, err := net.SplitHostPort(service.Host())
	require.NoError(t, err)

	user := postgresql.GetEnvUsername()
	password := postgresql.GetEnvPassword()
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/?sslmode=disable", user, password, host, port)

	// Setup test database
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	setupPostgresTestTable(t, db)
	defer cleanupTestTable(t, db, "postgres")

	// Insert test data: 10 rows
	insertTestData(t, db, "postgres", 10)

	// Create shared test paths - both MetricSets will use same data directory
	testPaths := createTestPaths(t)

	// Configuration for first MetricSet - query with LIMIT 2
	cfg1 := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "postgres",
		"period":              "10s",
		"sql_query":           fmt.Sprintf("SELECT id, event_data FROM %s WHERE id > :cursor ORDER BY id ASC LIMIT 2", testTableName),
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "id",
		"cursor.type":         cursor.CursorTypeInteger,
		"cursor.default":      "0",
	}

	// Configuration for second MetricSet - different query with LIMIT 3
	cfg2 := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "postgres",
		"period":              "10s",
		"sql_query":           fmt.Sprintf("SELECT id, event_data FROM %s WHERE id > :cursor ORDER BY id ASC LIMIT 3", testTableName),
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "id",
		"cursor.type":         cursor.CursorTypeInteger,
		"cursor.default":      "0",
	}

	// Create two MetricSet instances using the same paths
	ms1 := newMetricSetWithPaths(t, cfg1, testPaths)
	ms2 := newMetricSetWithPaths(t, cfg2, testPaths)

	// Extract the underlying modules
	metricSet1, ok := ms1.(*MetricSet)
	require.True(t, ok, "MetricSet should be *query.MetricSet")

	metricSet2, ok := ms2.(*MetricSet)
	require.True(t, ok, "MetricSet should be *query.MetricSet")

	// Type-assert to sql.Module interface to access GetCursorRegistry
	mod1, ok := metricSet1.BaseMetricSet.Module().(sqlmod.Module)
	require.True(t, ok, "Module should implement sqlmod.Module interface")

	mod2, ok := metricSet2.BaseMetricSet.Module().(sqlmod.Module)
	require.True(t, ok, "Module should implement sqlmod.Module interface")

	// Get registry from both modules
	registry1, err1 := mod1.GetCursorRegistry()
	require.NoError(t, err1, "GetCursorRegistry should not error")
	require.NotNil(t, registry1, "Registry should not be nil")

	registry2, err2 := mod2.GetCursorRegistry()
	require.NoError(t, err2, "GetCursorRegistry should not error")
	require.NotNil(t, registry2, "Registry should not be nil")

	// CRITICAL ASSERTION: Verify they're the SAME pointer (shared instance)
	// This is the core of the fix - if pointers differ, multiple stores will
	// try to access the same files, causing lock conflicts
	assert.Same(t, registry1, registry2,
		"Both module instances MUST share the exact same registry pointer to avoid file conflicts")

	t.Logf("✓ Registry sharing verified: both modules use registry at %p", registry1)

	// Also verify state isolation works correctly with the shared registry
	// Each query should maintain its own cursor state via unique state keys

	// Fetch from ms1 (gets 2 rows: id=1, id=2)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1)
	require.Len(t, events1, 2, "First fetch from ms1 should return 2 rows")

	// Fetch from ms2 (gets 3 rows: id=1, id=2, id=3)
	// This should have separate state from ms1 — cursor starts at 0 independently
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2)
	require.Len(t, events2, 3, "First fetch from ms2 should return 3 rows")

	// Fetch from ms1 again (continues from id=2, gets id=3, id=4)
	events3, errs3 := fetchEvents(t, ms1)
	require.Empty(t, errs3)
	require.Len(t, events3, 2, "Second fetch from ms1 should return next 2 rows")

	// Fetch from ms2 again (continues from id=3, gets id=4, id=5, id=6)
	events4, errs4 := fetchEvents(t, ms2)
	require.Empty(t, errs4)
	require.Len(t, events4, 3, "Second fetch from ms2 should return next 3 rows")

	// Cleanup
	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	t.Log("✓ State isolation verified: different queries maintain separate cursor states despite shared registry")
	t.Log("✓ Registry sharing test passed: single registry, no file conflicts, proper state isolation")
}

// ============================================================================
// QUERY TIMEOUT TEST
// ============================================================================

// TestCursorQueryTimeout verifies that a hung query is cancelled after the
// module's configured timeout. Uses PostgreSQL's pg_sleep() to simulate a
// query that takes longer than the timeout.
func TestCursorQueryTimeout(t *testing.T) {
	service := compose.EnsureUp(t, "postgresql")
	host, port, err := net.SplitHostPort(service.Host())
	require.NoError(t, err)

	user := postgresql.GetEnvUsername()
	password := postgresql.GetEnvPassword()
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/?sslmode=disable", user, password, host, port)

	// Ensure test table exists (we need a valid cursor query structure)
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	setupPostgresTestTable(t, db)
	defer cleanupTestTable(t, db, "postgres")

	testPaths := createTestPaths(t)

	// Query that sleeps for 30 seconds — well beyond the 2s timeout we'll set.
	// pg_sleep returns void, so we wrap it in a subquery that also selects from
	// our real table so the cursor column ("id") is present.
	// Using a CTE: the sleep executes, then we return rows from the real table.
	slowQuery := fmt.Sprintf(
		"SELECT id, event_data FROM %s WHERE id > :cursor AND pg_sleep(30) IS NOT NULL ORDER BY id ASC LIMIT 3",
		testTableName,
	)

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "postgres",
		"period":              "60s",
		"timeout":             "2s", // Very short timeout — query should be cancelled
		"sql_query":           slowQuery,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "id",
		"cursor.type":         cursor.CursorTypeInteger,
		"cursor.default":      "0",
	}

	ms := newMetricSetWithPaths(t, cfg, testPaths)

	// Measure the time taken — should be roughly the timeout, not 30s
	start := time.Now()
	_, errs := fetchEvents(t, ms)
	elapsed := time.Since(start)

	// Should have an error (context deadline exceeded)
	require.NotEmpty(t, errs, "Expected an error from the timed-out query")
	t.Logf("Query timeout error: %v", errs[0])

	// Verify it contains a context-related error
	errMsg := errs[0].Error()
	assert.True(t,
		strings.Contains(errMsg, "context deadline exceeded") ||
			strings.Contains(errMsg, "context canceled") ||
			strings.Contains(errMsg, "canceling statement due to user request"),
		"Error should indicate context cancellation, got: %s", errMsg)

	// Verify it completed quickly (within ~5s) rather than waiting 30s
	assert.Less(t, elapsed, 10*time.Second,
		"Query should have been cancelled by timeout, not waited for pg_sleep(30)")
	t.Logf("Query was cancelled after %v (timeout was 2s)", elapsed)

	// Verify cursor was NOT advanced (it should remain at default "0")
	queryMs, ok := ms.(*MetricSet)
	require.True(t, ok, "MetricSet should be of type *MetricSet")
	assert.Equal(t, "0", queryMs.cursorManager.CursorValueString(),
		"Cursor should remain at default after timeout")

	if closer, ok := ms.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}
}

// TestCursorNormalQueryCompletesWithinTimeout verifies that a normal (fast)
// query completes successfully even with a timeout configured.
func TestCursorNormalQueryCompletesWithinTimeout(t *testing.T) {
	service := compose.EnsureUp(t, "postgresql")
	host, port, err := net.SplitHostPort(service.Host())
	require.NoError(t, err)

	user := postgresql.GetEnvUsername()
	password := postgresql.GetEnvPassword()
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/?sslmode=disable", user, password, host, port)

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	setupPostgresTestTable(t, db)
	defer cleanupTestTable(t, db, "postgres")

	testPaths := createTestPaths(t)

	query := fmt.Sprintf("SELECT id, event_data FROM %s WHERE id > :cursor ORDER BY id ASC LIMIT 3", testTableName)

	cfg := map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{dsn},
		"driver":              "postgres",
		"period":              "60s",
		"timeout":             "30s", // Generous timeout — query should complete well within
		"sql_query":           query,
		"sql_response_format": tableResponseFormat,
		"raw_data.enabled":    true,
		"cursor.enabled":      true,
		"cursor.column":       "id",
		"cursor.type":         cursor.CursorTypeInteger,
		"cursor.default":      "0",
	}

	// First fetch - should work fine and return 3 rows
	ms1 := newMetricSetWithPaths(t, cfg, testPaths)
	events1, errs1 := fetchEvents(t, ms1)
	require.Empty(t, errs1, "Normal query should succeed with timeout configured")
	require.Len(t, events1, 3, "First fetch should return 3 events")

	if closer, ok := ms1.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	// Second fetch - should get remaining 2 rows (cursor persisted correctly)
	ms2 := newMetricSetWithPaths(t, cfg, testPaths)
	events2, errs2 := fetchEvents(t, ms2)
	require.Empty(t, errs2, "Second fetch should succeed")
	require.Len(t, events2, 2, "Second fetch should return 2 events")

	if closer, ok := ms2.(mb.Closer); ok {
		require.NoError(t, closer.Close())
	}

	t.Log("Normal query completes successfully with timeout configured")
}

// containsOracleClientError checks if the error message indicates Oracle Instant Client is missing
func containsOracleClientError(errMsg string) bool {
	oracleClientErrors := []string{
		"Cannot locate a 64-bit Oracle Client library",
		"libclntsh",
		"DPI-1047",
		"oracle client",
	}
	errLower := strings.ToLower(errMsg)
	for _, pattern := range oracleClientErrors {
		if strings.Contains(errLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}
