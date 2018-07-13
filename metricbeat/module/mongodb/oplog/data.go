package oplog

import (
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

var schema = s.Schema{
	"log_size": s.Object{
		"allocated": c.Int("logSize"),
		"used": c.Int("used"),
	},
	"time": s.Object{
		"first": c.Int("tFirst"),
		"last": c.Int("tLast"),
		"diff": c.Int("timeDiff"),
	},
}
