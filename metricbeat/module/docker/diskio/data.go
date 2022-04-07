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

package diskio

import (
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

func eventsMapping(r mb.ReporterV2, blkioStatsList []BlkioStats) {
	for i := range blkioStatsList {
		eventMapping(r, &blkioStatsList[i])
	}
}

func eventMapping(r mb.ReporterV2, stats *BlkioStats) {
	fields := common.MapStr{
		"read": common.MapStr{
			"ops":          stats.serviced.reads,
			"bytes":        stats.servicedBytes.reads,
			"rate":         stats.reads,
			"service_time": stats.servicedTime.reads,
			"wait_time":    stats.waitTime.reads,
			"queued":       stats.queued.reads,
		},
		"write": common.MapStr{
			"ops":          stats.serviced.writes,
			"bytes":        stats.servicedBytes.writes,
			"rate":         stats.writes,
			"service_time": stats.servicedTime.writes,
			"wait_time":    stats.waitTime.writes,
			"queued":       stats.queued.writes,
		},
		"summary": common.MapStr{
			"ops":          stats.serviced.totals,
			"bytes":        stats.servicedBytes.totals,
			"rate":         stats.totals,
			"service_time": stats.servicedTime.totals,
			"wait_time":    stats.waitTime.totals,
			"queued":       stats.queued.totals,
		},
	}

	rootFields := stats.Container.ToMapStr()
	// Add container ECS fields
	_, _ = rootFields.Put("container.disk.read.bytes", stats.servicedBytes.reads)
	_, _ = rootFields.Put("container.disk.write.bytes", stats.servicedBytes.writes)

	r.Event(mb.Event{
		RootFields:      rootFields,
		MetricSetFields: fields,
	})
}
