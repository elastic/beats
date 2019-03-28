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

package node_stats

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
)

type commonStats struct {
	Events  map[string]interface{} `json:"events"`
	JVM     map[string]interface{} `json:"jvm"`
	Reloads map[string]interface{} `json:"reloads"`
}

type cpu struct {
	Percent     int                    `json:"percent,omitempty"`
	LoadAverage map[string]interface{} `json:"load_average,omitempty"`
	NumCPUs     int                    `json:"num_cpus,omitempty"`
}

type process struct {
	OpenFileDescriptors int `json:"open_file_descriptors"`
	MaxFileDescriptors  int `json:"max_file_descriptors"`
	CPU                 cpu `json:"cpu"`
}

type os struct {
	CPU cpu `json:"cpu"`
}

// NodeStats represents the stats of a Logstash node
type NodeStats struct {
	commonStats
	Process   process                  `json:"process"`
	Pipelines map[string]PipelineStats `json:"pipelines"`
}

// LogstashStats represents the logstash_stats sub-document indexed into .monitoring-logstash-*
type LogstashStats struct {
	commonStats
	Process   process                `json:"process"`
	OS        os                     `json:"os"`
	Pipelines []PipelineStats        `json:"pipelines"`
	Logstash  map[string]interface{} `json:"logstash"`
	Queue     map[string]interface{} `json:"queue"`
	Timestamp common.Time            `json:"timestamp"`
}

// PipelineStats represents the stats of a Logstash pipeline
type PipelineStats struct {
	ID          string                   `json:"id"`
	Hash        string                   `json:"hash"`
	EphemeralID string                   `json:"ephemeral_id"`
	Events      map[string]interface{}   `json:"events"`
	Reloads     map[string]interface{}   `json:"reloads"`
	Queue       map[string]interface{}   `json:"queue"`
	Vertices    []map[string]interface{} `json:"vertices"`
	ClusterIDs  []string                 `json:"cluster_uuids,omitempty"` // TODO: see https://github.com/elastic/logstash/issues/10602
}

func eventMappingXPack(r mb.ReporterV2, m *MetricSet, content []byte) error {
	var nodeStats NodeStats
	err := json.Unmarshal(content, &nodeStats)
	if err != nil {
		return errors.Wrap(err, "could not parse node stats response")
	}

	var pipelines []PipelineStats
	for pipelineID, pipeline := range nodeStats.Pipelines {
		pipeline.ID = pipelineID
		pipelines = append(pipelines, pipeline)
	}

	pipelines = getUserDefinedPipelines(pipelines)
	clusterToPipelinesMap := makeClusterToPipelinesMap(pipelines)

	for clusterUUID, clusterPipelines := range clusterToPipelinesMap {
		timestamp := common.Time(time.Now())
		proc := process{
			nodeStats.Process.OpenFileDescriptors,
			nodeStats.Process.MaxFileDescriptors,
			cpu{
				Percent: nodeStats.Process.CPU.Percent,
			},
		}
		o := os{
			cpu{
				LoadAverage: nodeStats.Process.CPU.LoadAverage,
				NumCPUs:     nodeStats.Process.CPU.NumCPUs,
			},
		}
		logstash := map[string]interface{}{} // TODO; see https://github.com/elastic/logstash/issues/10121
		queue := map[string]interface{}{}    // TODO: see https://github.com/elastic/logstash/issues/10610

		logstashStats := LogstashStats{
			nodeStats.commonStats,
			proc,
			o,
			clusterPipelines,
			logstash,
			queue,
			timestamp,
		}

		event := mb.Event{}
		event.RootFields = common.MapStr{
			"timestamp":      timestamp,
			"interval_ms":    m.Module().Config().Period / time.Millisecond,
			"type":           "logstash_stats",
			"logstash_stats": logstashStats,
		}

		if clusterUUID != "" {
			event.RootFields["cluster_uuid"] = clusterUUID
		}

		event.Index = elastic.MakeXPackMonitoringIndexName(elastic.Logstash)
		r.Event(event)
	}

	return nil
}

func makeClusterToPipelinesMap(pipelines []PipelineStats) map[string][]PipelineStats {
	var clusterToPipelinesMap map[string][]PipelineStats

	for _, pipeline := range pipelines {
		clusterUUIDs := pipeline.ClusterIDs
		if clusterUUIDs == nil {
			clusterUUIDs = []string{""}
		}

		for _, clusterUUID := range clusterUUIDs {
			clusterPipelines := clusterToPipelinesMap[clusterUUID]
			if clusterPipelines == nil {
				clusterPipelines = []PipelineStats{}
			}

			clusterPipelines = append(clusterPipelines, pipeline)
		}
	}

	return clusterToPipelinesMap
}

func getUserDefinedPipelines(pipelines []PipelineStats) []PipelineStats {
	userDefinedPipelines := []PipelineStats{}
	for _, pipeline := range pipelines {
		if pipeline.ID[0] != '.' {
			userDefinedPipelines = append(userDefinedPipelines, pipeline)
		}
	}
	return userDefinedPipelines
}
