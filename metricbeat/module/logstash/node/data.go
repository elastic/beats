package node

import (
	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

var (
	schema = s.Schema{
		"host":    c.Str("host"),
		"version": c.Str("version"),
		"jvm": c.Dict("jvm", s.Schema{
			"version": c.Str("version"),
			"pid":     c.Int("pid"),
		}),
	}
)

func eventMapping(node map[string]interface{}) (common.MapStr, *s.Errors) {
	return schema.Apply(node)
}
