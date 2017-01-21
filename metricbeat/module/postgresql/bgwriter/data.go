package bgwriter

import (
	"time"

	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstrstr"
)

var schema = s.Schema{
	"checkpoints": s.Object{
		"scheduled": c.Int("checkpoints_timed"),
		"requested": c.Int("checkpoints_req"),
		"times": s.Object{
			"write": s.Object{"ms": c.Float("checkpoint_write_time")},
			"sync":  s.Object{"ms": c.Float("checkpoint_sync_time")},
		},
	},
	"buffers": s.Object{
		"checkpoints":   c.Int("buffers_checkpoint"),
		"clean":         c.Int("buffers_clean"),
		"clean_full":    c.Int("maxwritten_clean"),
		"backend":       c.Int("buffers_backend"),
		"backend_fsync": c.Int("buffers_backend_fsync"),
		"allocated":     c.Int("buffers_alloc"),
	},
	"stats_reset": c.Time(time.RFC3339Nano, "stats_reset", s.Optional),
}
