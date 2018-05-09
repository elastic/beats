package node_stats

import (
	"encoding/json"

	"time"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

var (
	sourceNodeXpack = s.Schema{
		"host":              c.Str("host"),
		"transport_address": c.Str("transport_address"),
		"ip":                c.Str("ip"),
		"name":              c.Str("name"),
	}

	schemaXpack = s.Schema{
		"indices": c.Dict("indices", s.Schema{
			"docs": c.Dict("docs", s.Schema{
				"count": c.Int("count"),
			}),
			"store": c.Dict("store", s.Schema{
				"size_in_bytes": c.Int("size_in_bytes"),
			}),
			"indexing": c.Dict("indexing", s.Schema{
				"index_total":             c.Int("index_total"),
				"index_time_in_millis":    c.Int("index_time_in_millis"),
				"throttle_time_in_millis": c.Int("throttle_time_in_millis"),
			}),
			"search": c.Dict("search", s.Schema{
				"query_total":          c.Int("query_total"),
				"query_time_in_millis": c.Int("query_time_in_millis"),
			}),
			"query_cache": c.Dict("query_cache", s.Schema{
				"memory_size_in_bytes": c.Int("memory_size_in_bytes"),
				"hit_count":            c.Int("hit_count"),
				"miss_count":           c.Int("miss_count"),
				"evictions":            c.Int("evictions"),
			}),
			"fielddata": c.Dict("fielddata", s.Schema{
				"memory_size_in_bytes": c.Int("memory_size_in_bytes"),
				"evictions":            c.Int("evictions"),
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
			"request_cache": c.Dict("request_cache", s.Schema{
				"memory_size_in_bytes": c.Int("memory_size_in_bytes"),
				"evictions":            c.Int("evictions"),
				"hit_count":            c.Int("hit_count"),
				"miss_count":           c.Int("miss_count"),
			}),
		}),
		"os": c.Dict("os", s.Schema{
			"cpu": c.Dict("cpu", s.Schema{
				"load_average": c.Dict("load_average", s.Schema{
					"1m":  c.Float("1m"),
					"5m":  c.Float("5m"),
					"15m": c.Float("15m"),
				}),
			}),
			"cgroup": c.Dict("cgroup", s.Schema{
				"cpuacct": c.Dict("cpuacct", s.Schema{
					"control_group": c.Str("control_group"),
					"usage_nanos":   c.Int("usage_nanos"),
				}),
				"cpu": c.Dict("cpu", s.Schema{
					"control_group":     c.Str("control_group"),
					"cfs_period_micros": c.Int("cfs_period_micros"),
					"cfs_quota_micros":  c.Int("cfs_quota_micros"),
					"stat": c.Dict("stat", s.Schema{
						"number_of_elapsed_periods": c.Int("number_of_elapsed_periods"),
						"number_of_times_throttled": c.Int("number_of_times_throttled"),
						"time_throttled_nanos":      c.Int("time_throttled_nanos"),
					}),
				}),
				"memory": c.Dict("memory", s.Schema{
					"control_group": c.Str("control_group"),
					// The two following values are currently string. See https://github.com/elastic/elasticsearch/pull/26166
					"limit_in_bytes": c.Str("limit_in_bytes"),
					"usage_in_bytes": c.Str("usage_in_bytes"),
				}),
			}),
		}),
		"process": c.Dict("process", s.Schema{
			"open_file_descriptors": c.Int("open_file_descriptors"),
			"max_file_descriptors":  c.Int("max_file_descriptors"),
			"cpu": c.Dict("cpu", s.Schema{
				"percent": c.Int("percent"),
			}),
		}),
		"jvm": c.Dict("jvm", s.Schema{
			"mem": c.Dict("mem", s.Schema{
				"heap_used_in_bytes": c.Int("heap_used_in_bytes"),
				"heap_used_percent":  c.Int("heap_used_percent"),
				"heap_max_in_bytes":  c.Int("heap_max_in_bytes"),
			}),
			"gc": c.Dict("gc", s.Schema{
				"collectors": c.Dict("collectors", s.Schema{
					"young": c.Dict("young", s.Schema{
						"collection_count":          c.Int("collection_count"),
						"collection_time_in_millis": c.Int("collection_time_in_millis"),
					}),
					"old": c.Dict("young", s.Schema{
						"collection_count":          c.Int("collection_count"),
						"collection_time_in_millis": c.Int("collection_time_in_millis"),
					}),
				}),
			}),
		}),
		"thread_pool": c.Dict("thread_pool", s.Schema{
			"bulk": c.Dict("bulk", s.Schema{
				"threads":  c.Int("threads"),
				"queue":    c.Int("queue"),
				"rejected": c.Int("rejected"),
			}),
			"generic": c.Dict("generic", s.Schema{
				"threads":  c.Int("threads"),
				"queue":    c.Int("queue"),
				"rejected": c.Int("rejected"),
			}),
			"get": c.Dict("get", s.Schema{
				"threads":  c.Int("threads"),
				"queue":    c.Int("queue"),
				"rejected": c.Int("rejected"),
			}),
			"index": c.Dict("index", s.Schema{
				"threads":  c.Int("threads"),
				"queue":    c.Int("queue"),
				"rejected": c.Int("rejected"),
			}),
			"management": c.Dict("management", s.Schema{
				"threads":  c.Int("threads"),
				"queue":    c.Int("queue"),
				"rejected": c.Int("rejected"),
			}),
			"search": c.Dict("search", s.Schema{
				"threads":  c.Int("threads"),
				"queue":    c.Int("queue"),
				"rejected": c.Int("rejected"),
			}),
			"watcher": c.Dict("watcher", s.Schema{
				"threads":  c.Int("threads"),
				"queue":    c.Int("queue"),
				"rejected": c.Int("rejected"),
			}),
		}),
		"fs": c.Dict("fs", s.Schema{
			"summary": c.Dict("total", s.Schema{
				"total_in_bytes":     c.Int("total_in_bytes"),
				"free_in_bytes":      c.Int("free_in_bytes"),
				"available_in_bytes": c.Int("available_in_bytes"),
			}),
		}),
	}
)

func eventsMappingXPack(r mb.ReporterV2, m *MetricSet, content []byte) {
	nodesStruct := struct {
		ClusterName string                            `json:"cluster_name"`
		Nodes       map[string]map[string]interface{} `json:"nodes"`
	}{}

	json.Unmarshal(content, &nodesStruct)

	// Normally the nodeStruct should only contain one node. But if _local is removed
	// from the path and Metricbeat is not installed on the same machine as the node
	// it will provid the data for multiple nodes. This will mean the detection of the
	// master node will not be accurate anymore as often in these cases a proxy is in front
	// of ES and it's not know if the request will be routed to the same node as before.
	for nodeID, node := range nodesStruct.Nodes {
		clusterID, err := elasticsearch.GetClusterID(m.http, m.HostData().SanitizedURI, nodeID)
		if err != nil {
			logp.Err("could not fetch cluster id: %s", err)
			continue
		}

		isMaster, _ := elasticsearch.IsMaster(m.http, m.HostData().SanitizedURI)

		event := mb.Event{}
		// Build source_node object
		sourceNode, _ := sourceNodeXpack.Apply(node)
		sourceNode["uuid"] = nodeID

		nodeData, _ := schemaXpack.Apply(node)
		nodeData["node_master"] = isMaster
		nodeData["node_id"] = nodeID

		event.RootFields = common.MapStr{
			"timestamp":    time.Now(),
			"cluster_uuid": clusterID,
			"interval_ms":  m.Module().Config().Period.Nanoseconds() / 1000 / 1000,
			"type":         "node_stats",
			"source_node":  sourceNode,
			"node_stats":   nodeData,
		}

		// Hard coded index prefix for monitoring, no detection done for ES version at the moment
		// It has an additonal md in the name to make it clear the data is coming from metricbeat
		event.Index = ".monitoring-es-6-mb"
		r.Event(event)
	}
}
