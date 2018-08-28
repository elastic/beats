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

// +build windows

package diskio

import (
	"github.com/elastic/beats/metricbeat/module/docker"
)

// BlkioService is a helper to collect and calculate disk I/O metrics
type BlkioService struct{}

// NewBlkioService builds a new initialized BlkioService
func NewBlkioService() *BlkioService {
	return &BlkioService{}
}

func (io *BlkioService) getBlkioStatsList(rawStats []docker.Stat, dedot bool) []BlkioStats {
	formattedStats := []BlkioStats{}

	for _, myRawStats := range rawStats {
		stats := BlkioStats{
			Time:      myRawStats.Stats.Read,
			Container: docker.NewContainer(myRawStats.Container, dedot),

			serviced: BlkioRaw{
				reads:  myRawStats.Stats.StorageStats.ReadCountNormalized,
				writes: myRawStats.Stats.StorageStats.WriteCountNormalized,
				totals: myRawStats.Stats.StorageStats.ReadCountNormalized + myRawStats.Stats.StorageStats.WriteCountNormalized,
			},

			servicedBytes: BlkioRaw{
				reads:  myRawStats.Stats.StorageStats.ReadSizeBytes,
				writes: myRawStats.Stats.StorageStats.WriteSizeBytes,
				totals: myRawStats.Stats.StorageStats.ReadSizeBytes + myRawStats.Stats.StorageStats.WriteSizeBytes,
			},
		}
		formattedStats = append(formattedStats, stats)
	}

	return formattedStats
}
