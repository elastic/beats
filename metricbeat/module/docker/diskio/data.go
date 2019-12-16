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
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

func eventsMapping(r mb.ReporterV2, blkioStatsList []BlkioStats) {
	for _, blkioStats := range blkioStatsList {
		eventMapping(r, &blkioStats)
	}
}

func eventMapping(r mb.ReporterV2, stats *BlkioStats) {
	fields := common.MapStr{
		"reads":  stats.reads,
		"writes": stats.writes,
		"total":  stats.totals,
		"read": common.MapStr{
			"ops":   stats.serviced.reads,
			"bytes": stats.servicedBytes.reads,
			"rate":  stats.reads,
		},
		"write": common.MapStr{
			"ops":   stats.serviced.writes,
			"bytes": stats.servicedBytes.writes,
			"rate":  stats.writes,
		},
		"summary": common.MapStr{
			"ops":   stats.serviced.totals,
			"bytes": stats.servicedBytes.totals,
			"rate":  stats.totals,
		},
	}

	r.Event(mb.Event{
		RootFields:      stats.Container.ToMapStr(),
		MetricSetFields: fields,
	})
}
