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
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common"
	s "github.com/menderesk/beats/v7/libbeat/common/schema"
	c "github.com/menderesk/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/menderesk/beats/v7/metricbeat/helper/elastic"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/module/logstash"
)

var (
	schema = s.Schema{
		"id":      c.Str("id"),
		"host":    c.Str("host"),
		"version": c.Str("version"),
		"jvm": c.Dict("jvm", s.Schema{
			"version": c.Str("version"),
			"pid":     c.Int("pid"),
		}),
	}
)

func commonFieldsMapping(event *mb.Event, fields common.MapStr) error {
	event.RootFields = common.MapStr{}
	event.RootFields.Put("service.name", logstash.ModuleName)

	// Set service ID
	serviceID, err := fields.GetValue("id")
	if err != nil {
		return elastic.MakeErrorForMissingField("id", elastic.Logstash)
	}
	event.RootFields.Put("service.id", serviceID)
	fields.Delete("id")

	// Set service hostname
	host, err := fields.GetValue("host")
	if err != nil {
		return elastic.MakeErrorForMissingField("host", elastic.Logstash)
	}
	event.RootFields.Put("service.hostname", host)
	fields.Delete("host")

	// Set service version
	version, err := fields.GetValue("version")
	if err != nil {
		return elastic.MakeErrorForMissingField("version", elastic.Logstash)
	}
	event.RootFields.Put("service.version", version)
	fields.Delete("version")

	// Set PID
	pid, err := fields.GetValue("jvm.pid")
	if err != nil {
		return elastic.MakeErrorForMissingField("jvm.pid", elastic.Logstash)
	}
	event.RootFields.Put("process.pid", pid)
	fields.Delete("jvm.pid")

	return nil
}

func eventMapping(r mb.ReporterV2, content []byte, pipelines []logstash.PipelineState, overrideClusterUUID string, isXpack bool) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Logstash Node API response")
	}

	fields, err := schema.Apply(data)
	if err != nil {
		return errors.Wrap(err, "failure applying node schema")
	}

	pipelines = getUserDefinedPipelines(pipelines)
	clusterToPipelinesMap := makeClusterToPipelinesMap(pipelines, overrideClusterUUID)

	for clusterUUID, pipelines := range clusterToPipelinesMap {
		for _, pipeline := range pipelines {
			removeClusterUUIDsFromPipeline(pipeline)

			// Rename key: graph -> representation
			pipeline.Representation = pipeline.Graph
			pipeline.Graph = nil

			logstashState := map[string]logstash.PipelineState{
				"pipeline": pipeline,
			}

			event := mb.Event{
				MetricSetFields: common.MapStr{
					"state": logstashState,
				},
				ModuleFields: common.MapStr{},
			}
			event.MetricSetFields.Update(fields)

			if err = commonFieldsMapping(&event, fields); err != nil {
				return err
			}

			if clusterUUID != "" {
				event.ModuleFields.Put("cluster.id", clusterUUID)
				event.ModuleFields.Put("elasticsearch.cluster.id", clusterUUID)
			}

			event.ID = pipeline.EphemeralID

			// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
			// When using Agent, the index name is overwritten anyways.
			if isXpack {
				index := elastic.MakeXPackMonitoringIndexName(elastic.Logstash)
				event.Index = index
			}

			r.Event(event)
		}
	}

	return nil
}

func makeClusterToPipelinesMap(pipelines []logstash.PipelineState, overrideClusterUUID string) map[string][]logstash.PipelineState {
	var clusterToPipelinesMap map[string][]logstash.PipelineState
	clusterToPipelinesMap = make(map[string][]logstash.PipelineState)

	if overrideClusterUUID != "" {
		clusterToPipelinesMap[overrideClusterUUID] = pipelines
		return clusterToPipelinesMap
	}

	for _, pipeline := range pipelines {
		clusterUUIDs := common.StringSet{}
		for _, vertex := range pipeline.Graph.Graph.Vertices {
			clusterUUID := logstash.GetVertexClusterUUID(vertex, overrideClusterUUID)
			if clusterUUID != "" {
				clusterUUIDs.Add(clusterUUID)
			}
		}

		// If no cluster UUID was found in this pipeline, assign it a blank one
		if len(clusterUUIDs) == 0 {
			clusterUUIDs.Add("")
		}

		for clusterUUID := range clusterUUIDs {
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
