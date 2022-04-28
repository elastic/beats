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

package mgr_osd_pool_stats

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/metricbeat/module/ceph/mgr"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type OsdPoolStat struct {
	PoolID       uint64 `json:"pool_id"`
	PoolName     string `json:"pool_name"`
	ClientIORate struct {
		ReadBytesSec  uint64 `json:"read_bytes_sec"`
		WriteBytesSec uint64 `json:"write_bytes_sec"`
		ReadOpPerSec  uint64 `json:"read_op_per_sec"`
		WriteOpPerSec uint64 `json:"write_op_per_sec"`
	} `json:"client_io_rate"`
}

func eventsMapping(content []byte) ([]mapstr.M, error) {
	var response []OsdPoolStat
	err := mgr.UnmarshalResponse(content, &response)
	if err != nil {
		return nil, errors.Wrap(err, "could not get response data")
	}

	var events []mapstr.M
	for _, stat := range response {
		event := mapstr.M{
			"pool_id":   stat.PoolID,
			"pool_name": stat.PoolName,
			"client_io_rate": mapstr.M{
				"read_bytes_sec":   stat.ClientIORate.ReadBytesSec,
				"write_bytes_sec":  stat.ClientIORate.WriteBytesSec,
				"read_op_per_sec":  stat.ClientIORate.ReadOpPerSec,
				"write_op_per_sec": stat.ClientIORate.WriteOpPerSec,
			},
		}
		events = append(events, event)
	}
	return events, nil
}
