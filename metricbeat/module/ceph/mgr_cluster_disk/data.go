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

package mgr_cluster_disk

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/metricbeat/module/ceph/mgr"
)

type DfResponse struct {
	Stats struct {
		TotalBytes          uint64 `json:"total_bytes"`
		TotalAvailableBytes uint64 `json:"total_avail_bytes"`
		TotalUsedBytes      uint64 `json:"total_used_bytes"`
	} `json:"stats"`
}

func eventMapping(content []byte) (common.MapStr, error) {
	var response DfResponse
	err := mgr.UnmarshalResponse(content, &response)
	if err != nil {
		return nil, errors.Wrap(err, "could not get response data")
	}

	return common.MapStr{
		"used": common.MapStr{
			"bytes": response.Stats.TotalUsedBytes,
		},
		"total": common.MapStr{
			"bytes": response.Stats.TotalBytes,
		},
		"available": common.MapStr{
			"bytes": response.Stats.TotalAvailableBytes,
		},
	}, nil
}
