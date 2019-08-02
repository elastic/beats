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
	pipelines = getUserDefinedPipelines(pipelines)
	clusterToPipelinesMap := makeClusterToPipelinesMap(pipelines)
	for clusterUUID, pipelines := range clusterToPipelinesMap {
		for _, pipeline := range pipelines {
			removeClusterUUIDsFromPipeline(pipeline)

			// Rename key: graph -> representation
			pipeline.Representation = pipeline.Graph
			pipeline.Graph = nil

			logstashState := map[string]logstash.PipelineState{
				"pipeline": pipeline,
			}

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

func makeClusterToPipelinesMap(pipelines []logstash.PipelineState) map[string][]logstash.PipelineState {
	var clusterToPipelinesMap map[string][]logstash.PipelineState
	clusterToPipelinesMap = make(map[string][]logstash.PipelineState)

	for _, pipeline := range pipelines {
		var clusterUUIDs []string
		for _, vertex := range pipeline.Graph.Graph.Vertices {
			c, ok := vertex["cluster_uuid"]
			if !ok {
				continue
			}

			clusterUUID, ok := c.(string)
			if !ok {
				continue
			}

			clusterUUIDs = append(clusterUUIDs, clusterUUID)
		}

		// If no cluster UUID was found in this pipeline, assign it a blank one
		if len(clusterUUIDs) == 0 {
			clusterUUIDs = []string{""}
		}

		for _, clusterUUID := range clusterUUIDs {
			clusterPipelines := clusterToPipelinesMap[clusterUUID]
			if clusterPipelines == nil {
				clusterToPipelinesMap[clusterUUID] = []logstash.PipelineState{}
			}

			clusterToPipelinesMap[clusterUUID] = append(clusterPipelines, pipeline)
		}
	}

	return clusterToPipelinesMap
}

func getUserDefinedPipelines(pipelines []logstash.PipelineState) []logstash.PipelineState {
	userDefinedPipelines := []logstash.PipelineState{}
	for _, pipeline := range pipelines {
		if pipeline.ID[0] != '.' {
			userDefinedPipelines = append(userDefinedPipelines, pipeline)
		}
	}
	return userDefinedPipelines
}

func removeClusterUUIDsFromPipeline(pipeline logstash.PipelineState) {
	for _, vertex := range pipeline.Graph.Graph.Vertices {
		_, exists := vertex["cluster_uuid"]
		if !exists {
			continue
		}

		delete(vertex, "cluster_uuid")
	}
}
