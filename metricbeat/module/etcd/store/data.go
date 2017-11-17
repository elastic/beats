package store

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"

	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

var (
	schema = s.Schema{
		"gets": s.Object{
			"success": c.Int("getsSuccess"),
			"fail":    c.Int("getsFail"),
		},
		"sets": s.Object{
			"success": c.Int("setsSuccess"),
			"fail":    c.Int("setsFail"),
		},
		"delete": s.Object{
			"success": c.Int("deleteSuccess"),
			"fail":    c.Int("deleteFail"),
		},
		"update": s.Object{
			"success": c.Int("updateSuccess"),
			"fail":    c.Int("updateFail"),
		},
		"create": s.Object{
			"success": c.Int("createSuccess"),
			"fail":    c.Int("createFail"),
		},
		"compareandswap": s.Object{
			"success": c.Int("compareAndSwapSuccess"),
			"fail":    c.Int("compareAndSwapFail"),
		},
		"compareanddelete": s.Object{
			"success": c.Int("compareAndDeleteSuccess"),
			"fail":    c.Int("compareAndDeleteFail"),
		},
		"expire": s.Object{
			"count": c.Int("expireCount"),
		},
		"watchers": c.Int("watchers"),
	}
)

func eventMapping(content []byte) common.MapStr {
	var data map[string]interface{}
	json.Unmarshal(content, &data)
	event, _ := schema.Apply(data)
	return event
}
