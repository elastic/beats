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

func eventsMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte) []error {

	var indicesStruct struct {
		Indices map[string]map[string]interface{} `json:"indices"`
	}

	err := json.Unmarshal(content, &indicesStruct)
	if err != nil {
		r.Error(err)
		return []error{err}
	}

	var errs []error
	for name, index := range indicesStruct.Indices {
		event := mb.Event{}
		event.MetricSetFields, err = schema.Apply(index)
		if err != nil {
			errs = append(errs, err)
		}
		// Write name here as full name only available as key
		event.MetricSetFields["name"] = name
		event.RootFields = common.MapStr{}
		event.RootFields.Put("service.name", "elasticsearch")
		event.ModuleFields = common.MapStr{}
		event.ModuleFields.Put("cluster.name", info.ClusterName)
		event.ModuleFields.Put("cluster.id", info.ClusterID)
		r.Event(event)
	}
	return errs
}
