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
		"queries":                     c.Float("Queries"),
		"deadlocks":                   c.Float("Deadlocks"),
		"volume_used.bytes":           c.Float("VolumeBytesUsed"),
		"free_local_storage":          c.Float("FreeLocalStorage"),
		"freeable_memory":             c.Float("FreeableMemory"),
		"throughput.select":           c.Float("SelectThroughput"),
		"cpu.total.pct":               c.Float("CPUUtilization"),
		"transactions.active":         c.Float("ActiveTransactions"),
		"throughput.dml":              c.Float("DMLThroughput"),
		"transactions.blocked":        c.Float("BlockedTransactions"),
		"throughput.network":          c.Float("NetworkThroughput"),
		"throughput.insert":           c.Float("InsertThroughput"),
		"latency.dml":                 c.Float("DMLLatency"),
		"latency.select":              c.Float("SelectLatency"),
		"throughput.network_transmit": c.Float("NetworkTransmitThroughput"),
		"throughput.delete":           c.Float("DeleteThroughput"),
		"throughput.commit":           c.Float("CommitThroughput"),
		"latency.update":              c.Float("UpdateLatency"),
		"throughput.update":           c.Float("UpdateThroughput"),
		"latency.ddl":                 c.Float("DDLLatency"),
		"login_failures":              c.Float("LoginFailures"),
		"throughput.network_receive":  c.Float("NetworkReceiveThroughput"),
		"latency.commit":              c.Float("CommitLatency"),
		"latency.insert":              c.Float("InsertLatency"),
	}
)
