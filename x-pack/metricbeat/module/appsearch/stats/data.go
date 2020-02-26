// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stats

import (
	"encoding/json"
	"errors"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

var (
	timerSchema = s.Schema{
		"max.ms": c.Float("max"),
		"avg.ms": c.Float("mean"),
	}
	schema = s.Schema{
		"jvm": c.Dict("jvm", s.Schema{
			"memory_usage": c.Dict("memory_usage", s.Schema{
				"heap_init.bytes":          c.Int("heap_init"),
				"heap_used.bytes":          c.Int("heap_used"),
				"heap_committed.bytes":     c.Int("heap_committed"),
				"heap_max.bytes":           c.Int("heap_max"),
				"non_heap_init.bytes":      c.Int("non_heap_init"),
				"non_heap_committed.bytes": c.Int("non_heap_committed"),
			}),
		}),
		"queues": c.Dict("queues", s.Schema{
			"analytics_events.count":        c.Int("analytics_events.pending"),
			"document_destroyer.count":      c.Int("document_destroyer.pending"),
			"engine_destroyer.count":        c.Int("engine_destroyer.pending"),
			"index_adder.count":             c.Int("index_adder.pending"),
			"indexed_doc_remover.count":     c.Int("indexed_doc_remover.pending"),
			"mailer.count":                  c.Int("mailer.pending"),
			"refresh_document_counts.count": c.Int("refresh_document_counts.pending"),
			"reindexer.count":               c.Int("reindexer.pending"),
			"schema_updater.count":          c.Int("schema_updater.pending"),
			"failed.count":                  c.Int("failed.count"),
		}),
		"requests": c.Dict(
			"stats.metrics",
			s.Schema{
				"api.response_time": c.Dict("timers.api.request.duration", timerSchema, c.DictOptional),
				"web.response_time": c.Dict("timers.web.request.duration", timerSchema, c.DictOptional),
				"count":             c.Int("counters.all.request", s.Optional),
			},
			c.DictOptional,
		),
	}
)

func eventMapping(input []byte) (common.MapStr, error) {
	var data map[string]interface{}
	err := json.Unmarshal(input, &data)
	if err != nil {
		return nil, err
	}

	queues, ok := data["queues"].(map[string]interface{})
	if !ok {
		return nil, errors.New("queues is not a map")
	}

	failed, ok := queues["failed"].([]interface{})
	if !ok {
		return nil, errors.New("queues.failed is not an array of maps")
	}
	queues["failed.count"] = len(failed)

	dataFields, err := schema.Apply(data)
	return dataFields, err
}
