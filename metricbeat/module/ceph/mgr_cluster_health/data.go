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

package mgr_cluster_health

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/metricbeat/module/ceph/mgr"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type StatusResponse struct {
	Health struct {
		Status string `json:"status"`
	} `json:"health"`
}

type TimeSyncStatusResponse struct {
	Timechecks struct {
		RoundStatus string `json:"round_status"`
		Epoch       int64  `json:"epoch"`
		Round       int64  `json:"round"`
	} `json:"timechecks"`
}

func eventMapping(statusContent, timeSyncStatusContent []byte) (mapstr.M, error) {
	var statusResponse StatusResponse
	err := mgr.UnmarshalResponse(statusContent, &statusResponse)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal response")
	}

	var timeSyncStatusResponse TimeSyncStatusResponse
	err = mgr.UnmarshalResponse(timeSyncStatusContent, &timeSyncStatusResponse)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal response")
	}

	return mapstr.M{
		"overall_status": statusResponse.Health.Status,
		"timechecks": mapstr.M{
			"epoch": timeSyncStatusResponse.Timechecks.Epoch,
			"round": mapstr.M{
				"value":  timeSyncStatusResponse.Timechecks.Round,
				"status": timeSyncStatusResponse.Timechecks.RoundStatus,
			},
		},
	}, nil
}
