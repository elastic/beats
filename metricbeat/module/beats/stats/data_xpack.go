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

package stats

import (
	"time"

	"github.com/elastic/beats/metricbeat/helper/elastic"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/beats"
)

func eventMappingXPack(r mb.ReporterV2, m *MetricSet, info beats.Info, content []byte) error {
	now := time.Now()
	clusterUUID := "TODO: Get from GET /state API response"

	fields := common.MapStr{
		"metrics":   "TODO: parse / construct from content",
		"beat":      "TODO: parse / construct from info",
		"timestamp": now,
	}

	var event mb.Event
	event.RootFields = common.MapStr{
		"cluster_uuid": clusterUUID,
		"timestamp":    now,
		"interval_ms":  m.calculateIntervalMs(),
		"type":         "beats_stats",
		"beats_stats":  fields,
	}

	event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Kibana)

	r.Event(event)
	return nil
}

func (m *MetricSet) calculateIntervalMs() int64 {
	return m.Module().Config().Period.Nanoseconds() / 1000 / 1000
}
