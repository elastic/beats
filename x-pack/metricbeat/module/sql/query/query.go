// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package query

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/jmoiron/sqlx"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("sql", "query", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	Driver     string
	Datasource string
	Query      string
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The sql query metricset is beta.")

	config := struct {
		Driver     string `config:"driver"`
		Datasource string `config:"datasource"`
		Query      string `config:"sql_query"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		Driver:        config.Driver,
		Datasource:    config.Datasource,
		Query:         config.Query,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	db, err := sqlx.Open(m.Driver, m.Datasource)
	if err != nil {
		return errors.Wrap(err, "error opening connection")
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		return errors.Wrap(err, "error testing connection")
	}

	rows, err := db.Queryx(m.Query)
	if err != nil {
		return errors.Wrap(err, "error executing query")
	}
	defer rows.Close()

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

	numericValue := common.MapStr{}

	stringValue := common.MapStr{}

	for rows.Next() {
		err = rows.Scan(vals...)
		if err != nil {
			m.Logger().Debug(errors.Wrap(err, "error trying to scan rows"))
			continue
		}

		data := common.MapStr{
			"metrics": common.MapStr{},
		}
		data.Put("driver", m.Driver)
		data.Put("query", m.Query)

		for i := 0; i < len(vals); i++ {
			dataMetrics := data["metrics"].(common.MapStr)

			value := getValue(vals[i].(*interface{}))

			num, err := strconv.ParseFloat(value, 64)
			if err == nil {
				numericValue.Put(cols[i], num)
				dataMetrics.Put("numeric", numericValue)
			} else {
				stringValue.Put(cols[i], value)
				dataMetrics.Put("string", stringValue)
			}

			report.Event(mb.Event{
				RootFields: common.MapStr{"sql": data},
			})
		}
	}

	if rows.Err() != nil {
		m.Logger().Debug(errors.Wrap(err, "error trying to read rows"))
	}

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
		return v.Format("2006-01-02 15:04:05.999")
	default:
		return fmt.Sprint(v)
	}
}
