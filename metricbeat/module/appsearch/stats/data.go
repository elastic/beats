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

package stats

import (
	"encoding/json"

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

	var queues = data["queues"].(map[string]interface{})
	var failed = queues["failed"].([]interface{})
	queues["failed.count"] = len(failed)

	dataFields, err := schema.Apply(data)
	return dataFields, err
}
