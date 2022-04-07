// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package activity

import (
	"time"

	s "github.com/elastic/beats/v8/libbeat/common/schema"
	c "github.com/elastic/beats/v8/libbeat/common/schema/mapstrstr"
)

// Based on: https://www.postgresql.org/docs/9.2/static/monitoring-stats.html#PG-STAT-ACTIVITY-VIEW
var schema = s.Schema{
	"database": s.Object{
		"oid":  c.Int("datid", s.Optional),
		"name": c.Str("datname"),
	},
	"pid": c.Int("pid"),
	"user": s.Object{
		"id":   c.Int("usesysid", s.Optional),
		"name": c.Str("usename"),
	},
	"application_name": c.Str("application_name"),
	"client": s.Object{
		"address":  c.Str("client_addr"),
		"hostname": c.Str("client_hostname"),
		"port":     c.Int("client_port", s.Optional),
	},
	"backend_start":     c.Time(time.RFC3339Nano, "backend_start"),
	"transaction_start": c.Time(time.RFC3339Nano, "xact_start", s.Optional),
	"query_start":       c.Time(time.RFC3339Nano, "query_start", s.Optional),
	"state_change":      c.Time(time.RFC3339Nano, "state_change", s.Optional),
	"waiting":           c.Bool("waiting", s.Optional),
	"state":             c.Str("state"),
	"query":             c.Str("query"),
	"backend_type":      c.Str("backend_type", s.Optional),
}

// Fields available in events from backend activity.
var backendSchema = s.Schema{
	"pid":             c.Int("pid"),
	"backend_start":   c.Time(time.RFC3339Nano, "backend_start"),
	"wait_event_type": c.Str("wait_event_type"),
	"wait_event":      c.Str("wait_event"),
	"backend_type":    c.Str("backend_type"),
}
