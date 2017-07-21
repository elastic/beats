package status

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

var (
	schema = s.Schema{
		"uuid": c.Str("uuid"),
		"name": c.Str("name"),
		"version": c.Dict("version", s.Schema{
			"number": c.Str("number"),
		}),
		"status": c.Dict("status", s.Schema{
			"overall": c.Dict("overall", s.Schema{
				"state": c.Str("state"),
			}),
		}),
		"metrics": c.Dict("metrics", s.Schema{
			"requests": c.Dict("requests", s.Schema{
				"total":       c.Int("total"),
				"disconnects": c.Int("disconnects"),
			}),
			"concurrent_connections": c.Int("concurrent_connections"),
		}),
	}
)

type OverallMetrics struct {
	Metrics map[string][][]uint64
}

func eventMapping(content []byte) common.MapStr {
	var data map[string]interface{}
	json.Unmarshal(content, &data)
	event, _ := schema.Apply(data)
	return event
}
