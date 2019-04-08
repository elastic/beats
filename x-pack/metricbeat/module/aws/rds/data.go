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
		"throughput.network_receive":     c.Float("NetworkReceiveThroughput"),
		"throughput.network_transmit":    c.Float("NetworkTransmitThroughput"),
		"oldest_replication_slot_lag.mb": c.Float("OldestReplicationSlotLag"),
		"read_io":                        c.Float("ReadIOPS"),
		"throughput": s.Object{
			"commit":  c.Float("CommitThroughput"),
			"delete":  c.Float("DeleteThroughput"),
			"dml":     c.Float("DMLThroughput"),
			"insert":  c.Float("InsertThroughput"),
			"network": c.Float("NetworkThroughput"),
			"read":    c.Float("ReadThroughput"),
			"select":  c.Float("SelectThroughput"),
			"update":  c.Float("UpdateThroughput"),
			"write":   c.Float("WriteThroughput"),
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
		},
		"replica_lag.sec": c.Float("ReplicaLag"),
		"disk_usage": s.Object{
			"bin_log.bytes":       c.Float("BinLogDiskUsage"),
			"replication_slot.mb": c.Float("ReplicationSlotDiskUsage"),
			"transaction_logs.mb": c.Float("TransactionLogsDiskUsage"),
		},
		"swap_usage.bytes":            c.Float("SwapUsage"),
		"transaction_logs_generation": c.Float("TransactionLogsGeneration"),
		"write_io":                    c.Float("WriteIOPS"),
		"queries":                     c.Float("Queries"),
		"deadlocks":                   c.Float("Deadlocks"),
		"volume_used.bytes":           c.Float("VolumeBytesUsed"),
		"free_local_storage":          c.Float("FreeLocalStorage"),
		"transactions": s.Object{
			"active":  c.Float("ActiveTransactions"),
			"blocked": c.Float("BlockedTransactions"),
		},
		"login_failures": c.Float("LoginFailures"),
	}
)
