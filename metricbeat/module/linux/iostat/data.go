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

package iostat

import (
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/metric/system/diskio"
)

// AddLinuxIOStat adds the linux iostat data to the provided map
func AddLinuxIOStat(extraMetrics diskio.IOMetric) common.MapStr {
	return common.MapStr{
		"read": common.MapStr{
			"request": common.MapStr{
				"merges_per_sec": extraMetrics.ReadRequestMergeCountPerSec,
				"per_sec":        extraMetrics.ReadRequestCountPerSec,
			},
			"per_sec": common.MapStr{
				"bytes": extraMetrics.ReadBytesPerSec,
			},
			"await": extraMetrics.AvgReadAwaitTime,
		},
		"write": common.MapStr{
			"request": common.MapStr{
				"merges_per_sec": extraMetrics.WriteRequestMergeCountPerSec,
				"per_sec":        extraMetrics.WriteRequestCountPerSec,
			},
			"per_sec": common.MapStr{
				"bytes": extraMetrics.WriteBytesPerSec,
			},
			"await": extraMetrics.AvgWriteAwaitTime,
		},
		"queue": common.MapStr{
			"avg_size": extraMetrics.AvgQueueSize,
		},
		"request": common.MapStr{
			"avg_size": extraMetrics.AvgRequestSize,
		},
		"await":        extraMetrics.AvgAwaitTime,
		"service_time": extraMetrics.AvgServiceTime,
		"busy":         extraMetrics.BusyPct,
	}
}
