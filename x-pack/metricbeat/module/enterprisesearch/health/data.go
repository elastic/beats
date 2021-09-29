// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package health

import (
	"encoding/json"
	"errors"

	"github.com/elastic/beats/v7/libbeat/common"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
)

var (
	schema = s.Schema{
		"cluster_uuid": c.Str("cluster_uuid"), // This is going to be included in 7.16+
		"name":         c.Str("name"),

		"version": c.Dict("version", s.Schema{
			"number":     c.Str("number"),
			"build_hash": c.Str("build_hash"),
		}),

		"jvm": c.Dict("jvm", s.Schema{
			"version": c.Str("version"),

			"gc": c.Dict("gc", s.Schema{
				"collection_count": c.Int("collection_count"),
				"collection_time":  s.Object{"ms": c.Int("collection_time")},
				// TODO: Add separate metrics for old and young generation collectors
			}),

			"memory_usage": c.Dict("memory_usage", s.Schema{
				"heap_init":                         s.Object{"bytes": c.Int("heap_init")},
				"heap_used":                         s.Object{"bytes": c.Int("heap_used")},
				"heap_committed":                    s.Object{"bytes": c.Int("heap_committed")},
				"heap_max":                          s.Object{"bytes": c.Int("heap_max")},
				"non_heap_init":                     s.Object{"bytes": c.Int("non_heap_init")},
				"non_heap_committed":                s.Object{"bytes": c.Int("non_heap_committed")},
				"object_pending_finalization_count": c.Int("object_pending_finalization_count"),
			}),

			"threads": c.Dict("threads", s.Schema{
				"thread_count":               c.Int("thread_count"),
				"peak_thread_count":          c.Int("peak_thread_count"),
				"total_started_thread_count": c.Int("total_started_thread_count"),
				"daemon_thread_count":        c.Int("daemon_thread_count"),
			}),
		}),

		"process": c.Dict("process", s.Schema{
			"pid":    c.Int("pid"),
			"uptime": s.Object{"sec": c.Int("uptime")},

			"filebeat": c.Dict("filebeat", s.Schema{
				"pid":                         c.Int("pid"),
				"restart_count":               c.Int("restart_count"),
				"time_since_last_restart.sec": c.Int("seconds_since_last_restart"),
			}),
		}),

		"crawler": c.Dict("crawler", s.Schema{
			"workers": c.Dict("workers", s.Schema{
				"pool_size": c.Int("pool_size"),
				"active":    c.Int("active"),
				"available": c.Int("available"),
			}),
		}),
	}
)

func eventMapping(input []byte) (common.MapStr, error) {
	var data map[string]interface{}
	err := json.Unmarshal(input, &data)
	if err != nil {
		return nil, err
	}

	system, ok := data["system"].(map[string]interface{})
	if !ok {
		return nil, errors.New("system is not a map")
	}

	jvm, ok := data["jvm"].(map[string]interface{})
	if !ok {
		return nil, errors.New("jvm is not a map")
	}

	// Add version info to the JVM section to help the schema mapper find it
	jvm["version"] = system["java_version"]

	// Collect process info in a form ready for mapping
	process := make(map[string]interface{})
	process["pid"] = jvm["pid"]
	process["uptime"] = jvm["uptime"]
	process["filebeat"] = data["filebeat"]
	data["process"] = process

	dataFields, err := schema.Apply(data)
	return dataFields, err
}
