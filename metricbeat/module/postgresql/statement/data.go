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
