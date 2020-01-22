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
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
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
		"timed_out": c.Int("timedOut"),
		"open": c.Dict("open", s.Schema{
			"no_timeout": c.Int("noTimeout"),
			"pinned":     c.Int("pinned"),
			"total":      c.Int("total"),
		}),
	}),
	"document": c.Dict("metrics.document", s.Schema{
		"deleted":  c.Int("deleted"),
		"inserted": c.Int("inserted"),
		"returned": c.Int("returned"),
		"updated":  c.Int("updated"),
	}),
	"get_last_error": c.Dict("metrics.getLastError", s.Schema{
		"write_wait": c.Dict("wtime", s.Schema{
			"ms":    c.Int("totalMillis"),
			"count": c.Int("num"),
		}),
		"write_timeouts": c.Int("wtimeouts"),
	}),
	"operation": c.Dict("metrics.operation", s.Schema{
		"scan_and_order":  c.Int("scanAndOrder"),
		"write_conflicts": c.Int("writeConflicts"),
	}),
	"query_executor": c.Dict("metrics.queryExecutor", s.Schema{
		"scanned_indexes":   s.Object{"count": c.Int("scanned")},
		"scanned_documents": s.Object{"count": c.Int("scannedObjects")},
	}),
	"replication": c.Dict("metrics.repl", replicationSchema, c.DictOptional),
	"storage": c.Dict("metrics.storage.freelist", s.Schema{
		"search": c.Dict("search", s.Schema{
			"bucket_exhausted": c.Int("bucketExhausted"),
			"requests":         c.Int("requests"),
			"scanned":          c.Int("scanned"),
		}),
	}),
	"ttl": c.Dict("metrics.ttl", s.Schema{
		"deleted_documents": s.Object{"count": c.Int("deletedDocuments")},
		"passes":            s.Object{"count": c.Int("passes")},
	}),
}

var commandSchema = s.Schema{
	"failed": c.Int("failed"),
	"total":  c.Int("total"),
}

var replicationSchema = s.Schema{
	"executor": c.Dict("executor", s.Schema{
		"counters": c.Dict("counters", s.Schema{
			"event_created": c.Int("eventCreated"),
			"event_wait":    c.Int("eventWait"),
			"cancels":       c.Int("cancels"),
			"waits":         c.Int("waits"),
			"scheduled": s.Object{
				"netcmd":    c.Int("scheduledNetCmd"),
				"dbwork":    c.Int("scheduledDBWork"),
				"exclusive": c.Int("scheduledXclWork"),
				"work_at":   c.Int("scheduledWorkAt"),
				"work":      c.Int("scheduledWork"),
				"failures":  c.Int("schedulingFailures"),
			},
		}),
		"queues": c.Dict("queues", s.Schema{
			"in_progress": s.Object{
				"network":   c.Int("networkInProgress"),
				"dbwork":    c.Int("dbWorkInProgress"),
				"exclusive": c.Int("exclusiveInProgress"),
			},
			"sleepers": c.Int("sleepers"),
			"ready":    c.Int("ready"),
			"free":     c.Int("free"),
		}),
		"unsignaled_events": c.Int("unsignaledEvents"),
		"event_waiters":     c.Int("eventWaiters"),
		"shutting_down":     c.Bool("shuttingDown"),
		"network_interface": c.Str("networkInterface"),
	}),
	"apply": c.Dict("apply", s.Schema{
		"attempts_to_become_secondary": c.Int("attemptsToBecomeSecondary"),
		"batches":                      c.Dict("batches", countAndTimeSchema),
		"ops":                          c.Int("ops"),
	}),
	"buffer": c.Dict("buffer", s.Schema{
		"count":    c.Int("count"),
		"max_size": s.Object{"bytes": c.Int("maxSizeBytes")},
		"size":     s.Object{"bytes": c.Int("sizeBytes")},
	}),
	"initial_sync": c.Dict("initialSync", s.Schema{
		"completed":       c.Int("completed"),
		"failed_attempts": c.Int("failedAttempts"),
		"failures":        c.Int("failures"),
	}),
	"network": c.Dict("network", s.Schema{
		"bytes":          c.Int("bytes"),
		"getmores":       c.Dict("getmores", countAndTimeSchema),
		"ops":            c.Int("ops"),
		"reders_created": c.Int("readersCreated"),
	}),
	"preload": c.Dict("preload", s.Schema{
		"docs":    c.Dict("docs", countAndTimeSchema),
		"indexes": c.Dict("indexes", countAndTimeSchema),
	}),
}

var countAndTimeSchema = s.Schema{
	"count": c.Int("num"),
	"time":  s.Object{"ms": c.Int("totalMillis")},
}
