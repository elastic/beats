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
	"github.com/elastic/beats/metricbeat/module/elasticsearch"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	clusterStatsSchema = s.Schema{
		"cluster_uuid": c.Str("cluster_uuid"),
		"timestamp":    c.Int("timestamp"),
		"status":       c.Str("status"),
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
				"evictions":            c.Int("evictions"),
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
			"os": c.Dict("os", s.Schema{
				"available_processors": c.Int("available_processors"),
				"allocated_processors": c.Int("allocated_processors"),
				"mem": c.Dict("mem", s.Schema{
					"total_in_bytes": c.Int("total_in_bytes"),
					"free_in_bytes":  c.Int("free_in_bytes"),
					"used_in_bytes":  c.Int("used_in_bytes"),
					"free_percent":   c.Int("free_percent"),
					"used_percent":   c.Int("used_percent"),
				}),
			}),
			"process": c.Dict("process", s.Schema{
				"cpu": c.Dict("cpu", s.Schema{
					"percent": c.Int("percent"),
				}),
				"open_file_descriptors": c.Dict("open_file_descriptors", s.Schema{
					"min": c.Int("min"),
					"max": c.Int("max"),
					"avg": c.Int("avg"),
				}),
			}),
			"jvm": c.Dict("jvm", s.Schema{
				"max_uptime_in_millis": c.Int("max_uptime_in_millis"),
				"mem": c.Dict("mem", s.Schema{
					"heap_used_in_bytes": c.Int("heap_used_in_bytes"),
					"heap_max_in_bytes":  c.Int("heap_max_in_bytes"),
				}),
				"threads": c.Int("threads"),
			}),
			"fs": c.Dict("fs", s.Schema{
				"total_in_bytes":     c.Int("total_in_bytes"),
				"free_in_bytes":      c.Int("free_in_bytes"),
				"available_in_bytes": c.Int("available_in_bytes"),
			}),
		}),
	}
)

func passthruField(fieldPath string, sourceData, targetData common.MapStr) error {
	fieldValue, err := sourceData.GetValue(fieldPath)
	if err != nil {
		return elastic.MakeErrorForMissingField(fieldPath, elastic.Elasticsearch)
	}

	targetData.Put(fieldPath, fieldValue)
	return nil
}

func eventMappingXPack(r mb.ReporterV2, m *MetricSet, content []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return err
	}

	clusterStats, err := clusterStatsSchema.Apply(data)
	if err != nil {
		return err
	}

	dataMS := common.MapStr(data)

	passthruFields := []string{
		"indices.segments.file_sizes",
		"nodes.versions",
		"nodes.os.names",
		"nodes.jvm.versions",
		"nodes.plugins",
		"nodes.network_types",
	}
	for _, fieldPath := range passthruFields {
		if err = passthruField(fieldPath, dataMS, clusterStats); err != nil {
			return err
		}
	}

	clusterName, ok := data["cluster_name"].(string)
	if !ok {
		return elastic.MakeErrorForMissingField("cluster_name", elastic.Elasticsearch)
	}

	info, err := elasticsearch.GetInfo(m.HTTP, m.HTTP.GetURI())
	if err != nil {
		return err
	}

	license, err := elasticsearch.GetLicense(m.HTTP, m.HTTP.GetURI())
	if err != nil {
		return err
	}

	// TODO: Inject `cluster_needs_tls` field under license object

	clusterState, err := elasticsearch.GetClusterState(m.HTTP, m.HTTP.GetURI())
	if err != nil {
		return err
	}

	if err = passthruField("status", dataMS, clusterState); err != nil {
		return err
	}

	// TODO: Compute and inject `node_hash` field under clusterState object

	stackStats, err := elasticsearch.GetStackStats(m.HTTP, m.HTTP.GetURI())
	if err != nil {
		return err
	}

	// TODO: Inject `apm.found` field under stackStats object

	event := mb.Event{}
	event.RootFields = common.MapStr{
		"cluster_uuid":  info.ClusterID,
		"cluster_name":  clusterName,
		"timestamp":     common.Time(time.Now()),
		"interval_ms":   m.Module().Config().Period / time.Millisecond,
		"type":          "cluster_stats",
		"license":       license,
		"version":       info.Version.Number,
		"cluster_stats": clusterStats,
		"cluster_state": clusterState,
		"stack_stats":   stackStats,
	}

	event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Elasticsearch)
	r.Event(event)

	return nil
}
