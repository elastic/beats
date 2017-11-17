package stats

import (
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	schema = s.Schema{
		"pid": c.Int("pid"),
		"uptime": s.Object{
			"sec": c.Int("uptime"),
		},
		"threads": c.Int("threads"),
		"connections": s.Object{
			"current": c.Int("curr_connections"),
			"total":   c.Int("total_connections"),
		},
		"get": s.Object{
			"hits":   c.Int("get_hits"),
			"misses": c.Int("get_misses"),
		},
		"cmd": s.Object{
			"get": c.Int("cmd_get"),
			"set": c.Int("cmd_set"),
		},
		"read": s.Object{
			"bytes": c.Int("bytes_read"),
		},
		"written": s.Object{
			"bytes": c.Int("bytes_written"),
		},
		"items": s.Object{
			"current": c.Int("curr_items"),
			"total":   c.Int("total_items"),
		},
		"evictions": c.Int("evictions"),
	}
)
