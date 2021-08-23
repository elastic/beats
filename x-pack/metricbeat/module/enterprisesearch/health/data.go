// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package health

import (
	"encoding/json"

	"github.com/elastic/beats/v7/libbeat/common"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
)

var (
	timerSchema = s.Schema{
		"max.ms": c.Float("max"),
		"avg.ms": c.Float("mean"),
	}
	schema = s.Schema{
		"jvm": c.Dict("jvm", s.Schema{
			"memory_usage": c.Dict("memory_usage", s.Schema{
				"heap_init.bytes":                   c.Int("heap_init"),
				"heap_used.bytes":                   c.Int("heap_used"),
				"heap_committed.bytes":              c.Int("heap_committed"),
				"heap_max.bytes":                    c.Int("heap_max"),
				"non_heap_init.bytes":               c.Int("non_heap_init"),
				"non_heap_committed.bytes":          c.Int("non_heap_committed"),
				"object_pending_finalization_count": c.Int("object_pending_finalization_count"),
			}),
			"threads": c.Dict("threads", s.Schema{
				"thread_count":               c.Int("thread_count"),
				"peak_thread_count":          c.Int("peak_thread_count"),
				"total_started_thread_count": c.Int("total_started_thread_count"),
				"daemon_thread_count":        c.Int("daemon_thread_count"),
			}),
		}),
		"crawler": c.Dict("crawler", s.Schema{
			"workers": c.Dict("workers", s.Schema{
				"pool_size": c.Int("pool_size"),
				"active":    c.Int("active"),
				"available": c.Int("available"),
			}),
		}),
		// TODO: Must not be optional once it is added to a released version of Enterprise Search health endpoint
		"cluster_uuid": c.Str("cluster_uuid", s.Optional),
	}
)

func eventMapping(input []byte) (common.MapStr, error) {
	var data map[string]interface{}
	err := json.Unmarshal(input, &data)
	if err != nil {
		return nil, err
	}

	dataFields, err := schema.Apply(data)
	return dataFields, err
}
