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

package mgr_pool_disk

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/metricbeat/module/ceph/mgr"
)

type DfResponse struct {
	Pools []struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Stats struct {
			BytesUsed uint64 `json:"bytes_used"`
			MaxAvail  uint64 `json:"max_avail"`
			Objects   uint64 `json:"objects"`
			KbUsed    uint64 `json:"kb_used"`
		} `json:"stats"`
	} `json:"pools"`
}

func eventsMapping(content []byte) ([]common.MapStr, error) {
	var response DfResponse
	err := mgr.UnmarshalResponse(content, &response)
	if err != nil {
		return nil, errors.Wrap(err, "could not get response data")
	}

	var events []common.MapStr
	for _, Pool := range response.Pools {
		event := common.MapStr{
			"name": Pool.Name,
			"id":   Pool.ID,
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
	return events, nil
}
