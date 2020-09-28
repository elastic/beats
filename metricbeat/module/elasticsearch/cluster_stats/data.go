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

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

var (
	schema = s.Schema{
		"status": c.Str("status"),
		"nodes": c.Dict("nodes", s.Schema{
			"total": c.Int("count.total"),
			"master": s.Object{
				"total": c.Int("count.master"),
			},
			"data": s.Object{
				"total": c.Int("count.data"),
			},
			"coordinating_only": s.Object{
				"total": c.Int("count.coordinating_only"),
			},
			"ml": s.Object{
				"total": c.Int("count.ml"),
			},
			"remote_cluster_client": s.Object{
				"total": c.Int("count.remote_cluster_client"),
			},
			"transform": s.Object{
				"total": c.Int("count.transform"),
			},
			"voting_only": s.Object{
				"total": c.Int("count.voting_only"),
			},
			"ingest": s.Object{
				"total": c.Int("count.ingest"),
				"pipelines": s.Object{
					"total": c.Int("ingest.number_of_pipelines"),
				},
				//TODO Not sure if to remove this one
				"processor_stats": c.Ifc("ingest.processor_stats"),
			},
			"jvm": c.Dict("jvm", s.Schema{
				"threads": s.Object{
					"total": c.Int("threads"),
				},
				"max_uptime": s.Object{
					"ms": c.Int("max_uptime_in_millis"),
				},
				"memory": c.Dict("mem", s.Schema{
					"heap": s.Object{
						"used": s.Object{
							"bytes": c.Int("heap_used_in_bytes"),
						},
						"max": s.Object{
							"bytes": c.Int("heap_max_in_bytes"),
						},
					},
				}),
			}),
			"fs": c.Dict("fs", s.Schema{
				"available": s.Object{
					"bytes": c.Int("available_in_bytes"),
				},
				"total": s.Object{
					"bytes": c.Int("total_in_bytes"),
				},
				"free": s.Object{
					"bytes": c.Int("free_in_bytes"),
				},
			}),
		}),
		"indices": c.Dict("indices", s.Schema{
			"docs": c.Dict("docs", s.Schema{
				"total": c.Int("count"),
				"deleted": s.Object{
					"total": c.Int("deleted"),
				},
			}),
			"total": c.Int("count"),
			"shards": c.Dict("shards", s.Schema{
				"total":       c.Int("total"),
				"primaries":   c.Int("primaries"),
				"replication": c.Int("replication"),
				"index":       c.Ifc("index"),
			}),
			"store": c.Dict("store", s.Schema{
				"size": s.Object{
					"bytes": c.Int("size_in_bytes"),
				},
				"reserved": s.Object{
					"bytes": c.Int("reserved_in_bytes"),
				},
			}),
			"query_cache": c.Dict("query_cache", s.Schema{
				"total": c.Int("total_count"),
				"hit": s.Object{
					"total": c.Ifc("hit_count"),
				},
				"miss": s.Object{
					"total": c.Ifc("miss_count"),
				},
				"cache": s.Object{
					"total": c.Ifc("cache_count"),
				},
				"evictions": c.Int("evictions"),
				"cache_size": s.Object{
					"bytes": c.Int("cache_size"),
				},
				"memory_size": s.Object{
					"bytes": c.Int("memory_size_in_bytes"),
				},
			}),
			"segments": c.Dict("segments", s.Schema{
				"total": c.Int("count"),
				"memory": s.Object{
					"stored_fields": s.Object{
						"bytes": c.Int("stored_fields_memory_in_bytes"),
					},
					"points": s.Object{
						"bytes": c.Int("points_memory_in_bytes"),
					},
					"doc_values": s.Object{
						"bytes": c.Int("doc_values_memory_in_bytes"),
					},
					"index_writer": s.Object{
						"bytes": c.Int("index_writer_memory_in_bytes"),
					},
					"fixed_bit_set": s.Object{
						"bytes": c.Int("fixed_bit_set_memory_in_bytes"),
					},
					"norms": s.Object{
						"bytes": c.Int("norms_memory_in_bytes"),
					},
					"version_map": s.Object{
						"bytes": c.Int("version_map_memory_in_bytes"),
					},
					"bytes": c.Int("memory_in_bytes"),
					"terms": s.Object{
						"bytes": c.Int("terms_memory_in_bytes"),
						"vectors": s.Object{
							"bytes": c.Int("term_vectors_memory_in_bytes"),
						},
					},
					//"file_sizes": Unknown format
				},
				"max_unsafe_auto_id": s.Object{
					"ms": c.Int("max_unsafe_auto_id_timestamp"),
				},
			}),
			"fielddata": c.Dict("fielddata", s.Schema{
				"memory": s.Object{
					"bytes": c.Int("memory_size_in_bytes"),
				},
			}),
		}),
	}

	stackSchema = s.Schema{
		"apm": c.Ifc("apm"),
		"xpack": c.Dict("xpack", s.Schema{
			"rollup":               c.Ifc("rollup"),
			"logstash":             c.Ifc("logstash"),
			"transform":            c.Ifc("transform"),
			"security":             c.Ifc("security"),
			"data_streams":         c.Ifc("data_streams"),
			"monitoring":           c.Ifc("monitoring"),
			"graph":                c.Ifc("graph"),
			"voting_only":          c.Ifc("voting_only"),
			"slm":                  c.Ifc("slm"),
			"frozen_indices":       c.Ifc("frozen_indices"),
			"spatial":              c.Ifc("spatial"),
			"searchable_snapshots": c.Ifc("searchable_snapshots"),
			"ccr":                  c.Ifc("ccr"),
			"vectors":              c.Ifc("vectors"),
			"ilm": c.Dict("ilm", s.Schema{
				"policy": s.Object{
					"total": c.Int("policy_count"),
					"stats": c.Ifc("policy_stats"),
				},
			}),
			//"watcher": c.Ifc("watcher"),
			"ml": c.Dict("ml", s.Schema{
				"node": s.Object{
					"total": c.Int("node_count"),
				},
				"available": c.Bool("available"),
				"enabled":   c.Bool("enabled"),
				"jobs": c.Dict("jobs", s.Schema{
					"total": c.Int("_all.count"),
				}),
			}),
		}),
	}
)

func _eventMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte) error {
	var event mb.Event
	//event.RootFields = common.MapStr{}
	//event.RootFields.Put("service.name", elasticsearch.ModuleName)

	event.ModuleFields = common.MapStr{}
	//event.ModuleFields.Put("cluster.name", info.ClusterName)
	//event.ModuleFields.Put("cluster.id", info.ClusterID)

	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Elasticsearch Cluster Stats API response")
	}

	metricSetFields, err := schema.Apply(data)
	if err != nil {
		return errors.Wrap(err, "failure applying cluster stats schema")
	}

	event.MetricSetFields = metricSetFields

	r.Event(event)
	return nil
}
