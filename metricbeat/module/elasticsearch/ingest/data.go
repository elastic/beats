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

package ingest

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/metricbeat/helper"
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

func eventsMapping(r mb.ReporterV2, httpClient *helper.HTTP, info elasticsearch.Info, content []byte, isXpack bool) error {
	var nodeIngestStats Stats
	if err := json.Unmarshal(content, &nodeIngestStats); err != nil {
		return errors.Wrap(err, "failure parsing Node Ingest Stats API response")
	}

	for nodeId, nodeStats := range nodeIngestStats.Nodes {
		if nodeStats.Ingest.Total.Count == 0 && nodeStats.Ingest.Total.Failed == 0 && nodeStats.Ingest.Total.TimeInMillis == 0 {
			continue
		}

		for pipelineId, pipelineStats := range nodeStats.Ingest.Pipelines {
			if pipelineStats.Count == 0 && pipelineStats.Failed == 0 && pipelineStats.TimeInMillis == 0 {
				continue
			}

			event := mb.Event{
				ModuleFields: mapstr.M{},
			}
			// Common fields
			// TODO: make more complete with Node Info API - cluster.id
			event.ModuleFields.Put("cluster.name", nodeIngestStats.ClusterName)
			event.ModuleFields.Put("node.id", nodeId)
			event.ModuleFields.Put("node.name", nodeStats.Name)
			event.ModuleFields.Put("node.roles", nodeStats.Roles)

			// Pipeline fields
			event.ModuleFields.Put("ingest.pipeline.name", pipelineId)
			event.ModuleFields.Put("ingest.pipeline.total.count", pipelineStats.Count)
			event.ModuleFields.Put("ingest.pipeline.total.failed", pipelineStats.Failed)
			event.ModuleFields.Put("ingest.pipeline.total.total_cpu_time", pipelineStats.TimeInMillis)

			selfCpuTime := pipelineStats.TimeInMillis
			for _, processorObj := range pipelineStats.Processors {
				// processorObj has a single key with the processor type
				for pType, processorStats := range processorObj {
					if pType == "pipeline" {
						selfCpuTime -= processorStats.Stats.TimeInMillis

						// TODO: add events for processors
					}
					break
				}
			}

			event.ModuleFields.Put("ingest.pipeline.total.self_cpu_time", selfCpuTime)
			r.Event(event)
		}
	}

	return nil
}
