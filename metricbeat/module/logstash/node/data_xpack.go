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

package node

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/logstash"
)

func eventMappingXPack(r mb.ReporterV2, m *MetricSet, pipelines []logstash.PipelineState) error {
	for _, pipeline := range pipelines {
		// Exclude internal pipelines
		if pipeline.ID[0] == '.' {
			continue
		}

		// Rename key: graph -> representation
		pipeline.Representation = pipeline.Graph
		pipeline.Graph = nil

		// Extract cluster_uuids
		clusterUUIDs := pipeline.ClusterIDs
		pipeline.ClusterIDs = nil

		logstashState := map[string]logstash.PipelineState{
			"pipeline": pipeline,
		}

		if pipeline.ClusterIDs == nil {
			pipeline.ClusterIDs = []string{""}
		}

		for _, clusterUUID := range clusterUUIDs {
			event := mb.Event{}
			event.RootFields = common.MapStr{
				"timestamp":      common.Time(time.Now()),
				"interval_ms":    m.Module().Config().Period / time.Millisecond,
				"type":           "logstash_state",
				"logstash_state": logstashState,
			}

			if clusterUUID != "" {
				event.RootFields["cluster_uuid"] = clusterUUID
			}

			event.ID = pipeline.EphemeralID
			event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Logstash)
			r.Event(event)
		}
	}

	return nil
}
