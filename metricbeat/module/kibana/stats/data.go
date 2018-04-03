package stats

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	schema = s.Schema{
		"cluster_uuid": c.Str("cluster_uuid"),
		"name":         c.Str("name"),
		"uuid":         c.Str("uuid"),
		"version": c.Dict("version", s.Schema{
			"number": c.Str("number"),
		}),
		"status": c.Dict("status", s.Schema{
			"overall": c.Dict("overall", s.Schema{
				"state": c.Str("state"),
			}),
		}),
		"response_times": c.Dict("response_times", s.Schema{
			"avg": s.Object{
				"ms": c.Float("avg_in_millis"),
			},
			"max": s.Object{
				"ms": c.Int("max_in_millis"),
			},
		}),
		"requests": c.Dict("requests", s.Schema{
			"total":       c.Int("total"),
			"disconnects": c.Int("disconnects"),
		}),
		"concurrent_connections": c.Int("concurrent_connections"),
		"sockets": c.Dict("sockets", s.Schema{
			"http": c.Dict("http", s.Schema{
				"total": c.Int("total"),
			}),
			"https": c.Dict("https", s.Schema{
				"total": c.Int("total"),
			}),
		}),
		"event_loop_delay": c.Float("event_loop_delay"),
		"process": c.Dict("process", s.Schema{
			"memory": c.Dict("mem", s.Schema{
				"heap": s.Object{
					"max": s.Object{
						"bytes": c.Int("heap_max_in_bytes"),
					},
					"used": s.Object{
						"bytes": c.Int("heap_used_in_bytes"),
					},
				},
				"resident_set_size": s.Object{
					"bytes": c.Int("resident_set_size_in_bytes"),
				},
				"external": s.Object{
					"bytes": c.Int("external_in_bytes"),
				},
			}),
			"pid": c.Int("pid"),
			"uptime": s.Object{
				"ms": c.Int("uptime_ms"),
			},
		}),
	}
)

func eventMapping(r mb.ReporterV2, content []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		r.Error(err)
		return err
	}

	dataFields, err := schema.Apply(data)
	event := mb.Event{}

	event.RootFields = common.MapStr{}
	event.RootFields.Put("service.name", "kibana")

	// Set elasticsearch cluster id
	if clusterID, ok := dataFields["cluster_uuid"]; ok {
		delete(dataFields, "cluster_uuid")
		event.RootFields.Put("elasticsearch.cluster.id", clusterID)
	}

	event.MetricSetFields = dataFields

	r.Event(event)

	return err
}
