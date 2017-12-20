package cluster_stats

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	schema = s.Schema{
		"status": c.Str("status"),
		"indices": c.Dict("indices", s.Schema{
			"count": c.Int("count"),
			"shards": c.Dict("shards", s.Schema{
				"total": c.Int("total"),
			}),
			"docs": c.Dict("docs", s.Schema{
				"count": c.Int("count"),
			}),
			"segments": c.Dict("segments", s.Schema{
				"count": c.Int("count"),
			}),
		}),
		"nodes": c.Dict("nodes", s.Schema{
			"count": c.Dict("count", s.Schema{
				"total": c.Int("total"),
			}),
			"jvm": c.Dict("jvm", s.Schema{
				"mem": c.Dict("mem", s.Schema{
					"heap": s.Object{
						"used": c.Int("heap_used_in_bytes"),
						"max":  c.Int("heap_max_in_bytes"),
					},
				}),
			}),
			"fs": c.Dict("fs", s.Schema{
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
)

func eventsMapping(content []byte) ([]common.MapStr, error) {
	var events []common.MapStr

	var clusterStruct map[string]interface{}
	err := json.Unmarshal(content, &clusterStruct)
	if err != nil {
		return events, err
	}

	clusterName := clusterStruct["cluster_name"]

	event, errs := schema.Apply(clusterStruct)

	// Write name here as full name only available as key
	event[mb.ModuleDataKey] = common.MapStr{
		"cluster": common.MapStr{
			"name": clusterName,
		},
	}
	event[mb.NamespaceKey] = "cluster.stats"
	events = append(events, event)

	return events, errs
}
