// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package query

import (
	"context"
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/helper/sql"
	"github.com/elastic/beats/v7/metricbeat/mb"
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
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	Config config
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

	return b, nil
}

// queryDBNames returns the query to list databases present in a server
// as per the driver name. If the given driver is not supported, queryDBNames
// returns an empty query.
func queryDBNames(driver string) string {
	switch sql.SwitchDriverName(driver) {
	// NOTE: Add support for other drivers in future as when the need arises.
	// dbSelector function would also required to be modified in order to add
	// support for a new driver.
	case "mssql":
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
	case "mssql":
		return fmt.Sprintf("USE [%s];", dbName)
	}
	return ""
}

func (m *MetricSet) fetch(ctx context.Context, db *sql.DbClient, reporter mb.ReporterV2, queries []query) (bool, error) {
	var ok bool
	merged := make(mapstr.M, 0)
	for _, q := range queries {
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
		ok = m.reportEvent(merged, reporter, "")
	}

	return ok, nil
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
// It calls m.fetchTableMode() or m.fetchVariableMode() depending on the response
// format of the query.
func (m *MetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) error {
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

// reportEvent using 'user' mode with keys under `sql.metrics.*` or using Raw data mode (module and metricset key spaces
// provided by the user)
func (m *MetricSet) reportEvent(ms mapstr.M, reporter mb.ReporterV2, qry string) bool {
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
		case float64:
			numericMetrics[k] = v
		case string:
			stringMetrics[k] = v
		case bool:
			boolMetrics[k] = v
		case nil:
		// Ignore because a nil has no data type and thus cannot be indexed
		default:
			stringMetrics[k] = v
		}
	}

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
