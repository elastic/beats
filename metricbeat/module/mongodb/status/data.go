package status

import (
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstriface"
)

var schema = s.Schema{
	"version": c.Str("version"),
	"uptime": s.Object{
		"ms": c.Int("uptimeMillis"),
	},
	"local_time":         c.Time("localTime"),
	"write_backs_queued": c.Bool("writeBacksQueued"),
	"asserts": c.Dict("asserts", s.Schema{
		"regular":   c.Int("regular"),
		"warning":   c.Int("warning"),
		"msg":       c.Int("msg"),
		"user":      c.Int("user"),
		"rollovers": c.Int("rollovers"),
	}),
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
	}),
	"connections": c.Dict("connections", s.Schema{
		"current":       c.Int("current"),
		"available":     c.Int("available"),
		"total_created": c.Int("totalCreated"),
	}),
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
	}),
	"extra_info": c.Dict("extra_info", s.Schema{
		"heap_usage":  s.Object{"bytes": c.Int("heap_usage_bytes")},
		"page_faults": c.Int("page_faults"),
	}),
	"network": c.Dict("network", s.Schema{
		"in":       s.Object{"bytes": c.Int("bytesIn")},
		"out":      s.Object{"bytes": c.Int("bytesOut")},
		"requests": c.Int("numRequests"),
	}),
	"memory": c.Dict("mem", s.Schema{
		"bits":                c.Int("bits"),
		"resident":            s.Object{"mb": c.Int("resident")},
		"virtual":             s.Object{"mb": c.Int("virtual")},
		"mapped":              s.Object{"mb": c.Int("mapped")},
		"mapped_with_journal": s.Object{"mb": c.Int("mappedWithJournal")},
	}),
	"opcounters": c.Dict("opcounters", s.Schema{
		"insert":  c.Int("insert"),
		"query":   c.Int("query"),
		"update":  c.Int("update"),
		"delete":  c.Int("delete"),
		"getmore": c.Int("getmore"),
		"command": c.Int("command"),
	}),
	"opcounters_replicated": c.Dict("opcountersRepl", s.Schema{
		"insert":  c.Int("insert"),
		"query":   c.Int("query"),
		"update":  c.Int("update"),
		"delete":  c.Int("delete"),
		"getmore": c.Int("getmore"),
		"command": c.Int("command"),
	}),
	"storage_engine": c.Dict("storageEngine", s.Schema{
		"name": c.Str("name"),
	}),
}

var eventMapping = schema.Apply
