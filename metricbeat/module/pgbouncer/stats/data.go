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
	"database":				c.Str("database"),
	"total_query_count":	c.Int("total_query_count"),
	"total_received":		c.Int("total_received"),
	"total_sent":			c.Int("total_sent"),
	"total_xact_time":		c.Int("total_xact_time"),
	"total_query_time":		c.Int("total_query_time"),
	"total_wait_time":		c.Int("total_wait_time"),
	"total_xact_count":		c.Int("total_xact_count"),
	"avg_xact_count":		c.Int("avg_xact_count"),
	"avg_query_count":		c.Int("avg_query_count"),
	"avg_recv":				c.Int("avg_recv"),
	"avg_sent":				c.Int("avg_sent"),
	"avg_xact_time":		c.Int("avg_xact_time"),
	"avg_query_time":		c.Int("avg_query_time"),
	"avg_wait_time":		c.Int("avg_wait_time"),
}