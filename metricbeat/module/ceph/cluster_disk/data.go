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

package cluster_disk

import (
	"encoding/json"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/pkg/errors"
)

type StatsCluster struct {
	TotalUsedBytes  int64 `json:"total_used_bytes"`
	TotalBytes      int64 `json:"total_bytes"`
	TotalAvailBytes int64 `json:"total_avail_bytes"`
}

type Output struct {
	StatsCluster StatsCluster `json:"stats"`
}

type DfRequest struct {
	Status string `json:"status"`
	Output Output `json:"output"`
}

func eventMapping(content []byte) (mapstr.M, error) {
	var d DfRequest
	err := json.Unmarshal(content, &d)
	if err != nil {
		return nil, errors.Wrap(err, "could not get DFRequest data")
	}

	return mapstr.M{
		"used": mapstr.M{
			"bytes": d.Output.StatsCluster.TotalUsedBytes,
		},
		"total": mapstr.M{
			"bytes": d.Output.StatsCluster.TotalBytes,
		},
		"available": mapstr.M{
			"bytes": d.Output.StatsCluster.TotalAvailBytes,
		},
	}, nil
}
