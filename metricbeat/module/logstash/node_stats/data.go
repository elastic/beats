package node_stats

import (
	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

var (
	schema = s.Schema{
		"events": c.Dict("events", s.Schema{
			"in":       c.Int("in"),
			"out":      c.Int("out"),
			"filtered": c.Int("filtered"),
		}),
	}
)

func eventMapping(node map[string]interface{}) (common.MapStr, *s.Errors) {
	return schema.Apply(node)
}
