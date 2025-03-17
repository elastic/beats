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

package stats

import (
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstrstr"
)

// Based on pgbouncer show stats;
var schema = s.Schema{
	"database": c.Str("database"),
	"query_count": s.Object{
		"total": c.Int("total_query_count"),
		"avg":   c.Int("avg_query_count"),
	},
	"server_assignment_count": s.Object{
		"total": c.Int("total_server_assignment_count"),
		"avg":   c.Int("avg_server_assignment_count"),
	},
	"received": s.Object{
		"total": c.Int("total_received"),
		"avg":   c.Int("avg_recv"),
	},
	"sent": s.Object{
		"total": c.Int("total_sent"),
		"avg":   c.Int("avg_sent"),
	},
	"xact_time_us": s.Object{
		"total": c.Int("total_xact_time"),
		"avg":   c.Int("avg_xact_time"),
	},
	"query_time_us": s.Object{
		"total": c.Int("total_query_time"),
		"avg":   c.Int("avg_query_time"),
	},
	"wait_time_us": s.Object{
		"total": c.Int("total_wait_time"),
		"avg":   c.Int("avg_wait_time"),
	},
	"xact_count": s.Object{
		"total": c.Int("total_xact_count"),
		"avg":   c.Int("avg_xact_count"),
	},
}
