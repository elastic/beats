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
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Based on pgbouncer show pools;
var schema = s.Schema{
	"database":              c.Str("database"),
	"user":                  c.Str("user"),
	"cl_active":             c.Int("cl_active"),
	"cl_waiting":            c.Int("cl_waiting"),
	"cl_active_cancel_req":  c.Int("cl_active_cancel_req"),
	"cl_waiting_cancel_req": c.Int("cl_waiting_cancel_req"),
	"sv_active":             c.Int("sv_active"),
	"sv_active_cancel":      c.Int("sv_active_cancel"),
	"sv_being_canceled":     c.Int("sv_being_canceled"),
	"sv_idle":               c.Int("sv_idle"),
	"sv_used":               c.Int("sv_used"),
	"sv_tested":             c.Int("sv_tested"),
	"sv_login":              c.Int("sv_login"),
	"maxwait_us":            c.Int("maxwait_us"),
	"pool_mode":             c.Str("pool_mode"),
}

// MapResult maps a single result to a mapstr.M
func MapResult(result map[string]interface{}) (mapstr.M, error) {
	return schema.Apply(result)
}
