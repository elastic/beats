package index_summary

import (
	"encoding/json"
	"time"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

var (
	schemaXPack = s.Schema{
		"primaries": c.Dict("primaries", s.Schema{
			"docs": c.Dict("docs", s.Schema{
				"count": c.Int("count"),
			}),
			"store": c.Dict("store", s.Schema{
				"size_in_bytes": c.Int("size_in_bytes"),
			}),
			"indexing": c.Dict("indexing", s.Schema{
				"index_total":             c.Int("index_total"),
				"index_time_in_millis":    c.Int("index_time_in_millis"),
				"is_throttled":            c.Bool("is_throttled"),
				"throttle_time_in_millis": c.Int("throttle_time_in_millis"),
			}),
			"search": c.Dict("search", s.Schema{
				"query_total":          c.Int("query_total"),
				"query_time_in_millis": c.Int("query_time_in_millis"),
			}),
		}),
		"total": c.Dict("total", s.Schema{
			"docs": c.Dict("docs", s.Schema{
				"count": c.Int("count"),
			}),
			"store": c.Dict("store", s.Schema{
				"size_in_bytes": c.Int("size_in_bytes"),
			}),
			"indexing": c.Dict("indexing", s.Schema{
				"index_total":             c.Int("index_total"),
				"index_time_in_millis":    c.Int("index_time_in_millis"),
				"is_throttled":            c.Bool("is_throttled"),
				"throttle_time_in_millis": c.Int("throttle_time_in_millis"),
			}),
			"search": c.Dict("search", s.Schema{
				"query_total":          c.Int("query_total"),
				"query_time_in_millis": c.Int("query_time_in_millis"),
			}),
		}),
	}
)

func eventMappingXPack(r mb.ReporterV2, m *MetricSet, info elasticsearch.Info, content []byte) []error {
	var all struct {
		Data map[string]interface{} `json:"_all"`
	}

	err := json.Unmarshal(content, &all)
	if err != nil {
		r.Error(err)
		return []error{err}
	}

	var errs []error

	fields, err := schemaXPack.Apply(all.Data)
	if err != nil {
		errs = append(errs, err)
	}

	nodeInfo, err := elasticsearch.GetNodeInfo(m.HTTP, m.HostData().SanitizedURI+statsPath, "")
	sourceNode := common.MapStr{
		"uuid":              nodeInfo.ID,
		"host":              nodeInfo.Host,
		"transport_address": nodeInfo.TransportAddress,
		"ip":                nodeInfo.IP,
		"name":              nodeInfo.Name,
		"timestamp":         common.Time(time.Now()),
	}
	event := mb.Event{}
	event.RootFields = common.MapStr{}
	event.RootFields.Put("indices_stats._all", fields)
	event.RootFields.Put("cluser_uuid", info.ClusterID)
	event.RootFields.Put("timestamp", common.Time(time.Now()))
	event.RootFields.Put("interval_ms", m.Module().Config().Period/time.Millisecond)
	event.RootFields.Put("type", "indices_stats")
	event.RootFields.Put("source_node", sourceNode)

	event.Index = ".monitoring-es-6-mb"

	r.Event(event)

	return errs
}
