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

package logstash

import (
	"encoding/json"
	"net/url"

	"github.com/pkg/errors"

	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
)

// ModuleName is the name of this module.
const ModuleName = "logstash"

// MetricSet can be used to build other metricsets within the Logstash module.
type MetricSet struct {
	mb.BaseMetricSet
	XPack bool
}

// PipelineState represents the state (shape) of a Logstash pipeline
type PipelineState struct {
	ID             string                 `json:"id"`
	Hash           string                 `json:"hash"`
	EphemeralID    string                 `json:"ephemeral_id"`
	Graph          map[string]interface{} `json:"graph,omitempty"`
	Representation map[string]interface{} `json:"representation"`
	BatchSize      int                    `json:"batch_size"`
	Workers        int                    `json:"workers"`
	ClusterIDs     []string               `json:"cluster_uuids,omitempty"` // TODO: see https://github.com/elastic/logstash/issues/10602
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

// NewMetricSet creates a metricset that can be used to build other metricsets
// within the Logstash module.
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	config := struct {
		XPack bool `config:"xpack.enabled"`
	}{
		XPack: false,
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		base,
		config.XPack,
	}, nil
}

// GetPipelines returns the list of pipelines running on a Logstash node
func GetPipelines(http *helper.HTTP, resetURI string) ([]PipelineState, error) {
	content, err := fetchPath(http, resetURI, "_node/pipelines", "graph=true")
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch node pipelines")
	}

	pipelinesResponse := struct {
		Pipelines map[string]PipelineState `json:"pipelines"`
	}{}

	err = json.Unmarshal(content, &pipelinesResponse)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse node pipelines response")
	}

	var pipelines []PipelineState
	for pipelineID, pipeline := range pipelinesResponse.Pipelines {
		pipeline.ID = pipelineID
		pipelines = append(pipelines, pipeline)
	}

	return pipelines, nil
}

// GetPipelinesStats returns the list of pipelines (and their stats) running on a Logstash node
func GetPipelinesStats(http *helper.HTTP, resetURI string) ([]PipelineStats, error) {
	content, err := fetchPath(http, resetURI, "_node/stats", "vertices=true")
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch node pipeline stats")
	}

	pipelinesResponse := struct {
		Pipelines map[string]PipelineStats `json:"pipelines"`
	}{}

	err = json.Unmarshal(content, &pipelinesResponse)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse node pipeline stats response")
	}

	var pipelines []PipelineStats
	for pipelineID, pipeline := range pipelinesResponse.Pipelines {
		pipeline.ID = pipelineID
		pipelines = append(pipelines, pipeline)
	}

	return pipelines, nil
}

func fetchPath(http *helper.HTTP, resetURI, path string, query string) ([]byte, error) {
	defer http.SetURI(resetURI)

	// Parses the uri to replace the path
	u, _ := url.Parse(resetURI)
	u.Path = path
	u.RawQuery = query

	// Http helper includes the HostData with username and password
	http.SetURI(u.String())
	return http.FetchContent()
}
