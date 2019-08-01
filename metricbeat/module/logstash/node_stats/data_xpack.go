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

type jvm struct {
	GC  map[string]interface{} `json:"gc"`
	Mem struct {
		HeapMaxInBytes  int `json:"heap_max_in_bytes"`
		HeapUsedInBytes int `json:"heap_used_in_bytes"`
		HeapUsedPercent int `json:"heap_used_percent"`
	} `json:"mem"`
	UptimeInMillis int `json:"uptime_in_millis"`
}

type events struct {
	DurationInMillis int `json:"duration_in_millis"`
	In               int `json:"in"`
	Filtered         int `json:"filtered"`
	Out              int `json:"out"`
}

type commonStats struct {
	Events  events                 `json:"events"`
	JVM     jvm                    `json:"jvm"`
	Reloads map[string]interface{} `json:"reloads"`
	Queue   struct {
		EventsCount int `json:"events_count"`
	} `json:"queue"`
}

type cpu struct {
	Percent     int                    `json:"percent,omitempty"`
	LoadAverage map[string]interface{} `json:"load_average,omitempty"`
}

type process struct {
	OpenFileDescriptors int `json:"open_file_descriptors"`
	MaxFileDescriptors  int `json:"max_file_descriptors"`
	CPU                 cpu `json:"cpu"`
}

type cgroup struct {
	CPUAcct map[string]interface{} `json:"cpuacct"`
	CPU     struct {
		Stat         map[string]interface{} `json:"stat"`
		ControlGroup string                 `json:"control_group"`
	} `json:"cpu"`
}

type os struct {
	CPU    cpu    `json:"cpu"`
	CGroup cgroup `json:"cgroup,omitempty"`
}

type pipeline struct {
	BatchSize int `json:"batch_size"`
	Workers   int `json:"workers"`
}

type nodeInfo struct {
	ID          string   `json:"id,omitempty"`
	UUID        string   `json:"uuid"`
	EphemeralID string   `json:"ephemeral_id"`
	Name        string   `json:"name"`
	Host        string   `json:"host"`
	Version     string   `json:"version"`
	Snapshot    bool     `json:"snapshot"`
	Status      string   `json:"status"`
	HTTPAddress string   `json:"http_address"`
	Pipeline    pipeline `json:"pipeline"`
}

type reloads struct {
	Successes int `json:"successes"`
	Failures  int `json:"failures"`
}

// NodeStats represents the stats of a Logstash node
type NodeStats struct {
	nodeInfo
	commonStats
	Process   process                  `json:"process"`
	OS        os                       `json:"os"`
	Pipelines map[string]PipelineStats `json:"pipelines"`
}

// LogstashStats represents the logstash_stats sub-document indexed into .monitoring-logstash-*
type LogstashStats struct {
	commonStats
	Process   process         `json:"process"`
	OS        os              `json:"os"`
	Pipelines []PipelineStats `json:"pipelines"`
	Logstash  nodeInfo        `json:"logstash"`
	Timestamp common.Time     `json:"timestamp"`
}

// PipelineStats represents the stats of a Logstash pipeline
type PipelineStats struct {
	ID          string                   `json:"id"`
	Hash        string                   `json:"hash"`
	EphemeralID string                   `json:"ephemeral_id"`
	Events      map[string]interface{}   `json:"events"`
	Reloads     reloads                  `json:"reloads"`
	Queue       map[string]interface{}   `json:"queue"`
	Vertices    []map[string]interface{} `json:"vertices"`
}

func eventMappingXPack(r mb.ReporterV2, m *MetricSet, content []byte) error {
	var nodeStats NodeStats
	err := json.Unmarshal(content, &nodeStats)
	if err != nil {
		return errors.Wrap(err, "could not parse node stats response")
	}

	timestamp := common.Time(time.Now())

	// Massage Logstash node basic info
	nodeStats.nodeInfo.UUID = nodeStats.nodeInfo.ID
	nodeStats.nodeInfo.ID = ""

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
		},
		nodeStats.OS.CGroup,
	}

	var pipelines []PipelineStats
	for pipelineID, pipeline := range nodeStats.Pipelines {
		pipeline.ID = pipelineID
		pipelines = append(pipelines, pipeline)
	}

	pipelines = getUserDefinedPipelines(pipelines)
	clusterToPipelinesMap := makeClusterToPipelinesMap(pipelines)

	for clusterUUID, clusterPipelines := range clusterToPipelinesMap {
		logstashStats := LogstashStats{
			nodeStats.commonStats,
			proc,
			o,
			clusterPipelines,
			nodeStats.nodeInfo,
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
	clusterToPipelinesMap = make(map[string][]PipelineStats)

	for _, pipeline := range pipelines {
		var clusterUUIDs []string
		for _, vertex := range pipeline.Vertices {
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
				clusterToPipelinesMap[clusterUUID] = []PipelineStats{}
			}

			clusterToPipelinesMap[clusterUUID] = append(clusterPipelines, pipeline)
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
