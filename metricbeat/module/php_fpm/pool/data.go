package pool

import (
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

var (
	schema = s.Schema{
		"name": c.Str("pool"),
		"connections": s.Object{
			"accepted": c.Int("accepted conn"),
			"queued":   c.Int("listen queue"),
		},
		"processes": s.Object{
			"idle":   c.Int("idle processes"),
			"active": c.Int("active processes"),
		},
		"slow_requests": c.Int("slow requests"),
	}
)
