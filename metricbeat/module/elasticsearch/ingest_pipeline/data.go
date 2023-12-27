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

package ingest_pipeline

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type Stats struct {
	ClusterName string               `json:"cluster_name"`
	Nodes       map[string]NodeStats `json:"nodes"`
}

type NodeStats struct {
	Name   string          `json:"name"`
	Roles  []string        `json:"roles"`
	Ingest NodeIngestStats `json:"ingest"`
}

type NodeIngestStats struct {
	Total     IngestStat              `json:"total"`
	Pipelines map[string]PipelineStat `json:"pipelines"`
}

type IngestStat struct {
	Count        int `json:"count"`
	TimeInMillis int `json:"time_in_millis"`
	Failed       int `json:"failed"`
}

type PipelineStat struct {
	IngestStat
	Processors []map[string]struct {
		Type  string     `json:"type"`
		Stats IngestStat `json:"stats"`
	} `json:"processors"`
}

func eventsMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte, isXpack bool, sampleProcessors bool) error {
	var nodeIngestStats Stats
	if err := json.Unmarshal(content, &nodeIngestStats); err != nil {
		return fmt.Errorf("failure parsing Node Ingest Stats API response: %w", err)
	}

	for nodeId, nodeStats := range nodeIngestStats.Nodes {
		// If there are no ingest stats on this node, don't create any events
		if nodeStats.Ingest.Total.Count == 0 && nodeStats.Ingest.Total.Failed == 0 && nodeStats.Ingest.Total.TimeInMillis == 0 {
			continue
		}

		for pipelineId, pipelineStats := range nodeStats.Ingest.Pipelines {
			// If there are no metrics on this node for this pipeline, don't create any events
			if pipelineStats.Count == 0 && pipelineStats.Failed == 0 && pipelineStats.TimeInMillis == 0 {
				continue
			}

			// Create the overall pipeline event
			event := mb.Event{
				ModuleFields:    mapstr.M{},
				MetricSetFields: mapstr.M{},
			}

			// Common fields
			addCommonFields(&event, &info, nodeId, &nodeStats, pipelineId)

			// Pipeline metrics
			event.MetricSetFields.Put("total.count", pipelineStats.Count)
			event.MetricSetFields.Put("total.failed", pipelineStats.Failed)
			event.MetricSetFields.Put("total.time.total.ms", pipelineStats.TimeInMillis)

			// Self time subtracts any processor pipelines
			selfCpuTime := pipelineStats.TimeInMillis
			for pIdx, processorObj := range pipelineStats.Processors {
				for pTypeTag, processorStats := range processorObj {
					if processorStats.Type == "pipeline" {
						selfCpuTime -= processorStats.Stats.TimeInMillis
					}

					// Skip creating the processor-level event when this fetch should not sample processors
					if !sampleProcessors {
						continue
					}

					// Create a processor event
					processorEvent := mb.Event{
						ModuleFields:    mapstr.M{},
						MetricSetFields: mapstr.M{},
					}

					// Common fields
					addCommonFields(&processorEvent, &info, nodeId, &nodeStats, pipelineId)

					// Processor metrics
					processorEvent.MetricSetFields.Put("processor.order_index", pIdx)
					processorEvent.MetricSetFields.Put("processor.type", processorStats.Type)
					processorEvent.MetricSetFields.Put("processor.type_tag", pTypeTag)
					processorEvent.MetricSetFields.Put("processor.count", processorStats.Stats.Count)
					processorEvent.MetricSetFields.Put("processor.failed", processorStats.Stats.Failed)
					processorEvent.MetricSetFields.Put("processor.time.total.ms", processorStats.Stats.TimeInMillis)
					r.Event(processorEvent)

					// processorObj has a single key with the processor type, so break early
					// Any other format would not be expected and would likely break dashboards
					break
				}
			}

			event.MetricSetFields.Put("total.time.self.ms", selfCpuTime)
			r.Event(event)
		}
	}

	return nil
}

func addCommonFields(event *mb.Event, info *elasticsearch.Info, nodeId string, nodeStats *NodeStats, pipelineId string) {
	event.ModuleFields.Put("cluster.id", info.ClusterID)
	event.ModuleFields.Put("cluster.name", info.ClusterName)
	event.ModuleFields.Put("node.id", nodeId)
	event.ModuleFields.Put("node.name", nodeStats.Name)
	event.ModuleFields.Put("node.roles", nodeStats.Roles)

	event.MetricSetFields.Put("name", pipelineId)
}
