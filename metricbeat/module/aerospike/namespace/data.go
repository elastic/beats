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

package namespace

import (
	s "github.com/elastic/beats/v8/libbeat/common/schema"
	c "github.com/elastic/beats/v8/libbeat/common/schema/mapstrstr"
)

var schema = s.Schema{
	"client": s.Object{
		"delete": s.Object{
			"error":     c.Int("client_delete_error"),
			"not_found": c.Int("client_delete_not_found"),
			"success":   c.Int("client_delete_success"),
			"timeout":   c.Int("client_delete_timeout"),
		},
		"read": s.Object{
			"error":     c.Int("client_read_error"),
			"not_found": c.Int("client_read_not_found"),
			"success":   c.Int("client_read_success"),
			"timeout":   c.Int("client_read_timeout"),
		},
		"write": s.Object{
			"error":   c.Int("client_write_error"),
			"success": c.Int("client_write_success"),
			"timeout": c.Int("client_write_timeout"),
		},
	},
	"device": s.Object{
		"available": s.Object{
			"pct": c.Float("device_available_pct", s.Optional),
		},
		"free": s.Object{
			"pct": c.Float("device_free_pct", s.Optional),
		},
		"used": s.Object{
			"bytes": c.Int("device_used_bytes", s.Optional),
		},
		"total": s.Object{
			"bytes": c.Int("device_total_bytes", s.Optional),
		},
	},
	"hwm_breached": c.Bool("hwm_breached"),
	"memory": s.Object{
		"free": s.Object{
			"pct": c.Float("memory_free_pct"),
		},
		"used": s.Object{
			"data": s.Object{
				"bytes": c.Int("memory_used_data_bytes"),
			},
			"index": s.Object{
				"bytes": c.Int("memory_used_index_bytes"),
			},
			"sindex": s.Object{
				"bytes": c.Int("memory_used_sindex_bytes"),
			},
			"total": s.Object{
				"bytes": c.Int("memory_used_bytes"),
			},
		},
	},
	"objects": s.Object{
		"master": c.Int("master_objects"),
		"total":  c.Int("objects"),
	},
	"stop_writes": c.Bool("stop_writes"),
}
