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

func eventsMapping(memoryDataList []MemoryData) []common.MapStr {
	events := []common.MapStr{}
	for _, memoryData := range memoryDataList {
		events = append(events, eventMapping(&memoryData))
	}
	return events
}

func eventMapping(memoryData *MemoryData) common.MapStr {
	event := common.MapStr{
		mb.ModuleDataKey: common.MapStr{
			"container": memoryData.Container.ToMapStr(),
		},
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
	return event
}
