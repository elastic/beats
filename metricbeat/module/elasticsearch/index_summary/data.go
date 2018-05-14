package index_summary

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
		"primaries": c.Dict("primaries", s.Schema{
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

func eventMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte) []error {
	var all struct {
		Data map[string]interface{} `json:"_all"`
	}

	err := json.Unmarshal(content, &all)
	if err != nil {
		r.Error(err)
		return []error{err}
	}

	var errs []error

	fields, err := schema.Apply(all.Data)
	if err != nil {
		errs = append(errs, err)
	}

	event := mb.Event{}
	event.RootFields = common.MapStr{}
	event.RootFields.Put("service.name", "elasticsearch")

	event.ModuleFields = common.MapStr{}
	event.ModuleFields.Put("cluster.name", info.ClusterName)
	event.ModuleFields.Put("cluster.id", info.ClusterID)

	event.MetricSetFields = fields

	r.Event(event)

	return errs
}
