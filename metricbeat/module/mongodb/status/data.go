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

package status

import (
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
)

var schema = s.Schema{
	"version": c.Str("version"),
	"process": c.Str("process"),
	"uptime": s.Object{
		"ms": c.Ifc("uptimeMillis"),
	},
	"local_time": c.Time("localTime"),
	"asserts": c.Dict("asserts", s.Schema{
		"regular":   c.Ifc("regular"),
		"warning":   c.Ifc("warning"),
		"msg":       c.Ifc("msg"),
		"user":      c.Ifc("user"),
		"rollovers": c.Ifc("rollovers"),
	}),
	"connections": c.Dict("connections", s.Schema{
		"current":       c.Ifc("current"),
		"available":     c.Ifc("available"),
		"total_created": c.Ifc("totalCreated"),
	}),
	"extra_info": c.Dict("extra_info", s.Schema{
		"heap_usage":  s.Object{"bytes": c.Ifc("heap_usage_bytes", s.Optional)},
		"page_faults": c.Ifc("page_faults"),
	}),
	"global_lock": c.Dict("globalLock", s.Schema{
		"total_time":     s.Object{"us": c.Ifc("totalTime")},
		"current_queue":  c.Dict("currentQueue", globalLockItemSchema),
		"active_clients": c.Dict("activeClients", globalLockItemSchema),
	}),
	"locks": c.Dict("locks", s.Schema{
		"global":     c.Dict("Global", lockItemSchema),
		"database":   c.Dict("Database", lockItemSchema),
		"collection": c.Dict("Collection", lockItemSchema),
		"meta_data":  c.Dict("Metadata", lockItemSchema),
		"oplog":      c.Dict("oplog", lockItemSchema),
	}),
	"network": c.Dict("network", s.Schema{
		"in":       s.Object{"bytes": c.Ifc("bytesIn")},
		"out":      s.Object{"bytes": c.Ifc("bytesOut")},
		"requests": c.Ifc("numRequests"),
	}),
	"ops": s.Object{
		"latencies": c.Dict("opLatencies", s.Schema{
			"reads":    c.Dict("reads", opLatenciesItemSchema),
			"writes":   c.Dict("writes", opLatenciesItemSchema),
			"commands": c.Dict("commands", opLatenciesItemSchema),
		}, c.DictOptional),
		"counters": c.Dict("opcounters", s.Schema{
			"insert":  c.Ifc("insert"),
			"query":   c.Ifc("query"),
			"update":  c.Ifc("update"),
			"delete":  c.Ifc("delete"),
			"getmore": c.Ifc("getmore"),
			"command": c.Ifc("command"),
		}),
		"replicated": c.Dict("opcountersRepl", s.Schema{
			"insert":  c.Ifc("insert"),
			"query":   c.Ifc("query"),
			"update":  c.Ifc("update"),
			"delete":  c.Ifc("delete"),
			"getmore": c.Ifc("getmore"),
			"command": c.Ifc("command"),
		}),
	},
	// ToDo add `repl` field
	"storage_engine": c.Dict("storageEngine", s.Schema{
		"name": c.Str("name"),
		// supportsCommitedReads boolean
		// readOnly boolean
		// persistent boolean
	}),
	// ToDo add `tcmalloc` field
	"wired_tiger":        c.Dict("wiredTiger", wiredTigerSchema, c.DictOptional),
	"write_backs_queued": c.Bool("writeBacksQueued", s.Optional),
	"memory": c.Dict("mem", s.Schema{
		"bits":                c.Ifc("bits"),
		"resident":            s.Object{"mb": c.Ifc("resident")},
		"virtual":             s.Object{"mb": c.Ifc("virtual")},
		"mapped":              s.Object{"mb": c.Ifc("mapped")},
		"mapped_with_journal": s.Object{"mb": c.Ifc("mappedWithJournal")},
	}),

	// MMPAV1 only
	"background_flushing": c.Dict("backgroundFlushing", s.Schema{
		"flushes": c.Ifc("flushes"),
		"total": s.Object{
			"ms": c.Ifc("total_ms"),
		},
		"average": s.Object{
			"ms": c.Ifc("average_ms"),
		},
		"last": s.Object{
			"ms": c.Ifc("last_ms"),
		},
		"last_finished": c.Time("last_finished"),
	}, c.DictOptional),

	// MMPAV1 only
	"journaling": c.Dict("dur", s.Schema{
		"commits": c.Ifc("commits"),
		"journaled": s.Object{
			"mb": c.Ifc("journaledMB"),
		},
		"write_to_data_files": s.Object{
			"mb": c.Ifc("writeToDataFilesMB"),
		},
		"compression":           c.Ifc("compression"),
		"commits_in_write_lock": c.Ifc("commitsInWriteLock"),
		"early_commits":         c.Ifc("earlyCommits"),
		"times": c.Dict("timeMs", s.Schema{
			"dt":                    s.Object{"ms": c.Ifc("dt")},
			"prep_log_buffer":       s.Object{"ms": c.Ifc("prepLogBuffer")},
			"write_to_journal":      s.Object{"ms": c.Ifc("writeToJournal")},
			"write_to_data_files":   s.Object{"ms": c.Ifc("writeToDataFiles")},
			"remap_private_view":    s.Object{"ms": c.Ifc("remapPrivateView")},
			"commits":               s.Object{"ms": c.Ifc("commits")},
			"commits_in_write_lock": s.Object{"ms": c.Ifc("commitsInWriteLock")},
		}),
	}, c.DictOptional),
}

var wiredTigerSchema = s.Schema{
	"concurrent_transactions": c.Dict("concurrentTransactions", s.Schema{
		"write": c.Dict("write", s.Schema{
			"out":           c.Ifc("out"),
			"available":     c.Ifc("available"),
			"total_tickets": c.Ifc("totalTickets"),
		}),
		"read": c.Dict("write", s.Schema{
			"out":           c.Ifc("out"),
			"available":     c.Ifc("available"),
			"total_tickets": c.Ifc("totalTickets"),
		}),
	}),
	"cache": c.Dict("cache", s.Schema{
		"maximum": s.Object{"bytes": c.Ifc("maximum bytes configured")},
		"used":    s.Object{"bytes": c.Ifc("bytes currently in the cache")},
		"dirty":   s.Object{"bytes": c.Ifc("tracked dirty bytes in the cache")},
		"pages": s.Object{
			"read":    c.Ifc("pages read into cache"),
			"write":   c.Ifc("pages written from cache"),
			"evicted": c.Ifc("unmodified pages evicted"),
		},
	}),
	"log": c.Dict("log", s.Schema{
		"size":          s.Object{"bytes": c.Ifc("total log buffer size")},
		"write":         s.Object{"bytes": c.Ifc("log bytes written")},
		"max_file_size": s.Object{"bytes": c.Ifc("maximum log file size")},
		"flushes":       c.Ifc("log flush operations"),
		"writes":        c.Ifc("log write operations"),
		"scans":         c.Ifc("log scan operations"),
		"syncs":         c.Ifc("log sync operations"),
	}),
}

var globalLockItemSchema = s.Schema{
	"total":   c.Ifc("total"),
	"readers": c.Ifc("readers"),
	"writers": c.Ifc("writers"),
}

var lockItemSchema = s.Schema{
	"acquire": s.Object{
		"count": c.Dict("acquireCount", lockItemModesSchema, c.DictOptional),
	},
	"wait": s.Object{
		"count": c.Dict("acquireWaitCount", lockItemModesSchema, c.DictOptional),
		"us":    c.Dict("timeAcquiringMicros", lockItemModesSchema, c.DictOptional),
	},
	"deadlock": s.Object{
		"count": c.Dict("deadlockCount", lockItemModesSchema, c.DictOptional),
	},
}

var lockItemModesSchema = s.Schema{
	"r": c.Ifc("r", s.Optional),
	"w": c.Ifc("w", s.Optional),
	"R": c.Ifc("R", s.Optional),
	"W": c.Ifc("W", s.Optional),
}

var opLatenciesItemSchema = s.Schema{
	"latency": c.Ifc("latency"),
	"count":   c.Ifc("ops"),
}
