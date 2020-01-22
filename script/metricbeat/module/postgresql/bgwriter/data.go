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

package bgwriter

import (
	"time"

	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
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
