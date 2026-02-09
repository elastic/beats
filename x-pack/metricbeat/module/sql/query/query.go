// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package query

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/metricbeat/helper/sql"
	"github.com/elastic/beats/v7/metricbeat/mb"
	sqlmod "github.com/elastic/beats/v7/x-pack/metricbeat/module/sql"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/sql/query/cursor"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// represents the response format of the query
const (
	tableResponseFormat    = "table"
	variableResponseFormat = "variables"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("sql", "query", New,
		mb.WithHostParser(ParseDSN),
	)
}

// Single query
type query struct {
	Query          string `config:"query" validate:"nonzero,required"`
	ResponseFormat string `config:"response_format" validate:"nonzero,required"`
}

// Metricset configuration
type config struct {
	// New flag
	RawData rawData `config:"raw_data"`

	Driver string `config:"driver" validate:"nonzero,required"`

	// Support either the query or list of queries.
	ResponseFormat string  `config:"sql_response_format"`
	Query          string  `config:"sql_query"`
	Queries        []query `config:"sql_queries"`
	MergeResults   bool    `config:"merge_results"`

	// Support fetch response for given queries from all databases.
	// NOTE: Currently, mssql driver only respects FetchFromAllDatabases.
	FetchFromAllDatabases bool `config:"fetch_from_all_databases"`

	// Cursor configuration for incremental data fetching
	Cursor cursor.Config `config:"cursor"`
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	Config config

	// Cursor-related fields (only used when cursor is enabled)
	cursorManager   *cursor.Manager
	translatedQuery string     // Query with driver-specific placeholder
	fetchMutex      sync.Mutex // Prevents concurrent fetch operations
}

// rawData is the minimum required set of fields to generate fully customized events with their own module key space
// and their own metricset key space.
type rawData struct {
	Enabled bool `config:"enabled"`
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	b := &MetricSet{BaseMetricSet: base}

	if err := base.Module().UnpackConfig(&b.Config); err != nil {
		return nil, fmt.Errorf("unpack config failed: %w", err)
	}

	if b.Config.ResponseFormat != "" {
		if b.Config.ResponseFormat != variableResponseFormat && b.Config.ResponseFormat != tableResponseFormat {
			return nil, fmt.Errorf("invalid sql_response_format value: %s", b.Config.ResponseFormat)
		}
	} else {
		// Backward compatibility, if no value is provided.
		// This will ensure there is no braking change, as the previous code worked with no ResponseFormat
		b.Config.ResponseFormat = variableResponseFormat
	}

	for _, q := range b.Config.Queries {
		if q.ResponseFormat != variableResponseFormat && q.ResponseFormat != tableResponseFormat {
			return nil, fmt.Errorf("invalid sql_response_format value: %s", q.ResponseFormat)
		}
	}

	if b.Config.Query == "" && len(b.Config.Queries) == 0 {
		return nil, fmt.Errorf("no query input provided, must provide either sql_query or sql_queries")
	}

	if b.Config.Query != "" && len(b.Config.Queries) > 0 {
		return nil, fmt.Errorf("both query inputs provided, must provide either sql_query or sql_queries")
	}

	// Initialize cursor if enabled
	if b.Config.Cursor.Enabled {
		if err := b.initCursor(base); err != nil {
			return nil, err
		}
	}

	return b, nil
}

// initCursor initializes the cursor manager if cursor is enabled.
// This validates cursor configuration and sets up the state store.
func (m *MetricSet) initCursor(base mb.BaseMetricSet) error {
	// Cursor only works with single query mode
	if len(m.Config.Queries) > 0 {
		return errors.New("cursor is not supported with sql_queries (multiple queries)")
	}

	// Cursor is not compatible with fetch_from_all_databases
	if m.Config.FetchFromAllDatabases {
		return errors.New("cursor is not supported with fetch_from_all_databases")
	}

	// Cursor requires table response format
	if m.Config.ResponseFormat != tableResponseFormat {
		return errors.New("cursor requires sql_response_format: table")
	}

	// Translate query placeholder for the driver
	m.translatedQuery = cursor.TranslateQuery(m.Config.Query, m.Config.Driver)

	// Get a cursor Store handle from the shared Module-level registry.
	// This ensures a single memlog.Registry is shared across all SQL MetricSet
	// instances, avoiding multiple independent stores operating on the same files.
	store, err := m.openCursorStore(base)
	if err != nil {
		return fmt.Errorf("cursor store initialization failed: %w", err)
	}

	// Create cursor manager.
	// We use the full URI (not just Host) for state key generation because
	// Host often strips the database name (for example, "localhost:5432" for both
	// postgres://...localhost:5432/db_a and db_b). The URI is hashed via
	// xxhash so there is no secret leakage risk in the stored key.
	mgr, err := cursor.NewManager(
		m.Config.Cursor,
		store,
		m.HostData().URI,
		m.Config.Query, // use original query for state key
		m.Logger(),
	)
	if err != nil {
		// Cleanup store on error
		if closeErr := store.Close(); closeErr != nil {
			m.Logger().Warnf("Failed to close store after cursor manager creation error: %v", closeErr)
		}
		return fmt.Errorf("cursor initialization failed: %w", err)
	}

	m.cursorManager = mgr
	return nil
}

// openCursorStore returns a cursor Store handle from the shared Module-level
// statestore registry. The registry must be initialized via sql.ModuleBuilder
// to ensure proper sharing across all SQL module instances.
//
// This method will fail if the module does not implement the sql.Module interface,
// preventing the creation of multiple independent stores that could cause file
// lock conflicts.
func (m *MetricSet) openCursorStore(base mb.BaseMetricSet) (*cursor.Store, error) {
	mod, ok := base.Module().(sqlmod.Module)
	if !ok {
		return nil, fmt.Errorf("cursor requires SQL module to implement registry interface; " +
			"ensure module is initialized via sql.ModuleBuilder (not DefaultModuleFactory)")
	}

	registry, err := mod.GetCursorRegistry()
	if err != nil {
		return nil, err
	}

	// Debug log to verify registry sharing is working
	m.Logger().Debugf("Using shared SQL cursor registry at %p", registry)

	return cursor.NewStoreFromRegistry(registry, m.Logger().Named("cursor"))
}

// Close implements mb.Closer for proper resource cleanup.
// This is called when the MetricSet is stopped.
func (m *MetricSet) Close() error {
	if m.cursorManager != nil {
		return m.cursorManager.Close()
	}
	return nil
}

// queryDBNames returns the query to list databases present in a server
// as per the driver name. If the given driver is not supported, queryDBNames
// returns an empty query.
func queryDBNames(driver string) string {
	switch sql.SwitchDriverName(driver) {
	// NOTE: Add support for other drivers in future as when the need arises.
	// dbSelector function would also required to be modified in order to add
	// support for a new driver.
	case "mssql", "sqlserver":
		return "SELECT [name] FROM sys.databases WITH (NOLOCK) WHERE state = 0 AND HAS_DBACCESS([name]) = 1"
		// case "mysql":
		// 	return "SHOW DATABASES"
		// case "godror":
		// 	// NOTE: Requires necessary priviledges to access DBA_USERS
		// 	// Ref: https://stackoverflow.com/a/3005623/5821408
		// 	return "SELECT * FROM DBA_USERS"
		// case "postgres":
		// 	return "SELECT datname FROM pg_database"
	}

	return ""
}

// dbSelector returns the statement to select a named database to run the
// subsequent statements. If the given driver is not supported, dbSelector
// returns an empty statement.
func dbSelector(driver, dbName string) string {
	switch sql.SwitchDriverName(driver) {
	// NOTE: Add support for other drivers in future as when the need arises.
	// queryDBNames function would also required to be modified in order to add
	// support for a new driver.
	//
	case "mssql", "sqlserver":
		return fmt.Sprintf("USE [%s];", dbName)
	}
	return ""
}

func (m *MetricSet) fetch(ctx context.Context, db *sql.DbClient, reporter mb.ReporterV2, queries []query) (_ bool, fetchErr error) {
	defer func() {
		fetchErr = sql.SanitizeError(fetchErr, m.HostData().URI)
	}()

	var ok bool
	merged := make(mapstr.M, 0)
	storeQueries := make([]string, 0, len(queries))
	for _, q := range queries {
		storeQueries = append(storeQueries, q.Query)
		if q.ResponseFormat == tableResponseFormat {
			// Table format
			mss, err := db.FetchTableMode(ctx, q.Query)
			if err != nil {
				return ok, fmt.Errorf("fetch table mode failed: %w", err)
			}

			for _, ms := range mss {
				if m.Config.MergeResults {
					if len(mss) > 1 {
						return ok, fmt.Errorf("cannot merge query resulting with more than one rows: %s", q)
					} else {
						for k, v := range ms {
							_, ok := merged[k]
							if ok {
								m.Logger().Warn("overwriting duplicate metrics:", k)
							}
							merged[k] = v
						}
					}
				} else {
					// Report immediately for non-merged cases.
					ok = m.reportEvent(ms, reporter, q.Query)
				}
			}
		} else {
			// Variable format
			ms, err := db.FetchVariableMode(ctx, q.Query)
			if err != nil {
				return ok, fmt.Errorf("fetch variable mode failed: %w", err)
			}

			if m.Config.MergeResults {
				for k, v := range ms {
					_, ok := merged[k]
					if ok {
						m.Logger().Warn("overwriting duplicate metrics:", k)
					}
					merged[k] = v
				}
			} else {
				// Report immediately for non-merged cases.
				ok = m.reportEvent(ms, reporter, q.Query)
			}
		}
	}

	if m.Config.MergeResults {
		// Report here for merged case.
		ok = m.reportEvent(merged, reporter, storeQueries...)
	}

	return ok, nil
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
// It calls m.fetchTableMode() or m.fetchVariableMode() depending on the response
// format of the query.
func (m *MetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) (fetchErr error) {
	defer func() {
		fetchErr = sql.SanitizeError(fetchErr, m.HostData().URI)
	}()

	// Handle cursor-enabled case with concurrent execution prevention
	if m.cursorManager != nil {
		// Try to acquire lock without blocking
		if !m.fetchMutex.TryLock() {
			m.Logger().Warn("Previous collection still in progress, skipping this cycle")
			return nil
		}
		defer m.fetchMutex.Unlock()

		return m.fetchWithCursor(ctx, reporter)
	}

	db, err := sql.NewDBClient(m.Config.Driver, m.HostData().URI, m.Logger())
	if err != nil {
		return fmt.Errorf("cannot open connection: %w", err)
	}
	defer db.Close()

	queries := m.Config.Queries
	if len(queries) == 0 {
		one_query := query{Query: m.Config.Query, ResponseFormat: m.Config.ResponseFormat}
		queries = append(queries, one_query)
	}

	if !m.Config.FetchFromAllDatabases {
		reported, err := m.fetch(ctx, db, reporter, queries)
		if err != nil {
			m.Logger().Warn("error while fetching:", err)
		}
		if !reported {
			m.Logger().Debug("error trying to emit event")
		}
		return nil
	}

	// NOTE: Only mssql driver is supported for now because:
	//
	// * Difference in queries to fetch the name of the databases
	// * The statement to select a named database (for subsequent statements
	// to be executed) may not be generic i.e, USE statement (e.g., USE <db_name>)
	// works for MSSQL but not Oracle.
	//
	// TODO: Add the feature for other drivers when need arises.
	validQuery := queryDBNames(m.Config.Driver)
	if validQuery == "" {
		return fmt.Errorf("fetch from all databases feature is not supported for driver: %s", m.Config.Driver)
	}

	// Discover all databases in the server and execute given queries on each
	// of the databases.
	dbNames, err := db.FetchTableMode(ctx, queryDBNames(m.Config.Driver))
	if err != nil {
		return fmt.Errorf("cannot fetch database names: %w", err)
	}

	if len(dbNames) == 0 {
		return errors.New("no database names found")
	}

	qs := make([]query, 0, len(queries))

	for i := range dbNames {
		// Create a copy of the queries as query would be modified on every
		// iteration.
		qs = qs[:0]                 // empty slice
		qs = append(qs, queries...) // copy queries

		val, err := dbNames[i].GetValue("name")
		if err != nil {
			m.Logger().Warn("error with database name:", err)
			continue
		}
		dbName, ok := val.(string)
		if !ok {
			m.Logger().Warn("error with database name's type")
			continue
		}

		// Prefix dbSelector to the query based on the driver
		// provided.
		// Example: USE <dbName>; @command (or @query)
		for i := range qs {
			qs[i].Query = dbSelector(m.Config.Driver, dbName) + " " + qs[i].Query
		}

		reported, err := m.fetch(ctx, db, reporter, qs)
		if err != nil {
			m.Logger().Warn("error while fetching:", err)
		}
		if !reported {
			m.Logger().Debug("error trying to emit event")
			return nil
		}
	}

	return nil
}

// fetchWithCursor executes the query with cursor-based incremental fetching.
// It uses the cursor manager to track the last fetched row and only retrieves new data.
//
// The context is wrapped with the module's configured timeout to prevent hung queries
// from blocking indefinitely and causing all subsequent collection cycles to be skipped
// via fetchMutex.TryLock(). The timeout defaults to the module's period if not set.
//
// Note: the timeout is applied here (cursor path only) rather than in Fetch() because
// the non-cursor path has never enforced a timeout. Applying it there would be a
// breaking change for existing users whose queries legitimately take longer than their
// configured period.
func (m *MetricSet) fetchWithCursor(ctx context.Context, reporter mb.ReporterV2) error {
	// Apply the module's configured timeout (defaults to period) to prevent hung queries.
	// Without this, a hung query blocks the goroutine indefinitely and all future
	// collection cycles are skipped via fetchMutex.TryLock().
	if timeout := m.Module().Config().Timeout; timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cursorVal := m.cursorManager.GetCurrentValue()

	m.Logger().Debugf("Executing query with cursor=%s", m.cursorManager.GetCurrentValueString())

	db, err := sql.NewDBClient(m.Config.Driver, m.HostData().URI, m.Logger())
	if err != nil {
		return fmt.Errorf("cannot open connection: %w", err)
	}
	defer db.Close()

	// Execute parameterized query with cursor value
	rows, err := db.FetchTableModeWithParams(ctx, m.translatedQuery, cursorVal)
	if err != nil {
		return fmt.Errorf("fetch with cursor failed: %w", err)
	}

	m.Logger().Debugf("Query returned %d rows", len(rows))

	if len(rows) == 0 {
		return nil
	}

	// Report events BEFORE updating cursor (at-least-once delivery)
	// This ensures we never lose data - if cursor update fails, we may
	// have duplicates but no data loss.
	for _, row := range rows {
		m.reportEvent(row, reporter, m.Config.Query)
	}

	// Update cursor state
	if err := m.cursorManager.UpdateFromResults(rows); err != nil {
		m.Logger().Warnf("Failed to save cursor state: %v", err)
		// Don't fail the fetch - events were already emitted
		// Next run will re-fetch some data (duplicates are better than data loss)
	}

	return nil
}

// reportEvent using 'user' mode with keys under `sql.metrics.*` or using Raw data mode (module and metricset key spaces
// provided by the user)
func (m *MetricSet) reportEvent(ms mapstr.M, reporter mb.ReporterV2, qry ...string) bool {
	var ok bool
	if m.Config.RawData.Enabled {
		// New usage.
		// Only driver & query field mapped.
		// metrics to be mapped by end user.
		if len(qry) > 0 {
			// set query.
			ok = reporter.Event(mb.Event{
				ModuleFields: mapstr.M{
					"metrics": ms, // Individual metric
					"driver":  m.Config.Driver,
					"query":   qry,
				},
			})
		} else {
			ok = reporter.Event(mb.Event{
				// Do not set query.
				ModuleFields: mapstr.M{
					"metrics": ms, // Individual metric
					"driver":  m.Config.Driver,
				},
			})
		}
	} else {
		// Previous usage. Backward compatibility.
		// Supports field mapping.
		ok = reporter.Event(mb.Event{
			ModuleFields: mapstr.M{
				"driver":  m.Config.Driver,
				"query":   qry,
				"metrics": inferTypeFromMetrics(ms),
			},
		})
	}
	return ok
}

// inferTypeFromMetrics to organize the output event into 'numeric', 'strings', 'floats' and 'boolean' values
// so we can dynamically map all fields inside those categories
func inferTypeFromMetrics(ms mapstr.M) mapstr.M {
	ret := mapstr.M{}

	numericMetrics := mapstr.M{}
	stringMetrics := mapstr.M{}
	boolMetrics := mapstr.M{}

	for k, v := range ms {
		switch v.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			numericMetrics[k] = v
		case string:
			stringMetrics[k] = v
		case bool:
			boolMetrics[k] = v
		case nil:
			// Ignore nil values as they cannot be indexed

		// TODO: Handle []interface{} properly; for now it is going to "string" field.
		// Keeping the behaviour as it is for now.
		//
		// case []interface{}:

		default:
			stringMetrics[k] = v
		}
	}

	// TODO: Ideally the field keys should have in sync with ES types like s/bool/boolean, etc.
	// But changing the field keys will be a breaking change. So, we are leaving it as it is.

	if len(numericMetrics) > 0 {
		ret["numeric"] = numericMetrics
	}

	if len(stringMetrics) > 0 {
		ret["string"] = stringMetrics
	}

	if len(boolMetrics) > 0 {
		ret["bool"] = boolMetrics
	}

	return ret
}
