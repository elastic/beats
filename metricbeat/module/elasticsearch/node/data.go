package node

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	schema = s.Schema{
		"name":    c.Str("name"),
		"version": c.Str("version"),
		"jvm": c.Dict("jvm", s.Schema{
			"version": c.Str("version"),
			"memory": c.Dict("mem", s.Schema{
				"heap": s.Object{
					"init": s.Object{
						"bytes": c.Int("heap_init_in_bytes"),
					},
					"max": s.Object{
						"bytes": c.Int("heap_max_in_bytes"),
					},
				},
				"nonheap": s.Object{
					"init": s.Object{
						"bytes": c.Int("non_heap_init_in_bytes"),
					},
					"max": s.Object{
						"bytes": c.Int("non_heap_max_in_bytes"),
					},
				},
			}),
		}),
		"process": c.Dict("process", s.Schema{
			"mlockall": c.Bool("mlockall"),
		}),
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
		event, errs := eventMapping(node)
		// Write name here as full name only available as key
		event["name"] = name
		event[mb.ModuleDataKey] = common.MapStr{
			"cluster": common.MapStr{
				"name": nodesStruct.ClusterName,
			},
		}
		events = append(events, event)
		errors.AddErrors(errs)
	}

	return events, errors
}

func eventMapping(node map[string]interface{}) (common.MapStr, *s.Errors) {
	return schema.Apply(node)
}
