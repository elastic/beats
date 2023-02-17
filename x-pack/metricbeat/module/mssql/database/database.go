package database


import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/mssql"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"strconv"
)

type databaseCounter struct {
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
	mb.Registry.MustAddMetricSet("mssql", "database", New,
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
	logger := logp.NewLogger("mssql.database").With("host", base.HostData().SanitizedURI)

	db, err := mssql.NewConnection(base.HostData().URI)
	if err != nil {
		return nil, fmt.Errorf("could not create connection to db %w", err)
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
	defer func() {
		if rows != nil {
			if err := rows.Close(); err != nil {
				m.log.Error("error closing rows: %s", err.Error())
			}
		}
	}()
	mapStr := mapstr.M{}

	tpsStr := m.fetchTps(reporter)
	mapStr.DeepUpdate(tpsStr)

	deadLockStr := m.fetchDeadLockCount(reporter)
	mapStr.DeepUpdate(deadLockStr)

	lockRequestTotalStr := m.fetchLockRequestTotal(reporter)
	mapStr.DeepUpdate(lockRequestTotalStr)

	tableFullScanTotalStr := m.fetchTableFullScanTotal(reporter)
	mapStr.DeepUpdate(tableFullScanTotalStr)

	planCacheHitRatioStr := m.fetchPlanCacheHitRatio(reporter)
	mapStr.DeepUpdate(planCacheHitRatioStr)

	pageFaultStr := m.fetchMemoryPageFault(reporter)
	mapStr.DeepUpdate(pageFaultStr)

	res, err := schema.Apply(mapStr)
	if err != nil {
		m.log.Error(fmt.Errorf("error applying schema %w", err))
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

func (m *MetricSet) fetchTps(reporter mb.ReporterV2) (mapstr.M) {
	query := `SELECT
    object_name,
    instance_name,
    counter_name,
    cntr_value
FROM sys.dm_os_performance_counters
WHERE
    object_name LIKE '%Transactions%' AND
    counter_name = 'Transactions';`
	return m.fetchRow(query, reporter)
}

func (m *MetricSet) fetchDeadLockCount(reporter mb.ReporterV2) (mapstr.M) {
	query := `SELECT
    object_name,
    instance_name,
    counter_name,
    cntr_value
FROM
    sys.dm_os_performance_counters
WHERE
    object_name LIKE '%Locks%' AND
    counter_name = 'Number of Deadlocks/sec' AND
    instance_name = '_Total';`
	return m.fetchRow(query, reporter)
}

func (m *MetricSet) fetchLockRequestTotal(reporter mb.ReporterV2) (mapstr.M) {
	query := `SELECT
    object_name,
    instance_name,
    counter_name,
    cntr_value
FROM
    sys.dm_os_performance_counters
WHERE
    object_name LIKE '%Locks%' AND
    counter_name = 'Lock Requests/sec' AND
    instance_name = '_Total';`
	return m.fetchRow(query, reporter)
}

func (m *MetricSet) fetchTableFullScanTotal(reporter mb.ReporterV2) mapstr.M {
	query := `SELECT
    object_name,
    instance_name,
    counter_name,
    cntr_value
FROM
    sys.dm_os_performance_counters
WHERE
    object_name LIKE '%Access Methods%' AND
    counter_name='Full Scans/sec';`
	return m.fetchRow(query, reporter)
}

func (m *MetricSet) fetchPlanCacheHitRatio(reporter mb.ReporterV2) mapstr.M {
	query := `
SELECT
    object_name,
    instance_name,
    counter_name,
    cntr_value
FROM
    sys.dm_os_performance_counters
WHERE
    object_name LIKE '%Plan Cache%' AND
    counter_name LIKE '%Cache Hit Ratio%' AND
    instance_name = '_Total';`
	mapStr := m.fetchRows(query, reporter)

	result := mapstr.M{}
	hitRatio, err := strconv.Atoi(mapStr["Cache Hit Ratio"].(string))
	if err != nil {
		reporter.Error(fmt.Errorf("parse cache hit ratio from string to int failed, val=%s", mapStr["Cache Hit Ratio"].(string)))
		return result
	}
	hitRatioBase, err := strconv.Atoi(mapStr["Cache Hit Ratio Base"].(string))
	if err != nil {
		reporter.Error(fmt.Errorf("parse cache hit ratio from string to int failed, val=%s", mapStr["Cache Hit Ratio Base"].(string)))
		return result
	}

	return mapstr.M{
		"PlanCacheHitRatio": fmt.Sprintf("%f", float64(hitRatio)/float64(hitRatioBase) * 100),
	}
}

func (m *MetricSet) fetchMemoryPageFault(reporter mb.ReporterV2) mapstr.M {
	query := `SELECT page_fault_count FROM sys.dm_os_process_memory;`
	row := m.db.QueryRow(query)
	var pageFaultCount interface{}
	if err := row.Scan(&pageFaultCount); err != nil {
		reporter.Error(fmt.Errorf("error scanning rows %w", err))
		return mapstr.M{}
	}
	return mapstr.M{
		"PageFaultCount": fmt.Sprintf("%v", pageFaultCount.(int64)),
	}
}

func (m *MetricSet) fetchRow(query string, reporter mb.ReporterV2) mapstr.M {
	var (
		err error
		row *sql.Row
	)
	row = m.db.QueryRow(query)

	mapStr := mapstr.M{}
	var counter databaseCounter
	if err = row.Scan(&counter.objectName, &counter.instanceName, &counter.counterName, &counter.counterValue); err != nil {
		reporter.Error(fmt.Errorf("error scanning rows %w", err))
		return mapStr
	}

	counter.counterName = strings.TrimSpace(counter.counterName)
	mapStr[counter.counterName] = fmt.Sprintf("%v", *counter.counterValue)
	return mapStr
}

func (m *MetricSet) fetchRows(query string, reporter mb.ReporterV2) mapstr.M {
	var (
		err error
		rows *sql.Rows
	)

	mapStr := mapstr.M{}

	rows, err = m.db.Query(query)
	if err != nil {
		reporter.Error(fmt.Errorf("error closing rows %w", err))
		return mapStr
	}
	defer func() {
		if err := rows.Close(); err != nil {
			m.log.Error("error closing rows: %s", err.Error())
		}
	}()

	for rows.Next() {
		var row databaseCounter
		if err = rows.Scan(&row.objectName, &row.instanceName, &row.counterName, &row.counterValue); err != nil {
			reporter.Error(fmt.Errorf("error scanning rows %w", err))
			continue
		}

		//cell values contains spaces at the beginning and at the end of the 'actual' value. They must be removed.
		row.counterName = strings.TrimSpace(row.counterName)
		row.instanceName = strings.TrimSpace(row.instanceName)
		row.objectName = strings.TrimSpace(row.objectName)

		mapStr[row.counterName] = fmt.Sprintf("%v", *row.counterValue)
	}
	return mapStr
}
