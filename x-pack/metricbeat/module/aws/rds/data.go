// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package rds

import (
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	schemaMetricSetFields = s.Schema{
		"burst_balance.pct": c.Float("BurstBalance"),
		"cpu": s.Object{
			"total": s.Object{
				"pct": c.Float("CPUUtilization"),
			},
			"credit_usage":   c.Float("CPUCreditUsage"),
			"credit_balance": c.Float("CPUCreditBalance"),
		},
		"database_connections":           c.Float("DatabaseConnections"),
		"disk_queue_depth":               c.Float("DiskQueueDepth"),
		"failed_sql_server_agent_jobs":   c.Float("FailedSQLServerAgentJobsCount"),
		"freeable_memory.bytes":          c.Float("FreeableMemory"),
		"free_storage.bytes":             c.Float("FreeStorageSpace"),
		"maximum_used_transaction_ids":   c.Float("MaximumUsedTransactionIDs"),
		"oldest_replication_slot_lag.mb": c.Float("OldestReplicationSlotLag"),
		"read.iops":            c.Float("ReadIOPS"),
		"throughput": s.Object{
			"commit":           c.Float("CommitThroughput"),
			"delete":           c.Float("DeleteThroughput"),
			"ddl":              c.Float("DDLThroughput"),
			"dml":              c.Float("DMLThroughput"),
			"insert":           c.Float("InsertThroughput"),
			"network":          c.Float("NetworkThroughput"),
			"network_receive":  c.Float("NetworkReceiveThroughput"),
			"network_transmit": c.Float("NetworkTransmitThroughput"),
			"read":             c.Float("ReadThroughput"),
			"select":           c.Float("SelectThroughput"),
			"update":           c.Float("UpdateThroughput"),
			"write":            c.Float("WriteThroughput"),
		},
		"latency": s.Object{
			"commit": c.Float("CommitLatency"),
			"ddl":    c.Float("DDLLatency"),
			"dml":    c.Float("DMLLatency"),
			"insert": c.Float("InsertLatency"),
			"read":   c.Float("ReadLatency"),
			"select": c.Float("SelectLatency"),
			"update": c.Float("UpdateLatency"),
			"write":  c.Float("WriteLatency"),
			"delete": c.Float("DeleteLatency"),
		},
		"replica_lag.sec": c.Float("ReplicaLag"),
		"disk_usage": s.Object{
			"bin_log.bytes":       c.Float("BinLogDiskUsage"),
			"replication_slot.mb": c.Float("ReplicationSlotDiskUsage"),
			"transaction_logs.mb": c.Float("TransactionLogsDiskUsage"),
		},
		"swap_usage.bytes":            c.Float("SwapUsage"),
		"transaction_logs_generation": c.Float("TransactionLogsGeneration"),
		"write.iops":        c.Float("WriteIOPS"),
		"queries":                     c.Float("Queries"),
		"deadlocks":                   c.Float("Deadlocks"),
		"volume_used.bytes":           c.Float("VolumeBytesUsed"),
		"free_local_storage.bytes":    c.Float("FreeLocalStorage"),
		"transactions": s.Object{
			"active":  c.Float("ActiveTransactions"),
			"blocked": c.Float("BlockedTransactions"),
		},
		"login_failures": c.Float("LoginFailures"),

		"db_instance.identifier":            c.Str("DBInstanceIdentifier"),
		"db_instance.db_cluster_identifier": c.Str("DBClusterIdentifier"),
		"db_instance.class":                 c.Str("DatabaseClass"),
		"db_instance.role":                  c.Str("Role"),
		"db_instance.engine_name":           c.Str("EngineName"),

		"aurora_bin_log_replica_lag": c.Float("AuroraBinlogReplicaLag"),

		"aurora_global_db.replicated_write_io.bytes": c.Float("AuroraGlobalDBReplicatedWriteIO"),
		"aurora_global_db.data_transfer.bytes":       c.Float("AuroraGlobalDBDataTransferBytes"),
		"aurora_global_db.replication_lag.ms":        c.Float("AuroraGlobalDBReplicationLag"),

		"aurora_replica.lag.ms":     c.Float("AuroraReplicaLag"),
		"aurora_replica.lag_max.ms": c.Float("AuroraReplicaLagMaximum"),
		"aurora_replica.lag_min.ms": c.Float("AuroraReplicaLagMinimum"),

		"backtrack_change_records.creation_rate": c.Float("BacktrackChangeRecordsCreationRate"),
		"backtrack_change_records.stored":        c.Float("BacktrackChangeRecordsStored"),

		"backtrack_window.actual": c.Float("BacktrackWindowActual"),
		"backtrack_window.alert":  c.Float("BacktrackWindowAlert"),

		"storage_used.backup_retention_period.bytes": c.Float("BackupRetentionPeriodStorageUsed"),
		"storage_used.snapshot.bytes":                c.Float("SnapshotStorageUsed"),

		"cache_hit_ratio.buffer":     c.Float("BufferCacheHitRatio"),
		"cache_hit_ratio.result_set": c.Float("ResultSetCacheHitRatio"),

		"engine_uptime.sec":                        c.Float("EngineUptime"),
		"volume.read.iops":                   c.Float("VolumeReadIOPs"),
		"volume.write.iops":                  c.Float("VolumeWriteIOPs"),
		"rds_to_aurora_postgresql_replica_lag.sec": c.Float("RDSToAuroraPostgreSQLReplicaLag"),
		"backup_storage_billed_total.bytes":        c.Float("TotalBackupStorageBilled"),
		"aurora_volume_left_total.bytes":           c.Float("AuroraVolumeBytesLeftTotal"),
	}
)
