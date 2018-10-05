// +build integration

package os

import (
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/mssql"
	"github.com/pkg/errors"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("mssql", "os", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	*mssql.MetricSet
	config *mssql.Config
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The mssql os metricset is experimental.")

	config := mssql.Config{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	metricSet, err := mssql.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating mssql metricset")
	}

	ms := &MetricSet{MetricSet: metricSet, config: &config}

	return ms, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) {
	fetcher, err := mssql.NewFetcher(m.config, []string{
		`SELECT * FROM sys.dm_os_sys_info;`,
		`SELECT * FROM sys.dm_os_sys_memory;`,
		`SELECT DATEDIFF(SECOND, sqlserver_start_time, SYSDATETIME()) AS [uptime_seconds] FROM sys.dm_os_sys_info;`,
		`SELECT DB_NAME(vfs.DbId) AS [db_name], SUM(vfs.IoStallReadMS) AS [io_stall_read_milliseconds], SUM(vfs.IoStallWriteMS) AS [io_stall_write_milliseconds] FROM fn_virtualfilestats(NULL, NULL) vfs INNER JOIN sys.master_files mf ON mf.database_id = vfs.DbId AND mf.FILE_ID = vfs.FileId GROUP BY DB_NAME(vfs.DbId);`,
	}, &schema)
	if err != nil {
		reporter.Error(errors.Wrap(err, "error creating fetcher"))
		return
	}
	defer fetcher.Close()

	if fetcher.Error != nil {
		reporter.Error(fetcher.Error)
	} else {
		for _, e := range fetcher.Maprs {
			reporter.Event(mb.Event{
				MetricSetFields: e,
			})
		}
	}
}
