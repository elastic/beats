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

package pool_disk

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Stats struct {
	BytesUsed int64 `json:"bytes_used"`
	MaxAvail  int64 `json:"max_avail"`
	Objects   int64 `json:"objects"`
	KbUsed    int64 `json:"kb_used"`
}

type Pool struct {
	Id    int64  `json:"id"`
	Name  string `json:"name"`
	Stats Stats  `json:"stats"`
}

type Output struct {
	Pools []Pool `json:"pools"`
}

type DfRequest struct {
	Status string `json:"status"`
	Output Output `json:"output"`
}

func eventsMapping(content []byte) []common.MapStr {
	var d DfRequest
	err := json.Unmarshal(content, &d)
	if err != nil {
		logp.Err("Error: %+v", err)
	}

	events := []common.MapStr{}

	for _, Pool := range d.Output.Pools {
		event := common.MapStr{
			"name": Pool.Name,
			"id":   Pool.Id,
			"stats": common.MapStr{
				"used": common.MapStr{
					"bytes": Pool.Stats.BytesUsed,
					"kb":    Pool.Stats.KbUsed,
				},
				"available": common.MapStr{
					"bytes": Pool.Stats.MaxAvail,
				},
				"objects": Pool.Stats.Objects,
			},
		}

		events = append(events, event)

	}

	return events
}
