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

package pools

import (
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstrstr"
)

// Based on pgbouncer show pools;
var schema = s.Schema{
	"database": c.Str("database"),
	"user":     c.Str("user"),
	"client": s.Object{
		"active":             c.Int("cl_active"),
		"waiting":            c.Int("cl_waiting"),
		"active_cancel_req":  c.Int("cl_active_cancel_req"),
		"waiting_cancel_req": c.Int("cl_waiting_cancel_req"),
	},
	"server": s.Object{
		"active":         c.Int("sv_active"),
		"active_cancel":  c.Int("sv_active_cancel"),
		"being_canceled": c.Int("sv_being_canceled"),
		"idle":           c.Int("sv_idle"),
		"used":           c.Int("sv_used"),
		"tested":         c.Int("sv_tested"),
		"login":          c.Int("sv_login"),
	},
	"maxwait_us": c.Int("maxwait_us"),
	"pool_mode":  c.Str("pool_mode"),
}
