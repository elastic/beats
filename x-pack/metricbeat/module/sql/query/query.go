// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package query

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

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
	Driver         string
	Query          string
	ResponseFormat string

	db *sqlx.DB
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The sql query metricset is beta.")

	config := struct {
		Driver         string `config:"driver" validate:"nonzero,required"`
		Query          string `config:"sql_query" validate:"nonzero,required"`
		ResponseFormat string `config:"sql_response_format"`
	}{ResponseFormat: tableResponseFormat}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if config.ResponseFormat != variableResponseFormat && config.ResponseFormat != tableResponseFormat {
		return nil, fmt.Errorf("invalid sql_response_format value: %s", config.ResponseFormat)
	}

	return &MetricSet{
		BaseMetricSet:  base,
		Driver:         config.Driver,
		Query:          config.Query,
		ResponseFormat: config.ResponseFormat,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
// It calls m.fetchTableMode() or m.fetchVariableMode() depending on the response
// format of the query.
func (m *MetricSet) Fetch(ctx context.Context, report mb.ReporterV2) error {
	db, err := sql.NewDBClient(m.Driver, m.HostData().URI, m.Logger())
	if err != nil {
		return errors.Wrap(err, "error opening connection")
	}
	defer db.Close()

	if m.ResponseFormat == tableResponseFormat {
		mss, err := db.FetchTableMode(ctx, m.Query)
		if err != nil {
			return err
		}

		for _, ms := range mss {
			report.Event(m.getEvent(ms))
		}

		return nil
	}

	ms, err := db.FetchVariableMode(ctx, m.Query)
	if err != nil {
		return err
	}
	report.Event(m.getEvent(ms))

	return nil
}

func (m *MetricSet) getEvent(ms mapstr.M) mb.Event {
	return mb.Event{
		RootFields: mapstr.M{
			"sql": mapstr.M{
				"driver":  m.Driver,
				"query":   m.Query,
				"metrics": getMetrics(ms),
			},
		},
	}
}

func getMetrics(ms mapstr.M) (ret mapstr.M) {
	ret = mapstr.M{}

	numericMetrics := mapstr.M{}
	stringMetrics := mapstr.M{}
	boolMetrics := mapstr.M{}

	for k, v := range ms {
		switch v.(type) {
		case float64:
			numericMetrics.Put(k, v)
		case string:
			stringMetrics.Put(k, v)
		case bool:
			boolMetrics.Put(k, v)
		case nil:
		//Ignore because a nil has no data type and thus cannot be indexed
		default:
			stringMetrics.Put(k, v)
		}
	}

	if len(numericMetrics) > 0 {
		ret.Put("numeric", numericMetrics)
	}

	if len(stringMetrics) > 0 {
		ret.Put("string", stringMetrics)
	}

	if len(boolMetrics) > 0 {
		ret.Put("bool", boolMetrics)
	}

	return
}

// Close closes the connection pool releasing its resources
func (m *MetricSet) Close() error {
	if m.db == nil {
		return nil
	}
	return errors.Wrap(m.db.Close(), "closing connection")
}
