// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package query

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

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

	// Support either the previous query / or the new list of queries.
	ResponseFormat string `config:"sql_response_format"`
	Query          string `config:"sql_query" `

	Queries      []query `config:"sql_queries" `
	MergeResults bool    `config:"merge_results"`
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	Config config
	db     *sqlx.DB
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
		// Backword compartibility, if no value is provided
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

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
// It calls m.fetchTableMode() or m.fetchVariableMode() depending on the response
// format of the query.
func (m *MetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) error {
	db, err := sql.NewDBClient(m.Config.Driver, m.HostData().URI, m.Logger())
	if err != nil {
		return fmt.Errorf("could not open connection: %w", err)
	}
	defer db.Close()

	queries := m.Config.Queries
	if len(queries) == 0 {
		one_query := query{Query: m.Config.Query, ResponseFormat: m.Config.ResponseFormat}
		queries = append(queries, one_query)
	}

	merged := mapstr.M{}

	for _, q := range queries {
		if q.ResponseFormat == tableResponseFormat {
			// Table format
			mss, err := db.FetchTableMode(ctx, q.Query)
			if err != nil {
				return fmt.Errorf("fetch table mode failed: %w", err)
			}

			for _, ms := range mss {
				if m.Config.MergeResults {
					if len(mss) > 1 {
						return fmt.Errorf("can not merge query resulting with more than one rows: %s", q)
					} else {
						for k, v := range ms {
							_, ok := merged[k]
							if ok {
								m.Logger().Warn("overwriting duplicate metrics: ", k)
							}
							merged[k] = v
						}
					}
				} else {
					// Report immediately for non-merged cases.
					m.reportEvent(ms, reporter, q.Query)
				}
			}
		} else {
			// Variable format
			ms, err := db.FetchVariableMode(ctx, q.Query)
			if err != nil {
				return fmt.Errorf("fetch variable mode failed: %w", err)
			}

			if m.Config.MergeResults {
				for k, v := range ms {
					_, ok := merged[k]
					if ok {
						m.Logger().Warn("overwriting duplicate metrics: ", k)
					}
					merged[k] = v
				}
			} else {
				// Report immediately for non-merged cases.
				m.reportEvent(ms, reporter, q.Query)
			}
		}
	}
	if m.Config.MergeResults {
		// Report here for merged case.
		m.reportEvent(merged, reporter, "")
	}

	return nil
}

// reportEvent using 'user' mode with keys under `sql.metrics.*` or using Raw data mode (module and metricset key spaces
// provided by the user)
func (m *MetricSet) reportEvent(ms mapstr.M, reporter mb.ReporterV2, qry string) {
	if m.Config.RawData.Enabled {

		// New usage.
		// Only driver & query field mapped.
		// metrics to be mapped by end user.
		if len(qry) > 0 {
			// set query.
			reporter.Event(mb.Event{
				ModuleFields: mapstr.M{
					"metrics": ms, // Individual metric
					"driver":  m.Config.Driver,
					"query":   qry,
				},
			})
		} else {
			reporter.Event(mb.Event{
				// Do not set query.
				ModuleFields: mapstr.M{
					"metrics": ms, // Individual metric
					"driver":  m.Config.Driver,
				},
			})

		}
	} else {
		// Previous usage. Backword compartibility.
		// Supports field mapping.
		reporter.Event(mb.Event{
			ModuleFields: mapstr.M{
				"driver":  m.Config.Driver,
				"query":   qry,
				"metrics": inferTypeFromMetrics(ms),
			},
		})
	}
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
		//Ignore because a nil has no data type and thus cannot be indexed
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

// Close closes the connection pool releasing its resources
func (m *MetricSet) Close() (err error) {
	if m.db == nil {
		return nil
	}
	return m.db.Close()
}
