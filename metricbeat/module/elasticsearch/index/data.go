package index

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

var (
	schema = s.Schema{
		"total": c.Dict("total", s.Schema{
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
	}
)

func eventsMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte) {

	var indicesStruct struct {
		Indices map[string]map[string]interface{} `json:"indices"`
	}

	json.Unmarshal(content, &indicesStruct)

	for name, index := range indicesStruct.Indices {
		event := mb.Event{}
		event.MetricSetFields = eventMapping(index)
		// Write name here as full name only available as key
		event.MetricSetFields["name"] = name
		event.RootFields = common.MapStr{}
		event.RootFields.Put("service.name", "elasticsearch")
		event.ModuleFields = common.MapStr{}
		event.ModuleFields.Put("cluster.name", info.ClusterName)
		event.ModuleFields.Put("cluster.id", info.ClusterID)
		r.Event(event)
	}
}

func eventMapping(node map[string]interface{}) common.MapStr {
	event, _ := schema.Apply(node)
	return event
}
