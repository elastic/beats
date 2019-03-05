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

package pool

import (
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

var (
	schema = s.Schema{
		"name":            c.Str("pool"),
		"process_manager": c.Str("process manager"),
		"slow_requests":   c.Int("slow requests"),
		"start_time":      c.Int("start time"),
		"start_since":     c.Int("start since"),
		"connections": s.Object{
			"accepted":         c.Int("accepted conn"),
			"listen_queue_len": c.Int("listen queue len"),
			"max_listen_queue": c.Int("max listen queue"),
			"queued":           c.Int("listen queue"),
		},
		"processes": s.Object{
			"active":               c.Int("active processes"),
			"idle":                 c.Int("idle processes"),
			"max_active":           c.Int("max active processes"),
			"max_children_reached": c.Int("max children reached"),
			"total":                c.Int("total processes"),
		},
	}
)
