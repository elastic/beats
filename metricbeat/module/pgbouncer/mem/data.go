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

package mem

import (
	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstrstr"
)

var schema = s.Schema{
	"user_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"credentials_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"db_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"peer_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"peer_pool_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"pool_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"outstanding_request_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"server_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"iobuf_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"var_list_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
	"server_prepared_statement_cache": s.Object{
		"size":     c.Int("size"),
		"used":     c.Int("used"),
		"free":     c.Int("free"),
		"memtotal": c.Int("memtotal"),
	},
}
