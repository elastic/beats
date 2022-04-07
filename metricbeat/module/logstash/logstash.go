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
	"fmt"
	"net/url"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/metricbeat/helper"
	"github.com/elastic/beats/v8/metricbeat/helper/elastic"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

func init() {
	// Register the ModuleFactory function for this module.
	if err := mb.Registry.AddModule(ModuleName, NewModule); err != nil {
		panic(err)
	}
}

// NewModule creates a new module
func NewModule(base mb.BaseModule) (mb.Module, error) {
	return elastic.NewModule(&base, []string{"node", "node_stats"}, logp.NewLogger(ModuleName))
}

// ModuleName is the name of this module.
const ModuleName = "logstash"

// PipelineGraphAPIsAvailableVersion is the version of Logstash since when its APIs
// can return pipeline graphs
var PipelineGraphAPIsAvailableVersion = common.MustNewVersion("7.3.0")

// MetricSet can be used to build other metricsets within the Logstash module.
type MetricSet struct {
	mb.BaseMetricSet
	*helper.HTTP
	XPackEnabled bool
}

type Graph struct {
	Vertices []map[string]interface{} `json:"vertices"`
	Edges    []map[string]interface{} `json:"edges"`
}

type GraphContainer struct {
	Graph   *Graph `json:"graph,omitempty"`
	Type    string `json:"type"`
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

// PipelineState represents the state (shape) of a Logstash pipeline
type PipelineState struct {
	ID             string          `json:"id"`
	Hash           string          `json:"hash"`
	EphemeralID    string          `json:"ephemeral_id"`
	Graph          *GraphContainer `json:"graph,omitempty"`
	Representation *GraphContainer `json:"representation"`
	BatchSize      int             `json:"batch_size"`
	Workers        int             `json:"workers"`
}

// NewMetricSet creates a metricset that can be used to build other metricsets
// within the Logstash module.
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	config := struct {
		XPackEnabled bool `config:"xpack.enabled"`
	}{
		XPackEnabled: false,
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		base,
		http,
		config.XPackEnabled,
	}, nil
}

// GetPipelines returns the list of pipelines running on a Logstash node and,
// optionally, an override cluster UUID.
func GetPipelines(m *MetricSet) ([]PipelineState, string, error) {
	content, err := fetchPath(m.HTTP, "_node/pipelines", "graph=true")
	if err != nil {
		return nil, "", errors.Wrap(err, "could not fetch node pipelines")
	}

	pipelinesResponse := struct {
		Monitoring struct {
			ClusterID string `json:"cluster_uuid"`
		} `json:"monitoring"`
		Pipelines map[string]PipelineState `json:"pipelines"`
	}{}

	err = json.Unmarshal(content, &pipelinesResponse)
	if err != nil {
		return nil, "", errors.Wrap(err, "could not parse node pipelines response")
	}

	var pipelines []PipelineState
	for pipelineID, pipeline := range pipelinesResponse.Pipelines {
		pipeline.ID = pipelineID
		pipelines = append(pipelines, pipeline)
	}

	return pipelines, pipelinesResponse.Monitoring.ClusterID, nil
}

// CheckPipelineGraphAPIsAvailable returns an error if pipeline graph APIs are not
// available in the version of the Logstash node.
func (m *MetricSet) CheckPipelineGraphAPIsAvailable() error {
	logstashVersion, err := m.getVersion()
	if err != nil {
		return err
	}

	arePipelineGraphAPIsAvailable := elastic.IsFeatureAvailable(logstashVersion, PipelineGraphAPIsAvailableVersion)

	if !arePipelineGraphAPIsAvailable {
		const errorMsg = "the %v metricset with X-Pack enabled is only supported with Logstash >= %v. You are currently running Logstash %v"
		return fmt.Errorf(errorMsg, m.FullyQualifiedName(), PipelineGraphAPIsAvailableVersion, logstashVersion)
	}

	return nil
}

// GetVertexClusterUUID returns the correct cluster UUID value for the given Elasticsearch
// vertex from a Logstash pipeline. If the vertex has no cluster UUID associated with it,
// the given override cluster UUID is returned.
func GetVertexClusterUUID(vertex map[string]interface{}, overrideClusterUUID string) string {
	c, ok := vertex["cluster_uuid"]
	if !ok {
		return overrideClusterUUID
	}

	clusterUUID, ok := c.(string)
	if !ok {
		return overrideClusterUUID
	}

	if clusterUUID == "" {
		return overrideClusterUUID
	}

	return clusterUUID
}

func (m *MetricSet) getVersion() (*common.Version, error) {
	const rootPath = "/"
	content, err := fetchPath(m.HTTP, rootPath, "")
	if err != nil {
		return nil, err
	}

	var response struct {
		Version *common.Version `json:"version"`
	}

	err = json.Unmarshal(content, &response)
	if err != nil {
		return nil, err
	}

	return response.Version, nil
}

func fetchPath(httpHelper *helper.HTTP, path string, query string) ([]byte, error) {
	currentURI := httpHelper.GetURI()
	defer httpHelper.SetURI(currentURI)

	// Parses the uri to replace the path
	u, _ := url.Parse(currentURI)
	u.Path = path
	u.RawQuery = query

	// Http helper includes the HostData with username and password
	httpHelper.SetURI(u.String())
	return httpHelper.FetchContent()
}
