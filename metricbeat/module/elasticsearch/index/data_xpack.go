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

package index

import (
	"encoding/json"
	"time"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

// TODO:
// "index_stats.created",
// "index_stats.status",
// "index_stats.version.created",
// "index_stats.version.upgraded",
// "index_stats.shards.total",
// "index_stats.shards.primaries",
// "index_stats.shards.replicas",
// "index_stats.shards.active_total",
// "index_stats.shards.active_primaries",
// "index_stats.shards.active_replicas",
// "index_stats.shards.unassigned_total",
// "index_stats.shards.unassigned_primaries",
// "index_stats.shards.unassigned_replicas",
// "index_stats.shards.initializing",
// "index_stats.shards.relocating",

var (
	xpackSchema = s.Schema{
		"uuid":      c.Str("uuid"),
		"primaries": c.Dict("primaries", indexStatsSchema),
		"total":     c.Dict("total", indexStatsSchema),
	}

	indexStatsSchema = s.Schema{
		"docs": c.Dict("docs", s.Schema{
			"count": c.Int("count"),
		}),
		"fielddata": c.Dict("fielddata", s.Schema{
			"memory_size_in_bytes": c.Int("memory_size_in_bytes"),
			"evictions":            c.Int("evictions"),
		}),
		"indexing": c.Dict("indexing", s.Schema{
			"index_total":             c.Int("index_total"),
			"index_time_in_millis":    c.Int("index_time_in_millis"),
			"throttle_time_in_millis": c.Int("throttle_time_in_millis"),
		}),
		"merges": c.Dict("merges", s.Schema{
			"total_size_in_bytes": c.Int("total_size_in_bytes"),
		}),
		"query_cache":   c.Dict("query_cache", cacheStatsSchema),
		"request_cache": c.Dict("request_cache", cacheStatsSchema),
		"search": c.Dict("search", s.Schema{
			"query_total":          c.Int("query_total"),
			"query_time_in_millis": c.Int("query_time_in_millis"),
		}),
		"segments": c.Dict("segments", s.Schema{
			"count":                         c.Int("count"),
			"memory_in_bytes":               c.Int("memory_in_bytes"),
			"terms_memory_in_bytes":         c.Int("terms_memory_in_bytes"),
			"stored_fields_memory_in_bytes": c.Int("stored_fields_memory_in_bytes"),
			"term_vectors_memory_in_bytes":  c.Int("term_vectors_memory_in_bytes"),
			"norms_memory_in_bytes":         c.Int("norms_memory_in_bytes"),
			"points_memory_in_bytes":        c.Int("points_memory_in_bytes"),
			"doc_values_memory_in_bytes":    c.Int("doc_values_memory_in_bytes"),
			"index_writer_memory_in_bytes":  c.Int("index_writer_memory_in_bytes"),
			"version_map_memory_in_bytes":   c.Int("version_map_memory_in_bytes"),
			"fixed_bit_set_memory_in_bytes": c.Int("fixed_bit_set_memory_in_bytes"),
		}),
		"store": c.Dict("store", s.Schema{
			"size_in_bytes": c.Int("size_in_bytes"),
		}),
		"refresh": c.Dict("refresh", s.Schema{
			"total_time_in_millis": c.Int("total_time_in_millis"),
		}),
	}

	cacheStatsSchema = s.Schema{
		"memory_size_in_bytes": c.Int("memory_size_in_bytes"),
		"evictions":            c.Int("evictions"),
		"hit_count":            c.Int("hit_count"),
		"miss_count":           c.Int("miss_count"),
	}
)

func eventsMappingXPack(r mb.ReporterV2, m *MetricSet, info elasticsearch.Info, content []byte) error {
	err := json.Unmarshal(content, &indicesStruct)
	if err != nil {
		return err
	}

	for name, index := range indicesStruct.Indices {
		event := mb.Event{}
		indexStats, err := xpackSchema.Apply(index)
		if err != nil {
			continue
		}
		indexStats["index"] = name

		event.RootFields = common.MapStr{
			"cluster_uuid": info.ClusterID,
			"timestamp":    common.Time(time.Now()),
			"interval_ms":  m.Module().Config().Period / time.Millisecond,
			"type":         "index_stats",
			"index_stats":  indexStats,
		}

		event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
		r.Event(event)
	}

	return nil
}
