package oplog

import (
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

var schema = s.Schema{
	"size": s.Object{
		"allocated": c.Int("logSize"),
		"used": c.Int("used"),
	},
	"first": s.Object {
		"ts": c.Int("tFirst"),
	},
	"last": s.Object {
		"ts": c.Int("tLast"),
	},
	"window": c.Int("timeDiff"),
}
