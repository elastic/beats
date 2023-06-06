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

package replication

import (
	"time"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstrstr"
)

// Based on https://www.postgresql.org/docs/9.5/monitoring-stats.html#PG-STAT-REPLICATION-VIEW
var schema = s.Schema{
	"pid": c.Int("pid"),
	"user": s.Object{
		"id":   c.Int("usesysid", s.Optional),
		"name": c.Str("usename"),
	},
	"application_name": c.Str("application_name"),
	"client": s.Object{
		"address":  c.Str("client_addr", s.Optional),
		"hostname": c.Str("client_hostname", s.Optional),
		"port":     c.Int("client_port", s.Optional),
	},
	"backend_start": c.Time(time.RFC3339Nano, "backend_start"),
	"backend_xmin":  c.Uint("backend_xmin", s.Optional),
	"state":         c.Str("state"),
	"sent_lsn":      c.Str("sent_lsn"),
	"write_lsn":     c.Str("write_lsn"),
	"flush_lsn":     c.Str("flush_lsn"),
	"replay_lsn":    c.Str("replay_lsn"),
	"write_lag":     c.Uint("write_lag", s.Optional),
	"flush_lag":     c.Uint("flush_lag", s.Optional),
	"replay_lag":    c.Uint("replay_lag", s.Optional),
	"sync_priority": c.Int("sync_priority", s.Optional),
	"sync_state":    c.Str("sync_state"),
}
