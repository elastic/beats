// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package metrics

import (
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
)

var schemaMetrics = s.Schema{
	"commands": c.Dict("metrics.commands", s.Schema{
		"is_self":                 c.Dict("_isSelf", commandSchema),
		"aggregate":               c.Dict("aggregate", commandSchema),
		"build_info":              c.Dict("buildInfo", commandSchema),
		"coll_stats":              c.Dict("collStats", commandSchema),
		"connection_pool_stats":   c.Dict("connPoolStats", commandSchema),
		"count":                   c.Dict("count", commandSchema),
		"db_stats":                c.Dict("dbStats", commandSchema),
		"distinct":                c.Dict("distinct", commandSchema),
		"find":                    c.Dict("find", commandSchema),
		"get_cmd_line_opts":       c.Dict("getCmdLineOpts", commandSchema),
		"get_last_error":          c.Dict("getLastError", commandSchema),
		"get_log":                 c.Dict("getLog", commandSchema),
		"get_more":                c.Dict("getMore", commandSchema),
		"get_parameter":           c.Dict("getParameter", commandSchema),
		"host_info":               c.Dict("hostInfo", commandSchema),
		"insert":                  c.Dict("insert", commandSchema),
		"is_master":               c.Dict("isMaster", commandSchema),
		"last_collections":        c.Dict("listCollections", commandSchema),
		"last_commands":           c.Dict("listCommands", commandSchema),
		"list_databased":          c.Dict("listDatabases", commandSchema),
		"list_indexes":            c.Dict("listIndexes", commandSchema),
		"ping":                    c.Dict("ping", commandSchema),
		"profile":                 c.Dict("profile", commandSchema),
		"replset_get_rbid":        c.Dict("replSetGetRBID", commandSchema),
		"replset_get_status":      c.Dict("replSetGetStatus", commandSchema),
		"replset_heartbeat":       c.Dict("replSetHeartbeat", commandSchema),
		"replset_update_position": c.Dict("replSetUpdatePosition", commandSchema),
		"server_status":           c.Dict("serverStatus", commandSchema),
		"update":                  c.Dict("update", commandSchema),
		"whatsmyuri":              c.Dict("whatsmyuri", commandSchema),
	}),
	"cursor": c.Dict("metrics.cursor", s.Schema{
		"timed_out": c.Ifc("timedOut"),
		"open": c.Dict("open", s.Schema{
			"no_timeout": c.Ifc("noTimeout"),
			"pinned":     c.Ifc("pinned"),
			"total":      c.Ifc("total"),
		}),
	}),
	"document": c.Dict("metrics.document", s.Schema{
		"deleted":  c.Ifc("deleted"),
		"inserted": c.Ifc("inserted"),
		"returned": c.Ifc("returned"),
		"updated":  c.Ifc("updated"),
	}),
	"get_last_error": c.Dict("metrics.getLastError", s.Schema{
		"write_wait": c.Dict("wtime", s.Schema{
			"ms":    c.Ifc("totalMillis"),
			"count": c.Ifc("num"),
		}),
		"write_timeouts": c.Ifc("wtimeouts"),
	}),
	"operation": c.Dict("metrics.operation", s.Schema{
		"scan_and_order":  c.Ifc("scanAndOrder"),
		"write_conflicts": c.Ifc("writeConflicts"),
	}),
	"query_executor": c.Dict("metrics.queryExecutor", s.Schema{
		"scanned_indexes":   s.Object{"count": c.Ifc("scanned")},
		"scanned_documents": s.Object{"count": c.Ifc("scannedObjects")},
	}),
	"replication": c.Dict("metrics.repl", replicationSchema, c.DictOptional),
	"storage": c.Dict("metrics.storage.freelist", s.Schema{
		"search": c.Dict("search", s.Schema{
			"bucket_exhausted": c.Ifc("bucketExhausted"),
			"requests":         c.Ifc("requests"),
			"scanned":          c.Ifc("scanned"),
		}),
	}),
	"ttl": c.Dict("metrics.ttl", s.Schema{
		"deleted_documents": s.Object{"count": c.Ifc("deletedDocuments")},
		"passes":            s.Object{"count": c.Ifc("passes")},
	}),
}

var commandSchema = s.Schema{
	"failed": c.Ifc("failed"),
	"total":  c.Ifc("total"),
}

var replicationSchema = s.Schema{
	"executor": c.Dict("executor", s.Schema{
		"counters": c.Dict("counters", s.Schema{
			"event_created": c.Ifc("eventCreated"),
			"event_wait":    c.Ifc("eventWait"),
			"cancels":       c.Ifc("cancels"),
			"waits":         c.Ifc("waits"),
			"scheduled": s.Object{
				"netcmd":    c.Ifc("scheduledNetCmd"),
				"dbwork":    c.Ifc("scheduledDBWork"),
				"exclusive": c.Ifc("scheduledXclWork"),
				"work_at":   c.Ifc("scheduledWorkAt"),
				"work":      c.Ifc("scheduledWork"),
				"failures":  c.Ifc("schedulingFailures"),
			},
		}),
		"queues": c.Dict("queues", s.Schema{
			"in_progress": s.Object{
				"network":   c.Ifc("networkInProgress"),
				"dbwork":    c.Ifc("dbWorkInProgress"),
				"exclusive": c.Ifc("exclusiveInProgress"),
			},
			"sleepers": c.Ifc("sleepers"),
			"ready":    c.Ifc("ready"),
			"free":     c.Ifc("free"),
		}),
		"unsignaled_events": c.Ifc("unsignaledEvents"),
		"event_waiters":     c.Ifc("eventWaiters"),
		"shutting_down":     c.Bool("shuttingDown"),
		"network_interface": c.Str("networkIfcerface"),
	}),
	"apply": c.Dict("apply", s.Schema{
		"attempts_to_become_secondary": c.Ifc("attemptsToBecomeSecondary"),
		"batches":                      c.Dict("batches", countAndTimeSchema),
		"ops":                          c.Ifc("ops"),
	}),
	"buffer": c.Dict("buffer", s.Schema{
		"count":    c.Ifc("count"),
		"max_size": s.Object{"bytes": c.Ifc("maxSizeBytes")},
		"size":     s.Object{"bytes": c.Ifc("sizeBytes")},
	}),
	"initial_sync": c.Dict("initialSync", s.Schema{
		"completed":       c.Ifc("completed"),
		"failed_attempts": c.Ifc("failedAttempts"),
		"failures":        c.Ifc("failures"),
	}),
	"network": c.Dict("network", s.Schema{
		"bytes":          c.Ifc("bytes"),
		"getmores":       c.Dict("getmores", countAndTimeSchema),
		"ops":            c.Ifc("ops"),
		"reders_created": c.Ifc("readersCreated"),
	}),
	"preload": c.Dict("preload", s.Schema{
		"docs":    c.Dict("docs", countAndTimeSchema),
		"indexes": c.Dict("indexes", countAndTimeSchema),
	}),
}

var countAndTimeSchema = s.Schema{
	"count": c.Ifc("num"),
	"time":  s.Object{"ms": c.Ifc("totalMillis")},
}
