// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package performance

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/elastic/beats/v8/libbeat/common"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/x-pack/metricbeat/module/mssql"
)

type performanceCounter struct {
	objectName   string
	instanceName string
	counterName  string
	counterValue *int64
}

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("mssql", "performance", New,
		mb.DefaultMetricSet(),
		mb.WithHostParser(mssql.HostParser))
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	log *logp.Logger
	db  *sql.DB
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logger := logp.NewLogger("mssql.performance").With("host", base.HostData().SanitizedURI)

	db, err := mssql.NewConnection(base.HostData().URI)
	if err != nil {
		return nil, errors.Wrap(err, "could not create connection to db")
	}

	return &MetricSet{
		BaseMetricSet: base,
		log:           logger,
		db:            db,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch(reporter mb.ReporterV2) {
	var err error
	var rows *sql.Rows
	rows, err = m.db.Query(`SELECT object_name,
       counter_name,
       instance_name,
       cntr_value
FROM   sys.dm_os_performance_counters
WHERE  counter_name = 'SQL Compilations/sec'
        OR counter_name = 'SQL Re-Compilations/sec'
        OR counter_name = 'User Connections'
        OR counter_name = 'Page splits/sec'
        OR ( counter_name = 'Lock Waits/sec'
             AND instance_name = '_Total' )
        OR counter_name = 'Page splits/sec'
        OR ( object_name = 'SQLServer:Buffer Manager'
             AND counter_name = 'Page life expectancy' )
        OR counter_name = 'Batch Requests/sec'
        OR ( counter_name = 'Buffer cache hit ratio'
             AND object_name = 'SQLServer:Buffer Manager' )
        OR ( counter_name = 'Target pages'
             AND object_name = 'SQLServer:Buffer Manager' )
        OR ( counter_name = 'Database pages'
             AND object_name = 'SQLServer:Buffer Manager' )
        OR ( counter_name = 'Checkpoint pages/sec'
             AND object_name = 'SQLServer:Buffer Manager' )
        OR ( counter_name = 'Lock Waits/sec'
             AND instance_name = '_Total' )
        OR ( counter_name = 'Transactions'
             AND object_name = 'SQLServer:General Statistics' )
        OR ( counter_name = 'Logins/sec'
             AND object_name = 'SQLServer:General Statistics' )
        OR ( counter_name = 'Logouts/sec'
             AND object_name = 'SQLServer:General Statistics' )
        OR ( counter_name = 'Connection Reset/sec'
             AND object_name = 'SQLServer:General Statistics' )
        OR ( counter_name = 'Active Temp Tables'
             AND object_name = 'SQLServer:General Statistics' )`)
	if err != nil {
		reporter.Error(errors.Wrapf(err, "error closing rows"))
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			m.log.Error("error closing rows: %s", err.Error())
		}
	}()

	mapStr := common.MapStr{}
	for rows.Next() {
		var row performanceCounter
		if err = rows.Scan(&row.objectName, &row.counterName, &row.instanceName, &row.counterValue); err != nil {
			reporter.Error(errors.Wrap(err, "error scanning rows"))
			continue
		}

		//cell values contains spaces at the beginning and at the end of the 'actual' value. They must be removed.
		row.counterName = strings.TrimSpace(row.counterName)
		row.instanceName = strings.TrimSpace(row.instanceName)
		row.objectName = strings.TrimSpace(row.objectName)

		if row.counterName == "Buffer cache hit ratio" {
			mapStr[row.counterName] = fmt.Sprintf("%v", float64(*row.counterValue)/100)
		} else {
			mapStr[row.counterName] = fmt.Sprintf("%v", *row.counterValue)
		}
	}

	res, err := schema.Apply(mapStr)
	if err != nil {
		m.log.Error(errors.Wrap(err, "error applying schema"))
		return
	}

	if isReported := reporter.Event(mb.Event{
		MetricSetFields: res,
	}); !isReported {
		m.log.Debug("event not reported")
	}
}

// Close closes the db connection to MS SQL at the Metricset level
func (m *MetricSet) Close() error {
	return m.db.Close()
}
