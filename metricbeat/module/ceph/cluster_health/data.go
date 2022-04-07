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

package cluster_health

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
)

// Timecheck contains part of the response from a HealthRequest
type Timecheck struct {
	RoundStatus string `json:"round_status"`
	Epoch       int64  `json:"epoch"`
	Round       int64  `json:"round"`
}

// Output is the body of the status response
type Output struct {
	OverallStatus string    `json:"overall_status"`
	Timechecks    Timecheck `json:"timechecks"`
}

// HealthRequest represents the response to a cluster health request
type HealthRequest struct {
	Status string `json:"status"`
	Output Output `json:"output"`
}

func eventMapping(content []byte) (common.MapStr, error) {
	var d HealthRequest
	err := json.Unmarshal(content, &d)
	if err != nil {
		return nil, errors.Wrap(err, "error getting HealthRequest data")
	}

	return common.MapStr{
		"overall_status": d.Output.OverallStatus,
		"timechecks": common.MapStr{
			"epoch": d.Output.Timechecks.Epoch,
			"round": common.MapStr{
				"value":  d.Output.Timechecks.Round,
				"status": d.Output.Timechecks.RoundStatus,
			},
		},
	}, nil
}
