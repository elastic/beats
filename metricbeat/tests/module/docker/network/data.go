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

package network

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

func eventsMapping(r mb.ReporterV2, netsStatsList []NetStats) {
	for i := range netsStatsList {
		eventMapping(r, &netsStatsList[i])
	}
}

func eventMapping(r mb.ReporterV2, stats *NetStats) {
	r.Event(mb.Event{
		RootFields: stats.Container.ToMapStr(),
		MetricSetFields: common.MapStr{
			"interface": stats.NameInterface,
			// Deprecated
			"in": common.MapStr{
				"bytes":   stats.RxBytes,
				"dropped": stats.RxDropped,
				"errors":  stats.RxErrors,
				"packets": stats.RxPackets,
			},
			// Deprecated
			"out": common.MapStr{
				"bytes":   stats.TxBytes,
				"dropped": stats.TxDropped,
				"errors":  stats.TxErrors,
				"packets": stats.TxPackets,
			},
			"inbound": common.MapStr{
				"bytes":   stats.Total.RxBytes,
				"dropped": stats.Total.RxDropped,
				"errors":  stats.Total.RxErrors,
				"packets": stats.Total.RxPackets,
			},
			"outbound": common.MapStr{
				"bytes":   stats.Total.TxBytes,
				"dropped": stats.Total.TxDropped,
				"errors":  stats.Total.TxErrors,
				"packets": stats.Total.TxPackets,
			},
		},
	})
}
