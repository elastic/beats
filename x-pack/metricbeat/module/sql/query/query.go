// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package query

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
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
	db, err := m.DB()
	if err != nil {
		return errors.Wrap(err, "error opening connection")
	}

	rows, err := db.QueryxContext(ctx, m.Query)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}
	defer rows.Close()

	if m.ResponseFormat == tableResponseFormat {
		return m.fetchTableMode(rows, report)
	}

	return m.fetchVariableMode(rows, report)
}

// DB gets a client ready to query the database
func (m *MetricSet) DB() (*sqlx.DB, error) {
	if m.db == nil {
		db, err := sqlx.Open(m.Driver, m.HostData().URI)
		if err != nil {
			return nil, errors.Wrap(err, "opening connection")
		}
		err = db.Ping()
		if err != nil {
			return nil, errors.Wrap(err, "testing connection")
		}

		m.db = db
	}
	return m.db, nil
}

// fetchTableMode scan the rows and publishes the event for querys that return the response in a table format.
func (m *MetricSet) fetchTableMode(rows *sqlx.Rows, report mb.ReporterV2) error {

	// Extracted from
	// https://stackoverflow.com/questions/23507531/is-golangs-sql-package-incapable-of-ad-hoc-exploratory-queries/23507765#23507765
	cols, err := rows.Columns()
	if err != nil {
		return errors.Wrap(err, "error getting columns")
	}

	for k, v := range cols {
		cols[k] = strings.ToLower(v)
	}

	vals := make([]interface{}, len(cols))
	for i := 0; i < len(cols); i++ {
		vals[i] = new(interface{})
	}

	for rows.Next() {
		err = rows.Scan(vals...)
		if err != nil {
			m.Logger().Debug(errors.Wrap(err, "error trying to scan rows"))
			continue
		}

		numericMetrics := common.MapStr{}
		stringMetrics := common.MapStr{}

		for i := 0; i < len(vals); i++ {
			value := getValue(vals[i].(*interface{}))
			num, err := strconv.ParseFloat(value, 64)
			if err == nil {
				numericMetrics[cols[i]] = num
			} else {
				stringMetrics[cols[i]] = value
			}

		}

		report.Event(mb.Event{
			RootFields: common.MapStr{
				"sql": common.MapStr{
					"driver": m.Driver,
					"query":  m.Query,
					"metrics": common.MapStr{
						"numeric": numericMetrics,
						"string":  stringMetrics,
					},
				},
			},
		})
	}

	if err = rows.Err(); err != nil {
		m.Logger().Debug(errors.Wrap(err, "error trying to read rows"))
	}

	return nil
}

// fetchVariableMode scan the rows and publishes the event for querys that return the response in a key/value format.
func (m *MetricSet) fetchVariableMode(rows *sqlx.Rows, report mb.ReporterV2) error {
	data := common.MapStr{}
	for rows.Next() {
		var key string
		var val interface{}
		err := rows.Scan(&key, &val)
		if err != nil {
			m.Logger().Debug(errors.Wrap(err, "error trying to scan rows"))
			continue
		}

		key = strings.ToLower(key)
		data[key] = val
	}

	if err := rows.Err(); err != nil {
		m.Logger().Debug(errors.Wrap(err, "error trying to read rows"))
	}

	numericMetrics := common.MapStr{}
	stringMetrics := common.MapStr{}

	for key, value := range data {
		value := getValue(&value)
		num, err := strconv.ParseFloat(value, 64)
		if err == nil {
			numericMetrics[key] = num
		} else {
			stringMetrics[key] = value
		}
	}

	report.Event(mb.Event{
		RootFields: common.MapStr{
			"sql": common.MapStr{
				"driver": m.Driver,
				"query":  m.Query,
				"metrics": common.MapStr{
					"numeric": numericMetrics,
					"string":  stringMetrics,
				},
			},
		},
	})

	return nil
}

func getValue(pval *interface{}) string {
	switch v := (*pval).(type) {
	case nil:
		return "NULL"
	case bool:
		if v {
			return "true"
		}
		return "false"
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.RFC3339Nano)
	default:
		return fmt.Sprint(v)
	}
}

// Close closes the connection pool releasing its resources
func (m *MetricSet) Close() error {
	if m.db == nil {
		return nil
	}
	return errors.Wrap(m.db.Close(), "closing connection")
}
