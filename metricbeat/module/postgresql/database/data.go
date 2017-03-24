package database

import (
	"time"

	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstrstr"
)

// Based on https://www.postgresql.org/docs/9.2/static/monitoring-stats.html#PG-STAT-DATABASE-VIEW
var schema = s.Schema{
	"oid":                c.Int("datid"),
	"name":               c.Str("datname"),
	"number_of_backends": c.Int("numbackends"),
	"transactions": s.Object{
		"commit":   c.Int("xact_commit"),
		"rollback": c.Int("xact_rollback"),
	},
	"blocks": s.Object{
		"read": c.Int("blks_read"),
		"hit":  c.Int("blks_hit"),
		"time": s.Object{
			"read":  s.Object{"ms": c.Int("blk_read_time")},
			"write": s.Object{"ms": c.Int("blk_write_time")},
		},
	},
	"rows": s.Object{
		"returned": c.Int("tup_returned"),
		"fetched":  c.Int("tup_fetched"),
		"inserted": c.Int("tup_inserted"),
		"updated":  c.Int("tup_updated"),
		"deleted":  c.Int("tup_deleted"),
	},
	"conflicts": c.Int("conflicts"),
	"temporary": s.Object{
		"files": c.Int("temp_files"),
		"bytes": c.Int("temp_bytes"),
	},
	"deadlocks":   c.Int("deadlocks"),
	"stats_reset": c.Time(time.RFC3339Nano, "stats_reset", s.Optional),
}
