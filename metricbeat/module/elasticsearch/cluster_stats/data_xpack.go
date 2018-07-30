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

package cluster_stats

import (
	"encoding/json"
	"time"

	"github.com/elastic/beats/metricbeat/helper/elastic"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

var (
	clusterStatsSchema = s.Schema{
		"status": c.Str("status"),
		"indices": c.Dict("indices", s.Schema{
			"count": c.Int("count"),
			"shards": c.Dict("shards", s.Schema{
				"total":       c.Int("total"),
				"primaries":   c.Int("primaries"),
				"replication": c.Int("replication"),
				"index": c.Dict("index", s.Schema{
					"shards": c.Dict("shards", s.Schema{
						"min": c.Int("min"),
						"max": c.Int("max"),
						"avg": c.Int("avg"),
					}),
					"primaries": c.Dict("primaries", s.Schema{
						"min": c.Int("min"),
						"max": c.Int("max"),
						"avg": c.Int("avg"),
					}),
					"replication": c.Dict("replication", s.Schema{
						"min": c.Int("min"),
						"max": c.Int("max"),
						"avg": c.Int("avg"),
					}),
				}),
			}),
			"docs": c.Dict("docs", s.Schema{
				"count":   c.Int("count"),
				"deleted": c.Int("deleted"),
			}),
			"store": c.Dict("store", s.Schema{
				"size_in_bytes": c.Int("size_in_bytes"),
			}),
			"fielddata": c.Dict("fielddata", s.Schema{
				"memory_size_in_bytes": c.Int("memory_size_in_bytes"),
				"evictions":            c.Int("evictions"),
			}),
			"query_cache": c.Dict("query_cache", s.Schema{
				"memory_size_in_bytes": c.Int("memory_size_in_bytes"),
				"total_count":          c.Int("total_count"),
				"hit_count":            c.Int("hit_count"),
				"miss_count":           c.Int("miss_count"),
				"cache_size":           c.Int("cache_size"),
				"cache_count":          c.Int("cache_count"),
			}),
			"completion": c.Dict("completion", s.Schema{
				"size_in_bytes": c.Int("size_in_bytes"),
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
				"max_unsafe_auto_id_timestamp":  c.Int("max_unsafe_auto_id_timestamp"),
			}),
		}),
		"nodes": c.Dict("nodes", s.Schema{
			"count": c.Dict("count", s.Schema{
				"total":             c.Int("total"),
				"data":              c.Int("data"),
				"coordinating_only": c.Int("coordinating_only"),
				"master":            c.Int("master"),
				"ingest":            c.Int("ingest"),
			}),
			// TODO: os
			// TODO: process
			// TODO: jvm
			// TODO: fs
			// TODO: network_types
		}),
	}
)

func eventMappingXPack(r mb.ReporterV2, m *MetricSet, info elasticsearch.Info, content []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		r.Error(err)
		return err
	}

	clusterStatsFields, err := clusterStatsSchema.Apply(data)
	if err != nil {
		r.Error(err)
		return err
	}

	// TODO: handle cluster stats fields:
	// - nodes.versions (array of strings)
	// - nodes.os.names (array of objects)
	// - nodes.jvm.versions (array of objects)
	// - nodes.plugins (array of ??)
	// - nodes.network_types.transport_types (is this an object with dynamic keys?)
	// - nodes.network_types.http_types (is this an object with dynamic keys?)

	clusterUUID, ok := data["cluster_uuid"].(string)
	if !ok {
		return elastic.ReportErrorForMissingField("cluster_uuid", elastic.Elasticsearch, r)
	}

	clusterName, ok := data["cluster_name"].(string)
	if !ok {
		return elastic.ReportErrorForMissingField("cluster_name", elastic.Elasticsearch, r)
	}

	version := "TODO: Fetch from http://localhost:9200/ and parse"
	licenseFields := "TODO: Fetch from http://localhost:9200/_xpack/license and parse"
	clusterStateFields := "TODO: Fetch from http://localhost:9200/_cluster/state (with response filtering) + http://localhost:9200/_cluster/health (for status field) and parse"
	stackStatsFields := "TODO: Fetch from http://localhost:9200/_xpack/usage + apm (from cluster state if apm-* indices exist) + and parse"

	event := mb.Event{}
	event.RootFields = common.MapStr{
		"cluster_uuid":  clusterUUID, // TODO: In New(), error out if ES version < 6.5.0 and xpack.enabled = true
		"cluster_name":  clusterName,
		"timestamp":     common.Time(time.Now()),
		"interval_ms":   m.Module().Config().Period / time.Millisecond,
		"type":          "cluster_stats",
		"license":       licenseFields,
		"version":       version,
		"cluster_stats": clusterStatsFields,
		"cluster_state": clusterStateFields,
		"stack_stats":   stackStatsFields,
	}

	event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
	r.Event(event)

	return nil
}
