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
	s "github.com/elastic/beats/v8/libbeat/common/schema"
	c "github.com/elastic/beats/v8/libbeat/common/schema/mapstrstr"
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
		"bytes": s.Object{
			"current": c.Int("bytes"),
			"limit":   c.Int("limit_maxbytes"),
		},
	}
)
