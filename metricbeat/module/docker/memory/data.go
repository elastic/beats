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

package memory

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

func eventsMapping(r mb.ReporterV2, memoryDataList []MemoryData) {
	for _, memoryData := range memoryDataList {
		eventMapping(r, &memoryData)
	}
}

func eventMapping(r mb.ReporterV2, memoryData *MemoryData) {

	//if we have windows memory data, just report windows stats
	var fields common.MapStr
	if memoryData.Commit+memoryData.CommitPeak+memoryData.PrivateWorkingSet > 0 {
		fields = common.MapStr{
			"commit": common.MapStr{
				"total": memoryData.Commit,
				"peak":  memoryData.CommitPeak,
			},
			"private_working_set": common.MapStr{
				"total": memoryData.PrivateWorkingSet,
			},
		}
	} else {
		fields = common.MapStr{
			"stats": memoryData.Stats,
			"fail": common.MapStr{
				"count": memoryData.Failcnt,
			},
			"limit": memoryData.Limit,
			"rss": common.MapStr{
				"total": memoryData.TotalRss,
				"pct":   memoryData.TotalRssP,
			},
			"usage": common.MapStr{
				"total": memoryData.Usage,
				"pct":   memoryData.UsageP,
				"max":   memoryData.MaxUsage,
			},
		}
	}

	r.Event(mb.Event{
		RootFields:      memoryData.Container.ToMapStr(),
		MetricSetFields: fields,
	})
}
