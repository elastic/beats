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

type nodesStruct struct {
	ClusterName string                            `json:"cluster_name"`
	Nodes       map[string]map[string]interface{} `json:"nodes"`
}

func eventsMapping(r mb.ReporterV2, content []byte) []error {

	nodeData := &nodesStruct{}
	err := json.Unmarshal(content, nodeData)
	if err != nil {
		r.Error(err)
		return nil
	}

	var errs []error
	for name, node := range nodeData.Nodes {
		event := mb.Event{}

		event.MetricSetFields, err = schema.Apply(node)
		if err != nil {
			errs = append(errs, err)
		}

		event.ModuleFields = common.MapStr{
			"node": common.MapStr{
				"name": name,
			},
			"cluster": common.MapStr{
				"name": nodeData.ClusterName,
			},
		}
		event.RootFields = common.MapStr{}
		event.RootFields.Put("service.name", "elasticsearch")
		r.Event(event)
	}
	return errs
}
