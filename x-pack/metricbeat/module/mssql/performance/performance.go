// +build integration

package performance

import (
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/mssql"
	"github.com/pkg/errors"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("mssql", "performance", New)
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
	cfgwarn.Experimental("The mssql performance metricset is experimental.")

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
		`SELECT [cntr_value] as page_life_expectancy FROM sys.dm_os_performance_counters WHERE [object_name] = 'SQLServer:Buffer Manager' AND [counter_name] = 'Page life expectancy'`,
		`SELECT (a.cntr_value * 1.0 / b.cntr_value) * 100.0 as buffer_cache_hit_ratio FROM sys.dm_os_performance_counters a JOIN  (SELECT cntr_value,OBJECT_NAME FROM sys.dm_os_performance_counters WHERE counter_name = 'Buffer cache hit ratio base' AND OBJECT_NAME = 'SQLServer:Buffer Manager') b ON  a.OBJECT_NAME = b.OBJECT_NAME WHERE a.counter_name = 'Buffer cache hit ratio' AND a.OBJECT_NAME = 'SQLServer:Buffer Manager';`,
		"SELECT cntr_value as batch_req_sec FROM sys.dm_os_performance_counters WHERE counter_name = 'Batch Requests/sec';",
		"SELECT cntr_value as transactions_sec, instance_name as db FROM sys.dm_os_performance_counters where counter_name = 'Transactions/sec';",
		"SELECT cntr_value as compilations_sec FROM sys.dm_os_performance_counters where counter_name = 'SQL Compilations/sec';",
		"SELECT cntr_value as recompilations_sec FROM sys.dm_os_performance_counters where counter_name = 'SQL Re-Compilations/sec';",
		"SELECT cntr_value as user_connections FROM sys.dm_os_performance_counters WHERE counter_name = 'User Connections';",
		"SELECT cntr_value as lock_waits_sec FROM sys.dm_os_performance_counters WHERE counter_name = 'Lock Waits/sec' and instance_name = '_Total';",
		"SELECT cntr_value as page_splits_sec FROM sys.dm_os_performance_counters WHERE counter_name = 'Page splits/sec'",
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
