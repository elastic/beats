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
	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	schema = s.Schema{
		"notifications": s.Object{
			"queue_length": c.Int("prometheus_notifications_queue_length"),
			"dropped":      c.Int("prometheus_notifications_dropped_total"),
		},
		"processes": s.Object{
			"open_fds": c.Int("process_open_fds"),
		},
		"storage": s.Object{
			"chunks_to_persist": c.Int("prometheus_local_storage_chunks_to_persist"),
		},
	}
)

func eventMapping(entries map[string]interface{}) (common.MapStr, error) {
	data, _ := schema.Apply(entries)
	return data, nil
}
