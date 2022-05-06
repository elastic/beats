// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package query

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
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

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	config config

	db *sqlx.DB
}

type config struct {
	Driver         string  `config:"driver" validate:"nonzero,required"`
	Query          string  `config:"sql_query" validate:"nonzero,required"`
	ResponseFormat string  `config:"sql_response_format"`
	RawData        rawData `config:"raw_data"`
}

// rawData is the minimum required set of fields to generate fully customized events with their own module key space
// and their own metricset key space.
type rawData struct {
	Enabled bool `config:"enabled"`
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The sql query metricset is beta.")

	cfg := config{ResponseFormat: tableResponseFormat}

	if err := base.Module().UnpackConfig(&cfg); err != nil {
		return nil, err
	}

	if cfg.ResponseFormat != variableResponseFormat && cfg.ResponseFormat != tableResponseFormat {
		return nil, fmt.Errorf("invalid sql_response_format value: %s", cfg.ResponseFormat)
	}

	return &MetricSet{
		BaseMetricSet: base,
		config:        cfg,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
// It calls m.fetchTableMode() or m.fetchVariableMode() depending on the response
// format of the query.
func (m *MetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) error {
	db, err := sql.NewDBClient(m.config.Driver, m.HostData().URI, m.Logger())
	if err != nil {
		return fmt.Errorf("could not open connection: %w", err)
	}
	defer db.Close()

	if m.config.ResponseFormat == tableResponseFormat {
		mss, err := db.FetchTableMode(ctx, m.config.Query)
		if err != nil {
			return err
		}

		for _, ms := range mss {
			m.reportEvent(ms, reporter)
		}

		return nil
	}

	ms, err := db.FetchVariableMode(ctx, m.config.Query)
	if err != nil {
		return err
	}

	m.reportEvent(ms, reporter)

	return nil
}

// reportEvent using 'user' mode with keys under `sql.metrics.*` or using Raw data mode (module and metricset key spaces
// provided by the user)
func (m *MetricSet) reportEvent(ms mapstr.M, reporter mb.ReporterV2) {
	if m.config.RawData.Enabled {
		reporter.Event(mb.Event{
			ModuleFields: ms,
		})
	} else {
		reporter.Event(mb.Event{
			ModuleFields: mapstr.M{
				"driver":  m.config.Driver,
				"query":   m.config.Query,
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
