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
	s "github.com/elastic/beats/v7/libbeat/common/schema"
)

type rowCounter = func(rows *sql.Rows, mapStr *mapstr.M) ( error)

type databaseCounter struct {
	objectName   string
	instanceName string
	counterName  string
	counterValue *int64
}

// init registers the MetricSet with the central registry AS soon AS the program
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
	mapStr := mapstr.M{}
	var err error

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

	err = m.reportEvent(mapStr, reporter, schema)
	if err != nil {
		m.log.Error(fmt.Errorf("error applying schema %w", err))
	}

	connectionPctStr := m.fetchConnectionsPct(reporter)
	err = m.reportEvent(connectionPctStr, reporter, databaseNetworkSchema)
	if err != nil {
		m.log.Error(fmt.Errorf("error applying schema %w", err))
	}

	ioWaitStrs := m.fetchIOWait(reporter)
	err = m.reportEvents(ioWaitStrs, reporter, ioWaitSchema)
	if err != nil {
		m.log.Error(fmt.Errorf("error applying io wait schema %w", err))
	}

	diskReadWriteBytesStrs := m.fetchDiskReadWriteBytes(reporter)
	err = m.reportEvents(diskReadWriteBytesStrs, reporter, diskReadWriteBytesSchema)
	if err != nil {
		m.log.Error(fmt.Errorf("error applying disk read write bytes schema %w", err))
	}

	tableUsedSpaceStrs := m.fetchTableUsedSpace(reporter)
	err = m.reportEvents(tableUsedSpaceStrs, reporter, tableSpaceSchema)
	if err != nil {
		m.log.Error(fmt.Errorf("error applying table space schema %w", err))
	}

	dbNetworkBytesStrs := m.fetchDatabaseNetworkBytes(reporter)
	err = m.reportEvents(dbNetworkBytesStrs, reporter, databaseNetworkSchema)
	if err != nil {
		m.log.Error(fmt.Errorf("error applying table space schema %w", err))
	}

	tableIndexSizeStrs := m.fetchTableIndexSize(reporter)
	err = m.reportEvents(tableIndexSizeStrs, reporter, tableIndexSchema)
	if err != nil {
		m.log.Error(fmt.Errorf("error applying table index schema %w", err))
	}

	logSizeStrs := m.fetchLogSize(reporter)
	err = m.reportEvents(logSizeStrs, reporter, tableLogSchema)
	if err != nil {
		m.log.Error(fmt.Errorf("error applying table log schema %w", err))
	}

	blockCountStrs := m.fetchBlockCount(reporter)
	err = m.reportEvents(blockCountStrs, reporter, databaseSessionSchema)
	if err != nil {
		m.log.Error(fmt.Errorf("error applying block count schema %w", err))
	}

}

func (m *MetricSet) reportEvent(mapStr mapstr.M, reporter mb.ReporterV2, schema s.Schema) error {
	res, err := schema.Apply(mapStr)
	if err != nil {
		return err
	}

	if isReported := reporter.Event(mb.Event{
		MetricSetFields: res,
	}); !isReported {
		m.log.Debug("event not reported")
	}
	return nil
}

func (m *MetricSet) reportEvents(mapStrs []mapstr.M, reporter mb.ReporterV2, schema s.Schema) error {
	for _, mapStr := range mapStrs {
		res, err := schema.Apply(mapStr)
		if err != nil {
			m.log.Error(fmt.Errorf("error applying schema %w", err))
			return err
		}
		if isReported := reporter.Event(mb.Event{
			MetricSetFields: res,
		}); !isReported {
			m.log.Debug("event not reported")
		}
	}
	return nil
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

func (m *MetricSet) fetchBlockCount(reporter mb.ReporterV2) []mapstr.M {
	query := `
SELECT
    DB_NAME(s.database_id) AS db_name,
    count(*) AS blocked_session_count
from sys.dm_exec_requests AS r1
JOIN sys.dm_os_waiting_tasks AS w ON r1.session_id = w.session_id
JOIN sys.dm_exec_sessions AS s ON r1.session_id = s.session_id
GROUP BY s.database_id;`

	type blockCountCounter struct {
		dbName string
		blockedSessionCount *int64
	}

	var counter rowCounter = func(rows *sql.Rows, mapStr *mapstr.M) error {
		var err error
		var row blockCountCounter
		if err = rows.Scan(&row.dbName, &row.blockedSessionCount); err != nil {
			reporter.Error(fmt.Errorf("error scanning rows %w", err))
			return err
		}
		(*mapStr)[row.dbName] = fmt.Sprintf("%v", *row.blockedSessionCount)
		return nil
	}
	mapStr := m.fetchRowsWithRowCounter(query, reporter, counter)
	result := make([]mapstr.M, 0)
	for dbName, item := range mapStr {
		result = append(result, mapstr.M{
			"session_block_count": item,
			"db_name": dbName,
		})
	}
	return result
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
		"PageFaultCount": fmt.Sprintf("%v", pageFaultCount),
	}
}

func (m *MetricSet) fetchIOWait(reporter mb.ReporterV2) []mapstr.M {
	query :=`SELECT
    DB_NAME(fs.database_id) AS db_name,
--     CAST(SUM(fs.io_stall)/ 1000.0 AS DECIMAL(18,2)) AS total_io_stall_ms,
--     CAST(SUM(fs.io_stall_read_ms + fs.io_stall_write_ms)/1000.0 AS DECIMAL(18,2)) AS total_io_stall_read_write_ms,
    CAST((SUM(fs.io_stall_read_ms + fs.io_stall_write_ms) / NULLIF(SUM(fs.num_of_reads + fs.num_of_writes), 0)) / 1000.0 AS DECIMAL(18,2)) AS avg_io_stall_read_write_ms
FROM
    sys.dm_io_virtual_file_stats(NULL, NULL) AS fs
INNER JOIN sys.database_files AS df ON df.file_id = fs.file_id
GROUP BY fs.database_id;`

	type ioRow struct {
		dbName string
		avgIOWait *float64
	}
	var counter rowCounter = func(rows *sql.Rows, mapStr *mapstr.M) error {
		var err error
		var row ioRow
		if err = rows.Scan(&row.dbName, &row.avgIOWait); err != nil {
			reporter.Error(fmt.Errorf("error scanning rows %w", err))
			return err
		}
		(*mapStr)[row.dbName] = fmt.Sprintf("%v", *row.avgIOWait)
		return nil
	}
	mapStr := m.fetchRowsWithRowCounter(query, reporter, counter)

	result := make([]mapstr.M, 0)
	for dbName, val := range mapStr {
		result = append(result, mapstr.M{
			"io_wait": val,
			"db_name": dbName,
		})
	}

	return result
}

func (m *MetricSet) fetchDiskReadWriteBytes(reporter mb.ReporterV2) []mapstr.M {
	query := `SELECT
    DB_NAME(vfs.database_id) AS db_name,
    mf.physical_name AS disk_name,
    mf.type_desc,
    vfs.num_of_bytes_read AS byte_reads,
    vfs.num_of_bytes_written AS byte_writes,
    CONVERT(FLOAT, (vfs.io_stall / (vfs.num_of_reads+vfs.num_of_writes))) / 1000.0 as avg_mill_seconds_per_io
    FROM sys.dm_io_virtual_file_stats(NULL, NULL ) AS vfs
INNER JOIN
        sys.master_files AS mf ON vfs.database_id = mf.database_id AND vfs.file_id = mf.file_id;`

	type diskRWBytesCounter struct {
		dbName string
		diskFileName string
		typeDesc string
		numOfReadBytes *int64
		numOfWrittenBytes *int64
		avgMilliSecondsPerIO *float64
	}
	var counter rowCounter = func(rows *sql.Rows, mapStr *mapstr.M) error {
		var err error
		var row diskRWBytesCounter
		if err = rows.Scan(&row.dbName, &row.diskFileName, &row.typeDesc, &row.numOfReadBytes, &row.numOfWrittenBytes, &row.avgMilliSecondsPerIO); err != nil {
			reporter.Error(fmt.Errorf("error scanning rows %w", err))
			return err
		}
		diskInputKey := fmt.Sprintf("%s-%s-%s-%s", row.dbName, row.diskFileName, row.typeDesc, "disk_input")
		(*mapStr)[diskInputKey] = fmt.Sprintf("%v", *row.numOfReadBytes)
		diskOutputKey := fmt.Sprintf("%s-%s-%s-%s", row.dbName, row.diskFileName, row.typeDesc, "disk_output")
		(*mapStr)[diskOutputKey] = fmt.Sprintf("%v", *row.numOfWrittenBytes)
		diskIOAvgkey := fmt.Sprintf("%s-%s-%s-%s", row.dbName, row.diskFileName, row.typeDesc, "disk_io_avg_milli_second")
		(*mapStr)[diskIOAvgkey] = fmt.Sprintf("%v", *row.avgMilliSecondsPerIO)
		return nil
	}
	mapStr := m.fetchRowsWithRowCounter(query, reporter, counter)
	result := make([]mapstr.M, 0)
	for key, item := range mapStr {
		keys := strings.SplitN(key, "-", 4)
		if len(keys) != 4 {
			reporter.Error(fmt.Errorf("split disk io bytes key failed, key=%s", key))
			continue
		}
		dbName, diskFileName, typeDesc, metricName := keys[0], keys[1], keys[2], keys[3]
		result = append(result, mapstr.M{
			metricName: item,
			"db_name": dbName,
			"disk_file": diskFileName,
			"type_desc": typeDesc,
		})
	}

	return result
}

func (m *MetricSet) fetchDatabaseNetworkBytes(reporter mb.ReporterV2) []mapstr.M {
	query := `
SELECT
    DB_NAME(DB_ID()) AS db_name,
    COUNT(*) AS connection_count,
    sum(num_reads) AS total_reads,
    sum(num_writes) AS total_writes,
    sum(num_reads * net_packet_size) AS total_reads_bytes,
    sum(num_writes * net_packet_size) AS total_written_bytes
--     sum(num_reads * net_packet_size / 1024.0) AS total_reads_kb,
--     sum(num_writes * net_packet_size / 1024.0) AS total_writes_kb,
FROM
    sys.dm_exec_connections;`

	type dbIOCounter struct {
		dbName string
		connectionCount *int64
		totalReads *int64
		totalWrites *int64
		totalReadsBytes *int64
		totalWrittenBytes *int64
	}
	var counter rowCounter = func(rows *sql.Rows, mapStr *mapstr.M) error {
		var err error
		var row dbIOCounter
		if err = rows.Scan(&row.dbName, &row.connectionCount, &row.totalReads, &row.totalWrites, &row.totalReadsBytes, &row.totalWrittenBytes); err != nil {
			reporter.Error(fmt.Errorf("error scanning rows %w", err))
			return err
		}
		inputBytesKey := fmt.Sprintf("%s-%s", row.dbName, "network_input_bytes")
		(*mapStr)[inputBytesKey] = fmt.Sprintf("%v", *row.totalReadsBytes)
		outputBytesKey := fmt.Sprintf("%s-%s", row.dbName, "network_output_bytes")
		(*mapStr)[outputBytesKey] = fmt.Sprintf("%v", *row.totalWrittenBytes)
		return nil
	}
	mapStr := m.fetchRowsWithRowCounter(query, reporter, counter)

	result := make([]mapstr.M, 0)
	for key, item := range mapStr {
		keys := strings.SplitN(key, "-", 2)
		if len(keys) != 2 {
			reporter.Error(fmt.Errorf("fetch db io bytes key failed, err=%s", key))
			continue
		}
		dbName, metricName := keys[0], keys[1]
		result = append(result, mapstr.M{
			metricName: item,
			"db_name": dbName,
		})
	}
	return result
}

func (m *MetricSet) fetchConnectionsPct(reporter mb.ReporterV2) mapstr.M {
	maxConnections := m.fetchMaxConnections(reporter)

	userConnections := m.fetchUserConnections(reporter)

	if maxConnections < 0 {
		return mapstr.M{
			"connections_used_pct": "-1",
		}
	} else if maxConnections == 0 {
		return mapstr.M{
			"connections_used_pct": "0",
		}
	} else {
		return mapstr.M{
			"connections_used_pct": fmt.Sprintf("%v", float64(userConnections) / float64(maxConnections)),
		}
	}

}
func (m *MetricSet) fetchUserConnections(reporter mb.ReporterV2) int64 {
	query := `select CONVERT(int, cntr_value) from sys.dm_os_performance_counters where counter_name = 'User Connections';`
	type maxConRow struct {
		userConnections *int64
	}
	var counter rowCounter = func(rows *sql.Rows, mapStr *mapstr.M) error {
		var err error
		var row maxConRow
		if err = rows.Scan(&row.userConnections); err != nil {
			return err
		}
		(*mapStr)["userConnections"] = *row.userConnections
		return err
	}
	mapStr := m.fetchRowsWithRowCounter(query, reporter, counter)
	return mapStr["userConnections"].(int64)
}


func (m *MetricSet) fetchMaxConnections(reporter mb.ReporterV2) int64 {
	queryMaxConnections := `SELECT CONVERT(int, @@MAX_CONNECTIONS) AS 'Max Connections';`
	type maxConRow struct {
		MaxConnection *int64
	}
	var counter rowCounter = func(rows *sql.Rows, mapStr *mapstr.M) error {
		var err error
		var row maxConRow
		if err = rows.Scan(&row.MaxConnection); err != nil {
			return err
		}
		(*mapStr)["maxConnection"] = *row.MaxConnection
		return err
	}
	mapStr := m.fetchRowsWithRowCounter(queryMaxConnections, reporter, counter)
	return mapStr["maxConnection"].(int64)
}

func (m *MetricSet) fetchTableUsedSpace(reporter mb.ReporterV2) []mapstr.M {

	allDbs := m.fetchAllDbs(reporter)

	if len(allDbs) == 0 {
		reporter.Error(fmt.Errorf("cannot access any database"))
		return []mapstr.M{}
	}

	dbNames := make([]string, 0, len(allDbs))
	for dbName := range allDbs {
		dbNames = append(dbNames, "'" + dbName + "'")
	}
	dbNameWhereCond := strings.Join(dbNames, ", ")

	query := fmt.Sprintf(`SELECT
    Schemas.TABLE_CATALOG AS 'db_name',
    t.NAME AS 'tb_name',
    SUM(a.total_pages) * 8 AS TotalSpaceKB,
    SUM(a.used_pages) * 8 AS UsedSpaceKB,
    (SUM(a.total_pages) - SUM(a.used_pages)) * 8 AS UnusedSpaceKB
FROM
    sys.tables t
INNER JOIN
    sys.indexes i ON t.OBJECT_ID = i.object_id
INNER JOIN
    sys.partitions p ON i.object_id = p.OBJECT_ID AND i.index_id = p.index_id
INNER JOIN
    sys.allocation_units a ON p.partition_id = a.container_id
INNER JOIN
    INFORMATION_SCHEMA.TABLES AS Schemas ON Schemas.TABLE_NAME = t.name
WHERE
	Schemas.TABLE_CATALOG IN (%s)
GROUP BY
    t.Name, p.Rows, Schemas.TABLE_CATALOG`, dbNameWhereCond)

	type tableSpaceRow struct {
		dbName string
		tableName string
		totalSpaceKB *int64
		usedSpaceKB *int64
		unusedSpaceKB *int64
	}

	var counter rowCounter = func(rows *sql.Rows, mapStr *mapstr.M) error {
		var err error
		var row tableSpaceRow
		if err = rows.Scan(&row.dbName, &row.tableName, &row.totalSpaceKB, &row.usedSpaceKB, &row.unusedSpaceKB); err != nil {
			reporter.Error(fmt.Errorf("error scanning rows %w", err))
			return err
		}
		var spaceUsedPct float64 = 0.00
		if *row.totalSpaceKB != 0 {
			spaceUsedPct = float64(*row.usedSpaceKB) / float64(*row.totalSpaceKB)
		}
		key := fmt.Sprintf("%s-%s", row.dbName, row.tableName)
		(*mapStr)[key] = mapstr.M{
			"table_total_space": fmt.Sprintf("%v", *row.totalSpaceKB),
			"table_used_space": fmt.Sprintf("%v", *row.usedSpaceKB),
			"table_unused_space": fmt.Sprintf("%v", *row.unusedSpaceKB),
			"table_space_used_pct": fmt.Sprintf("%v", spaceUsedPct),
		}

		return nil
	}
	mapStr := m.fetchRowsWithRowCounter(query, reporter, counter)

	result := make([]mapstr.M, 0)
	for key, item := range mapStr {
		keys := strings.SplitN(key, "-", 2)
		if len(keys) != 2 {
			continue
		}
		dbName, tblName := keys[0], keys[1]
		var (
			dbTotalSpace int64
			dbUsedSpace int64
			dbUnusedSpace int64
			dbSpaceUsedPct float64 = 0.00
		)
		for metricName, val := range item.(mapstr.M) {
			result = append(result, mapstr.M{
				metricName: val,
				"db_name": dbName,
				"table_name": tblName,
			})
			if metricName == "table_total_space" {
				if v, err := strconv.Atoi(val.(string)); err == nil{
					dbTotalSpace += int64(v)
				} else {
					reporter.Error(fmt.Errorf("parse table %s-%s total space failed, val=%s", dbName, tblName, val))
				}
			} else if metricName == "table_used_space" {
				if v, err := strconv.Atoi(val.(string)); err == nil{
					dbUsedSpace += int64(v)
				} else {
					reporter.Error(fmt.Errorf("parse table %s-%s used space failed, val=%s", dbName, tblName, val))
				}
			} else if metricName == "table_unused_space" {
				if v, err := strconv.Atoi(val.(string)); err == nil{
					dbUnusedSpace += int64(v)
				} else {
					reporter.Error(fmt.Errorf("parse table %s-%s unused space failed, val=%s", dbName, tblName, val))
				}
			}
		}

		result = append(result, mapstr.M{
			"used_space": fmt.Sprintf("%v", dbUsedSpace),
			"db_name": dbName,
		})
		result = append(result, mapstr.M{
			"unused_space": fmt.Sprintf("%v", dbUnusedSpace),
			"db_name": dbName,
		})
		result = append(result, mapstr.M{
			"total_space": fmt.Sprintf("%v", dbTotalSpace),
			"db_name": dbName,
		})
		if dbTotalSpace != 0 {
			dbSpaceUsedPct = float64(dbUsedSpace) / float64(dbTotalSpace)
			result = append(result, mapstr.M{
				"space_used_pct": fmt.Sprintf("%v", dbSpaceUsedPct),
				"db_name": dbName,
			})
		}
	}

	return result
}

func (m *MetricSet) fetchAllDbs(reporter mb.ReporterV2) mapstr.M {
	queryAllDbs := `SELECT name FROM sys.databases;`

	type dbRow struct {
		name string
	}
	var allDbsCounter rowCounter = func(rows *sql.Rows, mapStr *mapstr.M) error {
		var err error
		var row dbRow
		if err = rows.Scan(&row.name); err != nil {
			reporter.Error(fmt.Errorf("search all databases failed, err=%s", err))
			return err
		}
		(*mapStr)[row.name] = mapstr.M{
			"db_name": row.name,
		}
		return nil
	}
	return m.fetchRowsWithRowCounter(queryAllDbs, reporter,allDbsCounter)
}

func (m *MetricSet) fetchTableIndexSize(reporter mb.ReporterV2) []mapstr.M {
	query := `
SELECT
    Scheme.TABLE_CATALOG AS dbName,
    t.[name] AS TableName,
    i.[name] AS IndexName,
    SUM(s.[used_page_count]) * 8 AS IndexSizeKB
FROM sys.dm_db_partition_stats AS s
INNER JOIN sys.indexes AS i ON s.[object_id] = i.[object_id]
    AND s.[index_id] = i.[index_id]
JOIN sys.tables AS t ON s.[object_id] = t.[object_id]
JOIN INFORMATION_SCHEMA.TABLES AS Scheme ON Scheme.TABLE_NAME = t.name
WHERE i.[name] IS NOT NULL
GROUP BY Scheme.TABLE_CATALOG, i.[name], t.[name]`

	type tableIndexRow struct {
		dbName string
		tableName string
		indexName string
		indexSizeKB *int64
	}

	var counter rowCounter = func(rows *sql.Rows, mapStr *mapstr.M) error {
		var err error
		var row tableIndexRow
		if err = rows.Scan(&row.dbName, &row.tableName, &row.indexName, &row.indexSizeKB); err != nil {
			reporter.Error(fmt.Errorf("error scanning rows %w", err))
			return err
		}
		// database_index_size
		// database_table_index_size
		key := fmt.Sprintf("%s-%s-%s", row.dbName, row.tableName, row.indexName)
		(*mapStr)[key] = *row.indexSizeKB
		return nil
	}
	mapStr := m.fetchRowsWithRowCounter(query, reporter, counter)

	tempRecords := make(map[string]map[string]int64) // dbName:tblName:indexSize
	for key, item := range mapStr {
		strs := strings.SplitN(key, "-", 3)
		if len(strs) != 3 {
			continue
		}
		dbName, tableName, _ := strs[0], strs[1], strs[2]
		if _, ok := tempRecords[dbName]; !ok {
			tempRecords[dbName] = make(map[string]int64)
			tempRecords[dbName][tableName] = 0
		}
		tempRecords[dbName][tableName] += item.(int64)
	}
	results := make([]mapstr.M, 0)
	for dbName, tables := range tempRecords {
		var dbIndexSize int64 = 0
		for tblName, indexSize := range tables {
			results = append(results, mapstr.M{
				"table_index_size": fmt.Sprintf("%v", indexSize),
				"table_name": tblName,
				"db_name": dbName,
			})
			dbIndexSize+=indexSize
		}
		results = append(results, mapstr.M{
			"index_size": fmt.Sprintf("%v", dbIndexSize),
			"db_name": dbName,
		})
	}

	return results
}

func (m *MetricSet) fetchLogSize(reporter mb.ReporterV2) []mapstr.M {
	query := `SELECT
     DB_NAME(database_id) AS db_name,
     CAST(SUM(CASE WHEN type_desc = 'LOG' THEN size END) * 8. AS INT) AS  log_size_kb
--      CAST(SUM(CASE WHEN type_desc = 'ROWS' THEN size END) * 8. AS DECIMAL(8,2)) AS row_size_mb,
--      CAST(SUM(size) * 8. / 1024 AS DECIMAL(8,2)) AS total_size_mb
FROM sys.master_files WITH(NOWAIT)
GROUP BY database_id`

	type logSizeRow struct {
		dbName string
		logSizeKB *int64
	}
	var counter rowCounter = func(rows *sql.Rows, mapStr *mapstr.M) error {
		var err error
		var row logSizeRow
		if err = rows.Scan(&row.dbName, &row.logSizeKB); err != nil {
			reporter.Error(fmt.Errorf("error scanning rows %w", err))
			return err
		}
		(*mapStr)[row.dbName] = mapstr.M{
			"log_size": *row.logSizeKB,
		}
		return nil
	}

	mapStr := m.fetchRowsWithRowCounter(query, reporter, counter)
	result := make([]mapstr.M, 0)
	for dbName, item := range mapStr {
		metric := item.(mapstr.M)
		result = append(result, mapstr.M{
			"log_size": fmt.Sprintf("%v", metric["log_size"]),
			"db_name": dbName,
		})
	}

	return result
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

func (m * MetricSet) fetchRowsWithRowCounter(query string, reporter mb.ReporterV2, counter rowCounter) mapstr.M {
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
		if err = counter(rows, &mapStr); err != nil {
			reporter.Error(fmt.Errorf("error scanning rows %w", err))
			continue
		}
	}
	return mapStr
}
