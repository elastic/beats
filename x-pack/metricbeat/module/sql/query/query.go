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
	Enabled       bool   `config:"enabled"`
	RootLevelName string `config:"root_level_name"`
	DataLevelName string `config:"data_level_name"`
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
		evt, err := composeEventFromRoot(ms, m.config.RawData.RootLevelName, m.config.RawData.DataLevelName)
		if err != nil {
			m.Logger().Errorf("could not send event: '%w'", err)
			return
		}

		reporter.Event(*evt)
	} else {
		reporter.Event(*getUserEvent(ms, m.config.Driver, m.config.Query))
	}
}

// composeEventFromRoot using the provided metrics and organizing their position in the event by using the rootLevelName
// as the 'module' and the dataLevelName as the 'metricset'
func composeEventFromRoot(ms mapstr.M, rootLevelName, dataLevelName string) (*mb.Event, error) {
	// Check that we have every we need
	if rootLevelName == "" {
		return nil, fmt.Errorf("'raw_data.root_level_name' field is required in sql config file if raw_data is enabled")
	}

	if dataLevelName == "" {
		return nil, fmt.Errorf("'raw_data.data_level_name' field is required in sql config file if raw_data is enabled")
	}

	return &mb.Event{
		RootFields: mapstr.M{
			rootLevelName: mapstr.M{
				dataLevelName: ms,
			},
		},
		Namespace: dataLevelName,
		Service:   rootLevelName,
	}, nil
}

// getUserEvent from some metrics, organizing them into known "spaces" inside the event to map them without knowing
// their mapping in advance. To achieve this, all numeric values will go into `sql.metrics.numeric.*`, all string
// values into `sql.metrics.strings.*`, etc.
func getUserEvent(ms mapstr.M, driver, query string) *mb.Event {
	return &mb.Event{
		RootFields: mapstr.M{
			"sql": mapstr.M{
				"driver":  driver,
				"query":   query,
				"metrics": inferTypeFromMetrics(ms),
			},
		},
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
