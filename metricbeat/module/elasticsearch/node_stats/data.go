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

package node_stats

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/helper/elastic"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

var (
	schema = s.Schema{
		"name": c.Str("name"),
		"jvm": c.Dict("jvm", s.Schema{
			"mem": c.Dict("mem", s.Schema{
				"heap": s.Object{
					"max": s.Object{
						"bytes": c.Int("heap_max_in_bytes"),
					},
					"used": s.Object{
						"bytes": c.Int("heap_used_in_bytes"),
						"pct":   c.Int("heap_used_percent"),
					},
				},
			}),
			"gc": c.Dict("gc", s.Schema{
				"collectors": c.Dict("collectors", s.Schema{
					"young": c.Dict("young", collectorSchema),
					"old":   c.Dict("old", collectorSchema),
				}),
			}),
		}),
		"indices": c.Dict("indices", s.Schema{
			"bulk": c.Dict("bulk", s.Schema{
				"avg_size": s.Object{
					"bytes": c.Int("avg_size_in_bytes"),
				},
				"avg_time": s.Object{
					"ms": c.Int("avg_time_in_millis"),
				},
				"total_size": s.Object{
					"bytes": c.Int("total_size_in_bytes"),
				},
				"total_time": s.Object{
					"ms": c.Int("total_time_in_millis"),
				},
				"operations": s.Object{
					"total": s.Object{
						"count": c.Int("total_operations"),
					},
				},
			}, c.DictOptional),
			"docs": c.Dict("docs", s.Schema{
				"count":   c.Int("count"),
				"deleted": c.Int("deleted"),
			}),
			"fielddata": c.Dict("fielddata", s.Schema{
				"memory": s.Object{
					"bytes": c.Int("memory_size_in_bytes"),
				},
			}),
			"indexing": c.Dict("indexing", s.Schema{
				"index_time": s.Object{
					"ms": c.Int("index_time_in_millis"),
				},
				"index_total": s.Object{
					"count": c.Int("index_total"),
				},
				"throttle_time": s.Object{
					"ms": c.Int("throttle_time_in_millis"),
				},
			}),
			"query_cache": c.Dict("query_cache", s.Schema{
				"memory": s.Object{
					"bytes": c.Int("memory_size_in_bytes"),
				},
			}),
			"request_cache": c.Dict("request_cache", s.Schema{
				"memory": s.Object{
					"bytes": c.Int("memory_size_in_bytes"),
				},
			}),
			"search": c.Dict("search", s.Schema{
				"query_time": s.Object{
					"ms": c.Int("query_time_in_millis"),
				},
				"query_total": s.Object{
					"count": c.Int("query_total"),
				},
			}),
			"store": c.Dict("store", s.Schema{
				"size": s.Object{
					"bytes": c.Int("size_in_bytes"),
				},
			}),
			"segments": c.Dict("segments", s.Schema{
				"count": c.Int("count"),
				"memory": s.Object{
					"bytes": c.Int("memory_in_bytes"),
				},
				"doc_values": s.Object{
					"memory": s.Object{
						"bytes": c.Int("doc_values_memory_in_bytes"),
					},
				},
				"fixed_bit_set": s.Object{
					"memory": s.Object{
						"bytes": c.Int("fixed_bit_set_memory_in_bytes"),
					},
				},
				"index_writer": s.Object{
					"memory": s.Object{
						"bytes": c.Int("index_writer_memory_in_bytes"),
					},
				},
				"norms": s.Object{
					"memory": s.Object{
						"bytes": c.Int("norms_memory_in_bytes"),
					},
				},
				"points": s.Object{
					"memory": s.Object{
						"bytes": c.Int("points_memory_in_bytes"),
					},
				},
				"stored_fields": s.Object{
					"memory": s.Object{
						"bytes": c.Int("stored_fields_memory_in_bytes"),
					},
				},
				"term_vectors": s.Object{
					"memory": s.Object{
						"bytes": c.Int("term_vectors_memory_in_bytes"),
					},
				},
				"terms": s.Object{
					"memory": s.Object{
						"bytes": c.Int("terms_memory_in_bytes"),
					},
				},
				"version_map": s.Object{
					"memory": s.Object{
						"bytes": c.Int("version_map_memory_in_bytes"),
					},
				},
			}),
		}),
		"fs": c.Dict("fs", s.Schema{
			"summary": c.Dict("total", s.Schema{
				"total": s.Object{
					"bytes": c.Int("total_in_bytes"),
				},
				"free": s.Object{
					"bytes": c.Int("free_in_bytes"),
				},
				"available": s.Object{
					"bytes": c.Int("available_in_bytes"),
				},
			}),
			"total": c.Dict("total", s.Schema{
				"available_in_bytes": c.Int("available_in_bytes"),
				"total_in_bytes":     c.Int("total_in_bytes"),
			}),
			"io_stats": c.Dict("io_stats", s.Schema{
				"total": c.Dict("total", s.Schema{
					"operations": s.Object{
						"count": c.Int("operations"),
					},
					"read": s.Object{
						"kb": c.Int("read_kilobytes"),
						"operations": s.Object{
							"count": c.Int("read_operations"),
						},
					},
					"write": s.Object{
						"kb": c.Int("write_kilobytes"),
						"operations": s.Object{
							"count": c.Int("write_operations"),
						},
					},
				}, c.DictOptional),
			}, c.DictOptional),
		}),
		"os": c.Dict("os", s.Schema{
			"cpu": c.Dict("cpu", s.Schema{
				"load_avg": c.Dict("load_average", s.Schema{
					"1m": c.Float("1m", s.Optional),
				}, c.DictOptional), // No load average reported by ES on Windows
			}),
			"cgroup": c.Dict("cgroup", s.Schema{
				"cpuacct": c.Dict("cpuacct", s.Schema{
					"usage": s.Object{
						"ns": c.Int("usage_nanos"),
					},
				}),
				"cpu": c.Dict("cpu", s.Schema{
					"cfs": s.Object{
						"quota": s.Object{
							"us": c.Int("cfs_quota_micros"),
						},
					},
					"stat": c.Dict("stat", s.Schema{
						"elapsed_periods": s.Object{
							"count": c.Int("number_of_elapsed_periods"),
						},
						"times_throttled": s.Object{
							"count": c.Int("number_of_times_throttled"),
						},
					}),
				}),
				"memory": c.Dict("memory", s.Schema{
					"control_group": c.Str("control_group"),
					// The two following values are currently string. See https://github.com/elastic/elasticsearch/pull/26166
					"limit": s.Object{
						"bytes": c.Str("limit_in_bytes"),
					},
					"usage": s.Object{
						"bytes": c.Str("usage_in_bytes"),
					},
				}),
			}, c.DictOptional),
		}),
		"process": c.Dict("process", s.Schema{
			"cpu": c.Dict("cpu", s.Schema{
				"pct": c.Int("percent"),
			}),
		}),
		"thread_pool": c.Dict("thread_pool", s.Schema{
			"bulk":   c.Dict("bulk", threadPoolStatsSchema, c.DictOptional),
			"index":  c.Dict("index", threadPoolStatsSchema, c.DictOptional),
			"write":  c.Dict("write", threadPoolStatsSchema, c.DictOptional),
			"get":    c.Dict("get", threadPoolStatsSchema),
			"search": c.Dict("search", threadPoolStatsSchema),
		}),
		"indexing_pressure": c.Dict("indexing_pressure", s.Schema{
			"memory": c.Dict("memory", s.Schema{
				"current":        c.Dict("current", current_memory_pressure),
				"total":          c.Dict("total", total_memory_pressure),
				"limit_in_bytes": c.Int("limit_in_bytes"),
			}),
		}),
		"ingest": c.Dict("ingest", s.Schema{
			"total": c.Dict("total", s.Schema{
				"count":          c.Int("count"),
				"time_in_millis": c.Int("time_in_millis"),
				"current":        c.Int("current"),
				"failed":         c.Int("failed"),
			}),
		}),
	}

	collectorSchema = s.Schema{
		"collection": s.Object{
			"count": c.Int("collection_count"),
			"ms":    c.Int("collection_time_in_millis"),
		},
	}

	current_memory_pressure = s.Schema{
		"combined_coordinating_and_primary_in_bytes": c.Int("combined_coordinating_and_primary_in_bytes"),
		"coordinating_in_bytes":                      c.Int("coordinating_in_bytes"),
		"primary_in_bytes":                           c.Int("primary_in_bytes"),
		"replica_in_bytes":                           c.Int("replica_in_bytes"),
		"all_in_bytes":                               c.Int("all_in_bytes"),
	}

	total_memory_pressure = s.Schema{
		"combined_coordinating_and_primary_in_bytes": c.Int("combined_coordinating_and_primary_in_bytes"),
		"coordinating_in_bytes":                      c.Int("coordinating_in_bytes"),
		"primary_in_bytes":                           c.Int("primary_in_bytes"),
		"replica_in_bytes":                           c.Int("replica_in_bytes"),
		"all_in_bytes":                               c.Int("all_in_bytes"),
		"coordinating_rejections":                    c.Int("coordinating_rejections"),
		"primary_rejections":                         c.Int("primary_rejections"),
		"replica_rejections":                         c.Int("replica_rejections"),
	}

	threadPoolStatsSchema = s.Schema{
		"queue": s.Object{
			"count": c.Int("queue"),
		},
		"rejected": s.Object{
			"count": c.Int("rejected"),
		},
	}
)

type nodesStruct struct {
	Nodes map[string]map[string]interface{} `json:"nodes"`
}

func eventsMapping(r mb.ReporterV2, m elasticsearch.MetricSetAPI, info elasticsearch.Info, content []byte) error {
	nodeData := &nodesStruct{}
	err := json.Unmarshal(content, nodeData)
	if err != nil {
		return errors.Wrap(err, "failure parsing Elasticsearch Node Stats API response")
	}

	masterNodeID, err := m.GetMasterNodeID()
	if err != nil {
		return err
	}

	var errs multierror.Errors
	for nodeID, node := range nodeData.Nodes {
		isMaster := nodeID == masterNodeID

		mlockall, err := m.IsMLockAllEnabled(nodeID)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "error determining if mlockall is set on Elasticsearch node"))
			continue
		}

		event := mb.Event{}

		event.RootFields = common.MapStr{}
		event.RootFields.Put("service.name", elasticsearch.ModuleName)

		event.ModuleFields = common.MapStr{
			"node": common.MapStr{
				"id":       nodeID,
				"mlockall": mlockall,
				"master":   isMaster,
			},
			"cluster": common.MapStr{
				"name": info.ClusterName,
				"id":   info.ClusterID,
			},
		}

		event.MetricSetFields, err = schema.Apply(node)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "failure to apply node schema"))
			continue
		}

		name, err := event.MetricSetFields.GetValue("name")
		if err != nil {
			errs = append(errs, elastic.MakeErrorForMissingField("name", elastic.Elasticsearch))
			continue
		}

		nameStr, ok := name.(string)
		if !ok {
			errs = append(errs, fmt.Errorf("name is not a string"))
			continue
		}
		event.ModuleFields.Put("node.name", nameStr)
		event.MetricSetFields.Delete("name")

		r.Event(event)
	}
	return errs.Err()
}
