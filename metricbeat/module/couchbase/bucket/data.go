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

package bucket

import (
	"encoding/json"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type BucketQuota struct {
	RAM    int64 `json:"ram"`
	RawRAM int64 `json:"rawRAM"`
}

type BucketBasicStats struct {
	QuotaPercentUsed float64 `json:"quotaPercentUsed"`
	OpsPerSec        float64 `json:"opsPerSec"`
	DiskFetches      int64   `json:"diskFetches"`
	ItemCount        int64   `json:"itemCount"`
	DiskUsed         int64   `json:"diskUsed"`
	DataUsed         int64   `json:"dataUsed"`
	MemUsed          int64   `json:"memUsed"`
}

type Buckets []struct {
	Name       string           `json:"name"`
	BucketType string           `json:"bucketType"`
	Quota      BucketQuota      `json:"quota"`
	BasicStats BucketBasicStats `json:"basicStats"`
}

func eventsMapping(content []byte) []mapstr.M {
	var d Buckets
	err := json.Unmarshal(content, &d)
	if err != nil {
		logp.Err("Error: %+v", err)
	}

	events := []mapstr.M{}

	for _, Bucket := range d {
		event := mapstr.M{
			"name": Bucket.Name,
			"type": Bucket.BucketType,
			"data": mapstr.M{
				"used": mapstr.M{
					"bytes": Bucket.BasicStats.DataUsed,
				},
			},
			"disk": mapstr.M{
				"fetches": Bucket.BasicStats.DiskFetches,
				"used": mapstr.M{
					"bytes": Bucket.BasicStats.DiskUsed,
				},
			},
			"memory": mapstr.M{
				"used": mapstr.M{
					"bytes": Bucket.BasicStats.MemUsed,
				},
			},
			"quota": mapstr.M{
				"ram": mapstr.M{
					"bytes": Bucket.Quota.RAM,
				},
				"use": mapstr.M{
					"pct": Bucket.BasicStats.QuotaPercentUsed,
				},
			},
			"ops_per_sec": Bucket.BasicStats.OpsPerSec,
			"item_count":  Bucket.BasicStats.ItemCount,
		}

		events = append(events, event)
	}

	return events
}
