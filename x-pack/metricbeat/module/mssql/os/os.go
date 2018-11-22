// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package os

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/mssql"
)

func init() {
	mb.Registry.MustAddMetricSet("mssql", "os", New,
		mb.WithHostParser(mssql.HostParser))
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	log     *logp.Logger
	fetcher *mssql.Fetcher
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The mssql os metricset is experimental.")

	logger := logp.NewLogger("mssql.db").With("host", base.HostData().SanitizedURI)

	fetcher, err := mssql.NewFetcher(base.HostData().URI, []string{
		`SELECT * FROM sys.dm_os_sys_info;`,
		`SELECT * FROM sys.dm_os_sys_memory;`,
		`SELECT DATEDIFF(SECOND, sqlserver_start_time, SYSDATETIME()) AS [uptime_seconds] FROM sys.dm_os_sys_info;`,
		`SELECT DB_NAME(vfs.DbId) AS [db_name], SUM(vfs.IoStallReadMS) AS [io_stall_read_milliseconds], SUM(vfs.IoStallWriteMS) AS [io_stall_write_milliseconds] FROM fn_virtualfilestats(NULL, NULL) vfs INNER JOIN sys.master_files mf ON mf.database_id = vfs.DbId AND mf.FILE_ID = vfs.FileId GROUP BY DB_NAME(vfs.DbId);`,
	}, &schema, logger)
	if err != nil {
		return nil, errors.Wrap(err, "error creating fetcher")
	}

	return &MetricSet{
		BaseMetricSet: base,
		log:           logger,
		fetcher:       fetcher,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch(reporter mb.ReporterV2) {
	m.fetcher.Report(reporter)
}
