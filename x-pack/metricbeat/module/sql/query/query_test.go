// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package query

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/sql/query/cursor"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/paths"
)

type fakeDBClient struct {
	tableRows          []mapstr.M
	tableErr           error
	variableRows       mapstr.M
	variableErr        error
	tableWithParamRows []mapstr.M
	tableWithParamErr  error

	tableQueries     []string
	variableQueries  []string
	withParamQueries []string
	withParamArgs    [][]interface{}
	closed           bool

	fetchTableFn          func(ctx context.Context, query string) ([]mapstr.M, error)
	fetchVariableFn       func(ctx context.Context, query string) (mapstr.M, error)
	fetchTableWithParamFn func(ctx context.Context, query string, args ...interface{}) ([]mapstr.M, error)
}

func (f *fakeDBClient) FetchTableMode(ctx context.Context, query string) ([]mapstr.M, error) {
	f.tableQueries = append(f.tableQueries, query)
	if f.fetchTableFn != nil {
		return f.fetchTableFn(ctx, query)
	}
	return f.tableRows, f.tableErr
}

func (f *fakeDBClient) FetchTableModeWithParams(ctx context.Context, query string, args ...interface{}) ([]mapstr.M, error) {
	f.withParamQueries = append(f.withParamQueries, query)
	f.withParamArgs = append(f.withParamArgs, args)
	if f.fetchTableWithParamFn != nil {
		return f.fetchTableWithParamFn(ctx, query, args...)
	}
	return f.tableWithParamRows, f.tableWithParamErr
}

func (f *fakeDBClient) FetchVariableMode(ctx context.Context, query string) (mapstr.M, error) {
	f.variableQueries = append(f.variableQueries, query)
	if f.fetchVariableFn != nil {
		return f.fetchVariableFn(ctx, query)
	}
	return f.variableRows, f.variableErr
}

func (f *fakeDBClient) Close() error {
	f.closed = true
	return nil
}

type stopReporter struct {
	events int
}

func (r *stopReporter) Event(_ mb.Event) bool {
	r.events++
	return false
}

func (r *stopReporter) Error(_ error) bool {
	return false
}

func testMetricSetConfig(queryText string) map[string]interface{} {
	return map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{"postgres://user:pass@localhost:5432/mydb?sslmode=disable"},
		"driver":              "postgres",
		"sql_query":           queryText,
		"sql_response_format": "table",
	}
}

func newTestMetricSet(t *testing.T, cfg map[string]interface{}) *MetricSet {
	t.Helper()

	ms := mbtest.NewMetricSet(t, cfg)
	qms, ok := ms.(*MetricSet)
	require.Truef(t, ok, "expected *MetricSet, got %T", ms)
	return qms
}

func newTestCursorManager(t *testing.T, defaultValue string) *cursor.Manager {
	t.Helper()

	logger := logp.NewNopLogger()
	reg, err := memlog.New(logger.Named("memlog"), memlog.Settings{
		Root:     t.TempDir(),
		FileMode: 0o600,
	})
	require.NoError(t, err)

	registry := statestore.NewRegistry(reg)
	t.Cleanup(func() { registry.Close() })

	store, err := cursor.NewStoreFromRegistry(registry, logger.Named("cursor"))
	require.NoError(t, err)

	cfg := cursor.Config{
		Enabled:   true,
		Column:    "id",
		Type:      cursor.CursorTypeInteger,
		Default:   defaultValue,
		Direction: cursor.CursorDirectionAsc,
	}
	mgr, err := cursor.NewManager(cfg, store,
		"postgres://user:pass@localhost:5432/mydb?sslmode=disable",
		"SELECT id FROM t WHERE id > :cursor ORDER BY id ASC",
		logger,
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = mgr.Close() })
	return mgr
}

func withFakeDBClientFactory(t *testing.T, db dbClient) {
	t.Helper()

	original := newDBClient
	newDBClient = func(_, _ string, _ *logp.Logger) (dbClient, error) {
		return db, nil
	}
	t.Cleanup(func() {
		newDBClient = original
	})
}

func withTempDataPath(t *testing.T) {
	t.Helper()
	origData := paths.Paths.Data
	paths.Paths.Data = t.TempDir()
	t.Cleanup(func() {
		paths.Paths.Data = origData
	})
}

func instantiateMetricSetWithConfig(t *testing.T, cfg map[string]interface{}) error {
	t.Helper()
	withTempDataPath(t)

	c, err := conf.NewConfigFrom(cfg)
	require.NoError(t, err)
	_, _, err = mb.NewModule(c, mb.Registry, paths.New(), logptest.NewTestingLogger(t, ""))
	return err
}

func TestFetch_NonCursorPath_ReportsEvents(t *testing.T) {
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM t ORDER BY id"))

	fakeDB := &fakeDBClient{
		tableRows: []mapstr.M{
			{"id": int64(1), "name": "a"},
		},
	}
	withFakeDBClientFactory(t, fakeDB)

	reporter := &mbtest.CapturingReporterV2{}
	err := ms.Fetch(context.Background(), reporter)
	require.NoError(t, err)

	assert.True(t, fakeDB.closed, "DB client should be closed after fetch")
	assert.Len(t, fakeDB.tableQueries, 1, "table query should be executed once")
	assert.Len(t, reporter.GetEvents(), 1, "one event should be reported")
}

func TestFetch_CursorPath_AdvancesCursorOnSuccessfulReporting(t *testing.T) {
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM t WHERE id > :cursor ORDER BY id ASC"))
	ms.cursorManager = newTestCursorManager(t, "0")
	ms.translatedQuery = cursor.TranslateQuery(ms.Config.Query, ms.Config.Driver)

	fakeDB := &fakeDBClient{
		tableWithParamRows: []mapstr.M{
			{"id": int64(10)},
			{"id": int64(20)},
		},
	}
	withFakeDBClientFactory(t, fakeDB)

	reporter := &mbtest.CapturingReporterV2{}
	err := ms.Fetch(context.Background(), reporter)
	require.NoError(t, err)

	assert.True(t, fakeDB.closed, "DB client should be closed after fetch")
	assert.Len(t, fakeDB.withParamQueries, 1, "parameterized query should be executed once")
	assert.Equal(t, ms.translatedQuery, fakeDB.withParamQueries[0])
	require.Len(t, fakeDB.withParamArgs, 1)
	require.Len(t, fakeDB.withParamArgs[0], 1)
	assert.EqualValues(t, int64(0), fakeDB.withParamArgs[0][0], "default cursor value should be passed as arg")
	assert.Len(t, reporter.GetEvents(), 2, "all rows should be reported")
	assert.Equal(t, "20", ms.cursorManager.CursorValueString(), "cursor should advance to max id")
}

func TestFetch_CursorPath_DoesNotAdvanceWhenReporterStops(t *testing.T) {
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM t WHERE id > :cursor ORDER BY id ASC"))
	ms.cursorManager = newTestCursorManager(t, "0")
	ms.translatedQuery = cursor.TranslateQuery(ms.Config.Query, ms.Config.Driver)

	fakeDB := &fakeDBClient{
		tableWithParamRows: []mapstr.M{
			{"id": int64(10)},
			{"id": int64(20)},
		},
	}
	withFakeDBClientFactory(t, fakeDB)

	reporter := &stopReporter{}
	err := ms.Fetch(context.Background(), reporter)
	require.NoError(t, err)

	assert.Equal(t, 1, reporter.events, "fetch should stop after reporter rejects first event")
	assert.Equal(t, "0", ms.cursorManager.CursorValueString(), "cursor must not advance when reporting stops")
}

func TestFetch_CursorPath_ZeroRowsLeavesCursorUnchanged(t *testing.T) {
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM t WHERE id > :cursor ORDER BY id ASC"))
	ms.cursorManager = newTestCursorManager(t, "42")
	ms.translatedQuery = cursor.TranslateQuery(ms.Config.Query, ms.Config.Driver)

	fakeDB := &fakeDBClient{tableWithParamRows: []mapstr.M{}}
	withFakeDBClientFactory(t, fakeDB)

	reporter := &mbtest.CapturingReporterV2{}
	err := ms.Fetch(context.Background(), reporter)
	require.NoError(t, err)
	assert.Equal(t, "42", ms.cursorManager.CursorValueString())
	assert.Len(t, reporter.GetEvents(), 0)
}

func TestFetch_CursorPath_QueryErrorPropagates(t *testing.T) {
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM t WHERE id > :cursor ORDER BY id ASC"))
	ms.cursorManager = newTestCursorManager(t, "0")
	ms.translatedQuery = cursor.TranslateQuery(ms.Config.Query, ms.Config.Driver)

	fakeDB := &fakeDBClient{
		tableWithParamErr: errors.New("db query failed"),
	}
	withFakeDBClientFactory(t, fakeDB)

	err := ms.Fetch(context.Background(), &mbtest.CapturingReporterV2{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch with cursor failed")
}

func TestFetch_CursorPath_AppliesTimeoutContext(t *testing.T) {
	cfg := testMetricSetConfig("SELECT id FROM t WHERE id > :cursor ORDER BY id ASC")
	cfg["timeout"] = "50ms"
	ms := newTestMetricSet(t, cfg)
	ms.cursorManager = newTestCursorManager(t, "0")
	ms.translatedQuery = cursor.TranslateQuery(ms.Config.Query, ms.Config.Driver)

	deadlineSeen := false
	fakeDB := &fakeDBClient{
		fetchTableWithParamFn: func(ctx context.Context, _ string, _ ...interface{}) ([]mapstr.M, error) {
			_, ok := ctx.Deadline()
			deadlineSeen = ok
			return []mapstr.M{}, nil
		},
	}
	withFakeDBClientFactory(t, fakeDB)

	err := ms.Fetch(context.Background(), &mbtest.CapturingReporterV2{})
	require.NoError(t, err)
	assert.True(t, deadlineSeen, "cursor fetch path should apply module timeout")
}

func TestFetch_CursorPath_SkipsWhenPreviousFetchInProgress(t *testing.T) {
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM t WHERE id > :cursor ORDER BY id ASC"))
	ms.cursorManager = newTestCursorManager(t, "0")
	ms.translatedQuery = cursor.TranslateQuery(ms.Config.Query, ms.Config.Driver)

	fakeDB := &fakeDBClient{
		tableWithParamRows: []mapstr.M{{"id": int64(1)}},
	}
	withFakeDBClientFactory(t, fakeDB)

	ms.fetchMutex.Lock()
	defer ms.fetchMutex.Unlock()

	err := ms.Fetch(context.Background(), &mbtest.CapturingReporterV2{})
	require.NoError(t, err)
	assert.Len(t, fakeDB.withParamQueries, 0, "DB should not be called when TryLock fails")
}

func TestFetch_NonCursorPath_DBOpenError(t *testing.T) {
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM t"))

	orig := newDBClient
	newDBClient = func(_, _ string, _ *logp.Logger) (dbClient, error) {
		return nil, errors.New("open failed")
	}
	t.Cleanup(func() { newDBClient = orig })

	err := ms.Fetch(context.Background(), &mbtest.CapturingReporterV2{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot open connection")
}

func TestFetch_NonCursorPath_ReporterStopDoesNotError(t *testing.T) {
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM t ORDER BY id"))

	fakeDB := &fakeDBClient{
		tableRows: []mapstr.M{{"id": int64(1)}},
	}
	withFakeDBClientFactory(t, fakeDB)

	reporter := &stopReporter{}
	err := ms.Fetch(context.Background(), reporter)
	require.NoError(t, err)
	assert.Equal(t, 1, reporter.events)
}

func TestFetch_MergeResultsRejectsMultipleRowsPerQuery(t *testing.T) {
	ms := &MetricSet{
		Config: config{
			MergeResults: true,
		},
	}

	fakeDB := &fakeDBClient{
		tableRows: []mapstr.M{
			{"id": int64(1)},
			{"id": int64(2)},
		},
	}

	_, err := ms.fetch(context.Background(), fakeDB, &mbtest.CapturingReporterV2{}, []query{
		{Query: "SELECT id FROM t", ResponseFormat: tableResponseFormat},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot merge query resulting with more than one rows")
}

func TestFetch_TableAndVariableErrorsAreWrapped(t *testing.T) {
	ms := &MetricSet{Config: config{}}

	_, err := ms.fetch(context.Background(), &fakeDBClient{
		tableErr: errors.New("table err"),
	}, &mbtest.CapturingReporterV2{}, []query{
		{Query: "SELECT id FROM t", ResponseFormat: tableResponseFormat},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch table mode failed")

	_, err = ms.fetch(context.Background(), &fakeDBClient{
		variableErr: errors.New("var err"),
	}, &mbtest.CapturingReporterV2{}, []query{
		{Query: "SHOW STATUS", ResponseFormat: variableResponseFormat},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch variable mode failed")
}

func TestFetch_MergeResults_OverwritesDuplicateKeys(t *testing.T) {
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM t"))
	ms.Config.MergeResults = true
	ms.Config.Driver = "postgres"

	fakeDB := &fakeDBClient{
		fetchTableFn: func(_ context.Context, query string) ([]mapstr.M, error) {
			// one row/query so merge mode accepts it
			if strings.Contains(query, "table_a") {
				return []mapstr.M{{"dup": int64(1), "a": "x"}}, nil
			}
			if strings.Contains(query, "table_b") {
				return []mapstr.M{{"dup": int64(2), "b": "y"}}, nil
			}
			return nil, nil
		},
		fetchVariableFn: func(_ context.Context, _ string) (mapstr.M, error) {
			// duplicate key with table result to exercise overwrite path.
			return mapstr.M{"dup": int64(3), "var": "ok"}, nil
		},
	}

	reporter := &mbtest.CapturingReporterV2{}
	ok, err := ms.fetch(context.Background(), fakeDB, reporter, []query{
		{Query: "SELECT * FROM table_a", ResponseFormat: tableResponseFormat},
		{Query: "SELECT * FROM table_b", ResponseFormat: tableResponseFormat},
		{Query: "SHOW STATUS", ResponseFormat: variableResponseFormat},
	})
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, reporter.GetEvents(), 1)
}

func TestFetch_NonMerge_ReportsEachQueryResult(t *testing.T) {
	ms := &MetricSet{
		Config: config{
			MergeResults: false,
			Driver:       "postgres",
		},
	}

	fakeDB := &fakeDBClient{
		fetchTableFn: func(_ context.Context, _ string) ([]mapstr.M, error) {
			return []mapstr.M{
				{"id": int64(1)},
				{"id": int64(2)},
			}, nil
		},
		fetchVariableFn: func(_ context.Context, _ string) (mapstr.M, error) {
			return mapstr.M{"status": "ok"}, nil
		},
	}

	reporter := &mbtest.CapturingReporterV2{}
	ok, err := ms.fetch(context.Background(), fakeDB, reporter, []query{
		{Query: "SELECT id FROM t", ResponseFormat: tableResponseFormat},
		{Query: "SHOW STATUS", ResponseFormat: variableResponseFormat},
	})
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, reporter.GetEvents(), 3, "2 table rows + 1 variable result")
}

func TestFetch_NonCursorPath_FetchErrorIsSwallowed(t *testing.T) {
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM t"))

	withFakeDBClientFactory(t, &fakeDBClient{
		tableErr: errors.New("query failed"),
	})

	// Fetch logs and swallows per-query fetch errors in non-cursor path.
	err := ms.Fetch(context.Background(), &mbtest.CapturingReporterV2{})
	require.NoError(t, err)
}

func TestFetch_VariableMode_MergeResultsSuccess(t *testing.T) {
	ms := &MetricSet{
		Config: config{
			MergeResults: true,
			Driver:       "postgres",
		},
	}

	fakeDB := &fakeDBClient{
		variableRows: mapstr.M{
			"connections": int64(3),
			"db_name":     "mydb",
		},
	}

	reporter := &mbtest.CapturingReporterV2{}
	ok, err := ms.fetch(context.Background(), fakeDB, reporter, []query{
		{Query: "SHOW STATUS", ResponseFormat: variableResponseFormat},
	})
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, fakeDB.variableQueries, 1)
	assert.Len(t, reporter.GetEvents(), 1)
}

func TestInitCursorAndClose(t *testing.T) {
	withTempDataPath(t)
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM t WHERE id > :cursor ORDER BY id ASC"))
	ms.Config.Cursor = cursor.Config{
		Enabled: true,
		Column:  "id",
		Type:    cursor.CursorTypeInteger,
		Default: "0",
	}

	err := ms.initCursor(ms.BaseMetricSet)
	require.NoError(t, err)
	require.NotNil(t, ms.cursorManager)
	assert.Contains(t, ms.translatedQuery, "$1")

	require.NoError(t, ms.Close())
}

func TestCloseWithoutCursorManager(t *testing.T) {
	ms := &MetricSet{}
	require.NoError(t, ms.Close())
}

func TestReportEvent_RawDataWithAndWithoutQuery(t *testing.T) {
	ms := &MetricSet{
		Config: config{
			Driver: "postgres",
			RawData: rawData{
				Enabled: true,
			},
		},
	}

	reporter := &mbtest.CapturingReporterV2{}
	ok := ms.reportEvent(mapstr.M{"x": 1}, reporter, "SELECT 1")
	require.True(t, ok)
	require.Len(t, reporter.GetEvents(), 1)

	ms2 := &MetricSet{
		Config: config{
			Driver: "postgres",
			RawData: rawData{
				Enabled: true,
			},
		},
	}
	reporter2 := &mbtest.CapturingReporterV2{}
	ok = ms2.reportEvent(mapstr.M{"x": 2}, reporter2)
	require.True(t, ok)
	require.Len(t, reporter2.GetEvents(), 1)
}

func TestFetch_FetchFromAllDatabases_UnsupportedDriver(t *testing.T) {
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM t"))
	ms.Config.FetchFromAllDatabases = true
	ms.Config.Driver = "postgres"

	fakeDB := &fakeDBClient{}
	withFakeDBClientFactory(t, fakeDB)

	err := ms.Fetch(context.Background(), &mbtest.CapturingReporterV2{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch from all databases feature is not supported for driver: postgres")
}

func TestFetch_FetchFromAllDatabases_NoDatabaseNames(t *testing.T) {
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM t"))
	ms.Config.FetchFromAllDatabases = true
	ms.Config.Driver = "mssql"

	fakeDB := &fakeDBClient{
		tableRows: []mapstr.M{},
	}
	withFakeDBClientFactory(t, fakeDB)

	err := ms.Fetch(context.Background(), &mbtest.CapturingReporterV2{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no database names found")
}

func TestFetch_FetchFromAllDatabases_DBNamesQueryError(t *testing.T) {
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM t"))
	ms.Config.FetchFromAllDatabases = true
	ms.Config.Driver = "mssql"

	withFakeDBClientFactory(t, &fakeDBClient{
		tableErr: errors.New("list dbs failed"),
	})

	err := ms.Fetch(context.Background(), &mbtest.CapturingReporterV2{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot fetch database names")
}

func TestFetch_FetchFromAllDatabases_MSSQLSuccess(t *testing.T) {
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM metrics_table"))
	ms.Config.FetchFromAllDatabases = true
	ms.Config.Driver = "mssql"

	fakeDB := &fakeDBClient{
		fetchTableFn: func(_ context.Context, query string) ([]mapstr.M, error) {
			if strings.Contains(query, "sys.databases") {
				return []mapstr.M{{"name": "db1"}}, nil
			}
			if strings.Contains(query, "USE [db1];") {
				return []mapstr.M{{"id": int64(1)}}, nil
			}
			return nil, nil
		},
	}
	withFakeDBClientFactory(t, fakeDB)

	reporter := &mbtest.CapturingReporterV2{}
	err := ms.Fetch(context.Background(), reporter)
	require.NoError(t, err)
	assert.Len(t, reporter.GetEvents(), 1)
}

func TestFetch_FetchFromAllDatabases_SkipsRowsWithoutStringName(t *testing.T) {
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM metrics_table"))
	ms.Config.FetchFromAllDatabases = true
	ms.Config.Driver = "mssql"

	fakeDB := &fakeDBClient{
		fetchTableFn: func(_ context.Context, query string) ([]mapstr.M, error) {
			if strings.Contains(query, "sys.databases") {
				return []mapstr.M{
					{},                 // missing "name"
					{"name": int64(1)}, // wrong type
				}, nil
			}
			return nil, nil
		},
	}
	withFakeDBClientFactory(t, fakeDB)

	err := ms.Fetch(context.Background(), &mbtest.CapturingReporterV2{})
	require.NoError(t, err)
}

func TestFetch_FetchFromAllDatabases_InnerFetchErrorStopsCycle(t *testing.T) {
	ms := newTestMetricSet(t, testMetricSetConfig("SELECT id FROM metrics_table"))
	ms.Config.FetchFromAllDatabases = true
	ms.Config.Driver = "mssql"

	fakeDB := &fakeDBClient{
		fetchTableFn: func(_ context.Context, query string) ([]mapstr.M, error) {
			if strings.Contains(query, "sys.databases") {
				return []mapstr.M{{"name": "db1"}}, nil
			}
			// Per-database query fails inside m.fetch().
			if strings.Contains(query, "USE [db1];") {
				return nil, errors.New("db-specific failure")
			}
			return nil, nil
		},
	}
	withFakeDBClientFactory(t, fakeDB)

	// Current behavior: logs warning and returns nil when reporting/fetch fails for a DB.
	err := ms.Fetch(context.Background(), &mbtest.CapturingReporterV2{})
	require.NoError(t, err)
}

func TestOpenCursorStore_RequiresSQLModuleInterface(t *testing.T) {
	ms := &MetricSet{}
	_, err := ms.openCursorStore(mb.BaseMetricSet{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cursor requires SQL module to implement registry interface")
}

func TestInitCursor_PropagatesStoreInitError(t *testing.T) {
	ms := &MetricSet{
		Config: config{
			ResponseFormat: tableResponseFormat,
			Query:          "SELECT id FROM t WHERE id > :cursor",
			Cursor: cursor.Config{
				Enabled: true,
				Column:  "id",
				Type:    cursor.CursorTypeInteger,
				Default: "0",
			},
		},
	}

	err := ms.initCursor(mb.BaseMetricSet{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cursor store initialization failed")
}

func TestNew_ConfigValidationErrors(t *testing.T) {
	base := testMetricSetConfig("SELECT id FROM t")

	tests := []struct {
		name      string
		overrides map[string]interface{}
		wantErr   string
	}{
		{
			name: "invalid response format",
			overrides: map[string]interface{}{
				"sql_response_format": "bad",
			},
			wantErr: "invalid sql_response_format value: bad",
		},
		{
			name: "no query inputs",
			overrides: map[string]interface{}{
				"sql_query":   "",
				"sql_queries": []map[string]interface{}{},
			},
			wantErr: "no query input provided, must provide either sql_query or sql_queries",
		},
		{
			name: "both query and queries",
			overrides: map[string]interface{}{
				"sql_query": "SELECT 1",
				"sql_queries": []map[string]interface{}{
					{"query": "SELECT 2", "response_format": "table"},
				},
			},
			wantErr: "both query inputs provided, must provide either sql_query or sql_queries",
		},
		{
			name: "cursor with multiple queries",
			overrides: map[string]interface{}{
				"sql_query": "",
				"sql_queries": []map[string]interface{}{
					{"query": "SELECT id FROM t", "response_format": "table"},
				},
				"cursor.enabled": true,
				"cursor.column":  "id",
				"cursor.type":    "integer",
				"cursor.default": "0",
			},
			wantErr: "cursor is not supported with sql_queries (multiple queries)",
		},
		{
			name: "cursor with fetch from all databases",
			overrides: map[string]interface{}{
				"sql_query":                "SELECT id FROM t WHERE id > :cursor",
				"sql_response_format":      "table",
				"fetch_from_all_databases": true,
				"cursor.enabled":           true,
				"cursor.column":            "id",
				"cursor.type":              "integer",
				"cursor.default":           "0",
			},
			wantErr: "cursor is not supported with fetch_from_all_databases",
		},
		{
			name: "cursor requires table format",
			overrides: map[string]interface{}{
				"sql_query":           "SELECT id FROM t WHERE id > :cursor",
				"sql_response_format": "variables",
				"cursor.enabled":      true,
				"cursor.column":       "id",
				"cursor.type":         "integer",
				"cursor.default":      "0",
			},
			wantErr: "cursor requires sql_response_format: table",
		},
		{
			name: "invalid response format in sql_queries",
			overrides: map[string]interface{}{
				"sql_query": "",
				"sql_queries": []map[string]interface{}{
					{"query": "SELECT 1", "response_format": "invalid"},
				},
			},
			wantErr: "invalid sql_response_format value: invalid",
		},
		{
			name: "cursor query missing placeholder",
			overrides: map[string]interface{}{
				"sql_query":           "SELECT id FROM t",
				"sql_response_format": "table",
				"cursor.enabled":      true,
				"cursor.column":       "id",
				"cursor.type":         "integer",
				"cursor.default":      "0",
			},
			wantErr: "query must contain :cursor placeholder when cursor is enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := make(map[string]interface{}, len(base)+len(tt.overrides))
			for k, v := range base {
				cfg[k] = v
			}
			for k, v := range tt.overrides {
				cfg[k] = v
			}

			err := instantiateMetricSetWithConfig(t, cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestNew_DefaultsResponseFormatWhenEmpty(t *testing.T) {
	cfg := testMetricSetConfig("SELECT id FROM t")
	delete(cfg, "sql_response_format")
	err := instantiateMetricSetWithConfig(t, cfg)
	require.NoError(t, err)
}

func TestInferTypeFromMetricsAndDriverHelpers(t *testing.T) {
	typed := inferTypeFromMetrics(mapstr.M{
		"i":    int64(1),
		"f":    3.14,
		"s":    "hello",
		"b":    true,
		"n":    nil,
		"misc": time.Second, // default case -> string bucket
	})

	numeric, ok := typed["numeric"].(mapstr.M)
	require.True(t, ok)
	assert.Contains(t, numeric, "i")
	assert.Contains(t, numeric, "f")

	stringVals, ok := typed["string"].(mapstr.M)
	require.True(t, ok)
	assert.Contains(t, stringVals, "s")
	assert.Contains(t, stringVals, "misc")

	boolVals, ok := typed["bool"].(mapstr.M)
	require.True(t, ok)
	assert.Contains(t, boolVals, "b")
	assert.NotContains(t, typed, "n")

	assert.NotEmpty(t, queryDBNames("mssql"))
	assert.Equal(t, "", queryDBNames("postgres"))
	assert.Equal(t, "USE [mydb];", dbSelector("sqlserver", "mydb"))
	assert.Equal(t, "", dbSelector("postgres", "mydb"))
}
