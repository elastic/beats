package node_stats

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	schema = s.Schema{
		"jvm": c.Dict("jvm", s.Schema{
			"mem": c.Dict("mem", s.Schema{
				"heap": s.Object{
					"used": s.Object{
						"bytes": c.Int("heap_used_in_bytes"),
					},
					"max": s.Object{
						"bytes": c.Int("heap_max_in_bytes"),
					},
					"percent": c.Int("heap_used_percent"),
				},
				"pools": c.Dict("pools", s.Schema{
					"young":    c.Dict("young", poolSchema),
					"survivor": c.Dict("survivor", poolSchema),
					"old":      c.Dict("old", poolSchema),
				}),
			}),
			"gc": c.Dict("gc", s.Schema{
				"collectors": c.Dict("collectors", s.Schema{
					"young": c.Dict("young", collectorSchema),
					"old":   c.Dict("old", collectorSchema),
				}),
			}),
		}),
		"indices": c.Dict("indices", s.Schema{
			"docs": c.Dict("docs", s.Schema{
				"count":   c.Int("count"),
				"deleted": c.Int("deleted"),
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
			}),
			"fielddata": c.Dict("fielddata", s.Schema{
				"memory": s.Object{
					"bytes": c.Int("memory_size_in_bytes"),
				},
			}),
			"search": c.Dict("search", s.Schema{
				"query": s.Object{
					"total":   c.Int("query_total"),
					"current": c.Int("query_current"),
					"ms":      c.Int("query_time_in_millis"),
				},
			}),
			"indexing": c.Dict("indexing", s.Schema{
				"index": s.Object{
					"total":   c.Int("index_total"),
					"current": c.Int("index_current"),
					"ms":      c.Int("index_time_in_millis"),
				},
			}),
		}),
		"threadpool": c.Dict("thread_pool", s.Schema{
			"index": c.Dict("index", s.Schema{
				"queue":    c.Int("queue"),
				"active":   c.Int("active"),
				"rejected": c.Int("rejected"),
			}),
			"bulk": c.Dict("bulk", s.Schema{
				"queue":    c.Int("queue"),
				"active":   c.Int("active"),
				"rejected": c.Int("rejected"),
			}),
			"search": c.Dict("search", s.Schema{
				"queue":    c.Int("queue"),
				"active":   c.Int("active"),
				"rejected": c.Int("rejected"),
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
		}),
		"transport": c.Dict("transport", s.Schema{
			"rx": s.Object{
				"bytes": c.Int("rx_size_in_bytes"),
			},
			"tx": s.Object{
				"bytes": c.Int("tx_size_in_bytes"),
			},
		}),
	}

	poolSchema = s.Schema{
		"used": s.Object{
			"bytes": c.Int("used_in_bytes"),
		},
		"max": s.Object{
			"bytes": c.Int("max_in_bytes"),
		},
		"peak": s.Object{
			"bytes": c.Int("peak_used_in_bytes"),
		},
		"peak_max": s.Object{
			"bytes": c.Int("peak_max_in_bytes"),
		},
	}

	collectorSchema = s.Schema{
		"collection": s.Object{
			"count": c.Int("collection_count"),
			"ms":    c.Int("collection_time_in_millis"),
		},
	}
)

func eventsMapping(content []byte) ([]common.MapStr, error) {
	nodesStruct := struct {
		ClusterName string                            `json:"cluster_name"`
		Nodes       map[string]map[string]interface{} `json:"nodes"`
	}{}

	json.Unmarshal(content, &nodesStruct)

	var events []common.MapStr
	errors := s.NewErrors()

	for name, node := range nodesStruct.Nodes {
		event, errs := schema.Apply(node)
		// Write name here as full name only available as key
		event[mb.ModuleDataKey] = common.MapStr{
			"node": common.MapStr{
				"name": name,
			},
			"cluster": common.MapStr{
				"name": nodesStruct.ClusterName,
			},
		}
		event[mb.NamespaceKey] = "node.stats"
		events = append(events, event)
		errors.AddErrors(errs)
	}

	return events, errors
}
