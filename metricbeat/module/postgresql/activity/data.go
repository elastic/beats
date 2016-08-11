package activity

import (
	"time"

	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstrstr"
)

// Based on: https://www.postgresql.org/docs/9.2/static/monitoring-stats.html#PG-STAT-ACTIVITY-VIEW
var schema = s.Schema{
	"database": s.Object{
		"oid":  c.Int("datid"),
		"name": c.Str("datname"),
	},
	"pid": c.Int("pid"),
	"user": s.Object{
		"id":   c.Int("usesysid"),
		"name": c.Str("usename"),
	},
	"application_name": c.Str("application_name"),
	"client": s.Object{
		"address":  c.Str("client_addr"),
		"hostname": c.Str("client_hostname"),
		"port":     c.Int("client_port"),
	},
	"backend_start":     c.Time(time.RFC3339Nano, "backend_start"),
	"transaction_start": c.Time(time.RFC3339Nano, "xact_start", s.Optional),
	"query_start":       c.Time(time.RFC3339Nano, "query_start"),
	"state_change":      c.Time(time.RFC3339Nano, "state_change"),
	"waiting":           c.Bool("waiting"),
	"state":             c.Str("state"),
	"query":             c.Str("query"),
}

var eventMapping = schema.Apply
