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

package mgr_osd_perf

import (
	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/metricbeat/module/ceph/mgr"
)

type OsdPerfResponse struct {
	OsdStats struct {
		OsdPerfInfos []struct {
			ID        int64 `json:"id"`
			PerfStats struct {
				CommitLatencyMs uint64 `json:"commit_latency_ms"`
				ApplyLatencyMs  uint64 `json:"apply_latency_ms"`
				CommitLatencyNs uint64 `json:"commit_latency_ns"`
				ApplyLatencyNs  uint64 `json:"apply_latency_ns"`
			} `json:"perf_stats"`
		} `json:"osd_perf_infos"`
	} `json:"osdstats"`
}

func eventsMapping(content []byte) ([]common.MapStr, error) {
	var response OsdPerfResponse
	err := mgr.UnmarshalResponse(content, &response)
	if err != nil {
		return nil, errors.Wrap(err, "could not get response data")
	}

	var events []common.MapStr
	for _, OsdPerfInfo := range response.OsdStats.OsdPerfInfos {
		event := common.MapStr{
			"id": OsdPerfInfo.ID,
			"stats": common.MapStr{
				"commit_latency_ms": OsdPerfInfo.PerfStats.CommitLatencyMs,
				"apply_latency_ms":  OsdPerfInfo.PerfStats.ApplyLatencyMs,
				"commit_latency_ns": OsdPerfInfo.PerfStats.CommitLatencyNs,
				"apply_latency_ns":  OsdPerfInfo.PerfStats.ApplyLatencyNs,
			},
		}
		events = append(events, event)
	}
	return events, nil
}
