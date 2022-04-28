// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transaction_log

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/mssql"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type dbInfo struct {
	id   int
	name string
}

type logSpace struct {
	id                             int
	totalLogSizeInBytes            int
	usedLogSpaceInBytes            int
	usedLogSpaceInPercent          float64
	logSpaceInBytesSinceLastBackup int
}

type logStats struct {
	databaseID            int
	sizeMB                float64
	activeSizeMB          float64
	backupTime            string
	sinceLastBackupMB     *float64
	sinceLastCheckpointMB float64
	recoverySizeMB        float64
}

func init() {
	mb.Registry.MustAddMetricSet("mssql", "transaction_log", New,
		mb.DefaultMetricSet(),
		mb.WithHostParser(mssql.HostParser))
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	log *logp.Logger
	db  *sql.DB
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logger := logp.NewLogger("mssql.transaction_log").With("host", base.HostData().SanitizedURI)

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
	dbs, err := m.getDbsNames()
	if err != nil {
		reporter.Error(err)
		return
	}

	for _, db := range dbs {
		moduleFields := mapstr.M{
			"database": mapstr.M{
				"id":   db.id,
				"name": db.name,
			},
		}
		metricsetFields := mapstr.M{}

		spaceUsage, err := m.getLogSpaceUsageForDb(db.name)
		if err != nil {
			reporter.Error(err)
		} else {
			metricsetFields["space_usage"] = spaceUsage
		}

		stats, err := m.getLogStats(db)
		if err != nil {
			reporter.Error(err)
		} else {
			metricsetFields["stats"] = stats
		}

		if len(metricsetFields) == 0 {
			m.log.Debug("no data to report")
			continue
		}

		// Both log space and log size are available, so report both
		if isReported := reporter.Event(mb.Event{
			ModuleFields:    moduleFields,
			MetricSetFields: metricsetFields,
		}); !isReported {
			m.log.Debug("event not reported")
		}

	}
}

// Close the connection to the server at the engine level
func (m *MetricSet) Close() error {
	return m.db.Close()
}

func (m *MetricSet) getLogSpaceUsageForDb(dbName string) (mapstr.M, error) {
	// According to MS docs a single result is always returned for this query
	row := m.db.QueryRow(fmt.Sprintf(`USE [%s]; SELECT * FROM sys.dm_db_log_space_usage;`, dbName))

	var res logSpace
	if err := row.Scan(&res.id, &res.totalLogSizeInBytes, &res.usedLogSpaceInBytes, &res.usedLogSpaceInPercent,
		&res.logSpaceInBytesSinceLastBackup); err != nil {
		// Because this query only returns a single result an error in the first scan is
		// probably a "data returned but not properly scanned"
		err = errors.Wrap(err, "error scanning single result")
		return nil, err
	}

	return mapstr.M{
		"total": mapstr.M{
			"bytes": res.totalLogSizeInBytes,
		},
		"used": mapstr.M{
			"bytes": res.usedLogSpaceInBytes,
			"pct":   res.usedLogSpaceInPercent,
		},
		"since_last_backup": mapstr.M{
			"bytes": res.logSpaceInBytesSinceLastBackup,
		},
	}, nil
}
func (m *MetricSet) getLogStats(db dbInfo) (mapstr.M, error) {
	// According to MS docs a single result is always returned for this query
	row := m.db.QueryRow(fmt.Sprintf(`USE [%s]; SELECT database_id,total_log_size_mb,active_log_size_mb,log_backup_time,log_since_last_log_backup_mb,log_since_last_checkpoint_mb,log_recovery_size_mb FROM sys.dm_db_log_stats(%d);`, db.name, db.id))

	var res logStats
	if err := row.Scan(&res.databaseID, &res.sizeMB, &res.activeSizeMB, &res.backupTime, &res.sinceLastBackupMB, &res.sinceLastCheckpointMB, &res.recoverySizeMB); err != nil {
		// Because this query only returns a single result an error in the first scan is
		// probably a "data returned but not properly scanned"
		err = errors.Wrap(err, "error scanning single result")
		return nil, err
	}

	result := mapstr.M{
		"total_size": mapstr.M{
			"bytes": res.sizeMB * 1048576,
		},
		"active_size": mapstr.M{
			"bytes": res.activeSizeMB * 1048576,
		},
		"backup_time": res.backupTime,
		"since_last_checkpoint": mapstr.M{
			"bytes": res.sinceLastCheckpointMB * 1048576,
		},
		"recovery_size": mapstr.M{
			"bytes": res.recoverySizeMB,
		},
	}

	if res.sinceLastBackupMB != nil {
		result.Put("since_last_backup.bytes", *res.sinceLastBackupMB*1048576)
	}

	return result, nil
}

func (m *MetricSet) getDbsNames() ([]dbInfo, error) {
	res := make([]dbInfo, 0)

	var rows *sql.Rows
	rows, err := m.db.Query("SELECT name, database_id FROM sys.databases")
	if err != nil {
		return nil, errors.Wrap(err, "error doing query 'SELECT name, database_id FROM sys.databases'")
	}
	defer closeRows(rows)

	for rows.Next() {
		var row dbInfo
		if err = rows.Scan(&row.name, &row.id); err != nil {
			return nil, errors.Wrap(err, "error scanning row results")
		}

		res = append(res, row)
	}

	return res, nil
}

func closeRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		logp.Err("error closing rows: %s", err)
	}
}
