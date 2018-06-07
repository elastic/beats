package statement

import (
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

// Based on: https://www.postgresql.org/docs/9.2/static/monitoring-stats.html#PG-STAT-ACTIVITY-VIEW
var schema = s.Schema{
	"user": s.Object{
		"id": c.Int("userid"),
	},
	"database": s.Object{
		"oid": c.Int("dbid"),
	},
	"query": s.Object{
		"id":    c.Str("queryid"),
		"text":  c.Str("query"),
		"calls": c.Int("calls"),
		"rows":  c.Int("rows"),
		"time": s.Object{
			"total":  s.Object{"ms": c.Float("total_time")},
			"min":    s.Object{"ms": c.Float("min_time")},
			"max":    s.Object{"ms": c.Float("max_time")},
			"mean":   s.Object{"ms": c.Float("mean_time")},
			"stddev": s.Object{"ms": c.Float("stddev_time")},
		},
		"memory": s.Object{
			"shared": s.Object{
				"hit":     c.Int("shared_blks_hit"),
				"read":    c.Int("shared_blks_read"),
				"dirtied": c.Int("shared_blks_dirtied"),
				"written": c.Int("shared_blks_written"),
			},
			"local": s.Object{
				"hit":     c.Int("local_blks_hit"),
				"read":    c.Int("local_blks_read"),
				"dirtied": c.Int("local_blks_dirtied"),
				"written": c.Int("local_blks_written"),
			},
			"temp": s.Object{
				"read":    c.Int("temp_blks_read"),
				"written": c.Int("temp_blks_written"),
			},
		},
	},
}
