package status

import "github.com/elastic/beats/libbeat/common"

var schema = NewMSchema(common.MapStr{
	"version": String("version"),
	"uptime": common.MapStr{
		"ms": Int("uptimeMillis"),
	},
	"local_time":         Time("localTime"),
	"write_backs_queued": Bool("writeBacksQueued"),
	"asserts": Map("asserts", common.MapStr{
		"regular":   Int("regular"),
		"warning":   Int("warning"),
		"msg":       Int("msg"),
		"user":      Int("user"),
		"rollovers": Int("rollovers"),
	}),
	"background_flushing": Map("backgroundFlushing", common.MapStr{
		"flushes": Int("flushes"),
		"total": common.MapStr{
			"ms": Int("total_ms"),
		},
		"average": common.MapStr{
			"ms": Int("average_ms"),
		},
		"last": common.MapStr{
			"ms": Int("last_ms"),
		},
		"last_finished": Time("last_finished"),
	}),
	"connections": Map("connections", common.MapStr{
		"current":       Int("current"),
		"available":     Int("available"),
		"total_created": Int("totalCreated"),
	}),
	"journaling": Map("dur", common.MapStr{
		"commits": Int("commits"),
		"journaled": common.MapStr{
			"mb": Int("journaledMB"),
		},
		"write_to_data_files": common.MapStr{
			"mb": Int("writeToDataFilesMB"),
		},
		"compression":           Int("compression"),
		"commits_in_write_lock": Int("commitsInWriteLock"),
		"early_commits":         Int("earlyCommits"),
		"times": Map("timeMs", common.MapStr{
			"dt":                    common.MapStr{"ms": Int("dt")},
			"prep_log_buffer":       common.MapStr{"ms": Int("prepLogBuffer")},
			"write_to_journal":      common.MapStr{"ms": Int("writeToJournal")},
			"write_to_data_files":   common.MapStr{"ms": Int("writeToDataFiles")},
			"remap_private_view":    common.MapStr{"ms": Int("remapPrivateView")},
			"commits":               common.MapStr{"ms": Int("commits")},
			"commits_in_write_lock": common.MapStr{"ms": Int("commitsInWriteLock")},
		}),
	}),
	"extra_info": Map("extra_info", common.MapStr{
		"heap_usage":  common.MapStr{"bytes": Int("heap_usage_bytes")},
		"page_faults": Int("page_faults"),
	}),
	"network": Map("network", common.MapStr{
		"in":       common.MapStr{"bytes": Int("bytesIn")},
		"out":      common.MapStr{"bytes": Int("bytesOut")},
		"requests": Int("numRequests"),
	}),
	"memory": Map("mem", common.MapStr{
		"bits":                Int("bits"),
		"resident":            common.MapStr{"mb": Int("resident")},
		"virtual":             common.MapStr{"mb": Int("virtual")},
		"mapped":              common.MapStr{"mb": Int("mapped")},
		"mapped_with_journal": common.MapStr{"mb": Int("mappedWithJournal")},
	}),
	"opcounters": Map("opcounters", common.MapStr{
		"insert":  Int("insert"),
		"query":   Int("query"),
		"update":  Int("update"),
		"delete":  Int("delete"),
		"getmore": Int("getmore"),
		"command": Int("command"),
	}),
	"opcounters_replicated": Map("opcountersRepl", common.MapStr{
		"insert":  Int("insert"),
		"query":   Int("query"),
		"update":  Int("update"),
		"delete":  Int("delete"),
		"getmore": Int("getmore"),
		"command": Int("command"),
	}),
	"storage_engine": Map("storageEngine", common.MapStr{
		"name": String("name"),
	}),
})

func eventMapping(status map[string]interface{}) common.MapStr {
	return schema.Apply(status)
}
