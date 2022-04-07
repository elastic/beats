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
	s "github.com/elastic/beats/v8/libbeat/common/schema"
	c "github.com/elastic/beats/v8/libbeat/common/schema/mapstriface"
)

var schema = s.Schema{
	"version": c.Str("version"),
	"process": c.Str("process"),
	"uptime": s.Object{
		"ms": c.Int("uptimeMillis"),
	},
	"local_time": c.Time("localTime"),
	"asserts": c.Dict("asserts", s.Schema{
		"regular":   c.Int("regular"),
		"warning":   c.Int("warning"),
		"msg":       c.Int("msg"),
		"user":      c.Int("user"),
		"rollovers": c.Int("rollovers"),
	}),
	"connections": c.Dict("connections", s.Schema{
		"current":       c.Int("current"),
		"available":     c.Int("available"),
		"total_created": c.Int("totalCreated"),
	}),
	"extra_info": c.Dict("extra_info", s.Schema{
		"heap_usage":  s.Object{"bytes": c.Int("heap_usage_bytes", s.Optional)},
		"page_faults": c.Int("page_faults"),
	}),
	"global_lock": c.Dict("globalLock", s.Schema{
		"total_time":     s.Object{"us": c.Int("totalTime")},
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
		"in":       s.Object{"bytes": c.Int("bytesIn")},
		"out":      s.Object{"bytes": c.Int("bytesOut")},
		"requests": c.Int("numRequests"),
	}),
	"ops": s.Object{
		"latencies": c.Dict("opLatencies", s.Schema{
			"reads":    c.Dict("reads", opLatenciesItemSchema),
			"writes":   c.Dict("writes", opLatenciesItemSchema),
			"commands": c.Dict("commands", opLatenciesItemSchema),
		}, c.DictOptional),
		"counters": c.Dict("opcounters", s.Schema{
			"insert":  c.Int("insert"),
			"query":   c.Int("query"),
			"update":  c.Int("update"),
			"delete":  c.Int("delete"),
			"getmore": c.Int("getmore"),
			"command": c.Int("command"),
		}),
		"replicated": c.Dict("opcountersRepl", s.Schema{
			"insert":  c.Int("insert"),
			"query":   c.Int("query"),
			"update":  c.Int("update"),
			"delete":  c.Int("delete"),
			"getmore": c.Int("getmore"),
			"command": c.Int("command"),
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
		"bits":                c.Int("bits"),
		"resident":            s.Object{"mb": c.Int("resident")},
		"virtual":             s.Object{"mb": c.Int("virtual")},
		"mapped":              s.Object{"mb": c.Int("mapped")},
		"mapped_with_journal": s.Object{"mb": c.Int("mappedWithJournal")},
	}),

	// MMPAV1 only
	"background_flushing": c.Dict("backgroundFlushing", s.Schema{
		"flushes": c.Int("flushes"),
		"total": s.Object{
			"ms": c.Int("total_ms"),
		},
		"average": s.Object{
			"ms": c.Int("average_ms"),
		},
		"last": s.Object{
			"ms": c.Int("last_ms"),
		},
		"last_finished": c.Time("last_finished"),
	}, c.DictOptional),

	// MMPAV1 only
	"journaling": c.Dict("dur", s.Schema{
		"commits": c.Int("commits"),
		"journaled": s.Object{
			"mb": c.Int("journaledMB"),
		},
		"write_to_data_files": s.Object{
			"mb": c.Int("writeToDataFilesMB"),
		},
		"compression":           c.Int("compression"),
		"commits_in_write_lock": c.Int("commitsInWriteLock"),
		"early_commits":         c.Int("earlyCommits"),
		"times": c.Dict("timeMs", s.Schema{
			"dt":                    s.Object{"ms": c.Int("dt")},
			"prep_log_buffer":       s.Object{"ms": c.Int("prepLogBuffer")},
			"write_to_journal":      s.Object{"ms": c.Int("writeToJournal")},
			"write_to_data_files":   s.Object{"ms": c.Int("writeToDataFiles")},
			"remap_private_view":    s.Object{"ms": c.Int("remapPrivateView")},
			"commits":               s.Object{"ms": c.Int("commits")},
			"commits_in_write_lock": s.Object{"ms": c.Int("commitsInWriteLock")},
		}),
	}, c.DictOptional),
}

var wiredTigerSchema = s.Schema{
	"concurrent_transactions": c.Dict("concurrentTransactions", s.Schema{
		"write": c.Dict("write", s.Schema{
			"out":           c.Int("out"),
			"available":     c.Int("available"),
			"total_tickets": c.Int("totalTickets"),
		}),
		"read": c.Dict("write", s.Schema{
			"out":           c.Int("out"),
			"available":     c.Int("available"),
			"total_tickets": c.Int("totalTickets"),
		}),
	}),
	"cache": c.Dict("cache", s.Schema{
		"maximum": s.Object{"bytes": c.Int("maximum bytes configured")},
		"used":    s.Object{"bytes": c.Int("bytes currently in the cache")},
		"dirty":   s.Object{"bytes": c.Int("tracked dirty bytes in the cache")},
		"pages": s.Object{
			"read":    c.Int("pages read into cache"),
			"write":   c.Int("pages written from cache"),
			"evicted": c.Int("unmodified pages evicted"),
		},
	}),
	"log": c.Dict("log", s.Schema{
		"size":          s.Object{"bytes": c.Int("total log buffer size")},
		"write":         s.Object{"bytes": c.Int("log bytes written")},
		"max_file_size": s.Object{"bytes": c.Int("maximum log file size")},
		"flushes":       c.Int("log flush operations"),
		"writes":        c.Int("log write operations"),
		"scans":         c.Int("log scan operations"),
		"syncs":         c.Int("log sync operations"),
	}),
}

var globalLockItemSchema = s.Schema{
	"total":   c.Int("total"),
	"readers": c.Int("readers"),
	"writers": c.Int("writers"),
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
	"r": c.Int("r", s.Optional),
	"w": c.Int("w", s.Optional),
	"R": c.Int("R", s.Optional),
	"W": c.Int("W", s.Optional),
}

var opLatenciesItemSchema = s.Schema{
	"latency": c.Int("latency"),
	"count":   c.Int("ops"),
}
