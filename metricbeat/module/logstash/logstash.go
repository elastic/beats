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

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	// Register the ModuleFactory function for this module.
	if err := mb.Registry.AddModule(ModuleName, NewModule); err != nil {
		panic(err)
	}
}

// NewModule creates a new module after performing validation.
func NewModule(base mb.BaseModule) (mb.Module, error) {
	if err := validateXPackMetricsets(base); err != nil {
		return nil, err
	}

	return &base, nil
}

// Validate that correct metricsets have been specified if xpack.enabled = true.
func validateXPackMetricsets(base mb.BaseModule) error {
	config := struct {
		Metricsets   []string `config:"metricsets"`
		XPackEnabled bool     `config:"xpack.enabled"`
	}{}
	if err := base.UnpackConfig(&config); err != nil {
		return err
	}

	// Nothing to validate if xpack.enabled != true
	if !config.XPackEnabled {
		return nil
	}

	expectedXPackMetricsets := []string{
		"node",
		"node_stats",
	}

	if !common.MakeStringSet(config.Metricsets...).Equals(common.MakeStringSet(expectedXPackMetricsets...)) {
		return errors.Errorf("The %v module with xpack.enabled: true must have metricsets: %v", ModuleName, expectedXPackMetricsets)
	}

	return nil
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
	XPack bool
}

type graph struct {
	Vertices []map[string]interface{} `json:"vertices"`
	Edges    []map[string]interface{} `json:"edges"`
}

type graphContainer struct {
	Graph   *graph `json:"graph,omitempty"`
	Type    string `json:"type"`
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

// PipelineState represents the state (shape) of a Logstash pipeline
type PipelineState struct {
	ID             string          `json:"id"`
	Hash           string          `json:"hash"`
	EphemeralID    string          `json:"ephemeral_id"`
	Graph          *graphContainer `json:"graph,omitempty"`
	Representation *graphContainer `json:"representation"`
	BatchSize      int             `json:"batch_size"`
	Workers        int             `json:"workers"`
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

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		base,
		http,
		config.XPack,
	}, nil
}

// GetPipelines returns the list of pipelines running on a Logstash node
func GetPipelines(m *MetricSet) ([]PipelineState, error) {
	content, err := fetchPath(m.HTTP, "_node/pipelines", "graph=true")
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

// GetVersion returns the version of the Logstash node
func GetVersion(m *MetricSet) (*common.Version, error) {
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

// ArePipelineGraphAPIsAvailable returns whether Logstash APIs that returns pipeline graphs
// are available in the given version of Logstash
func ArePipelineGraphAPIsAvailable(currentLogstashVersion *common.Version) bool {
	return elastic.IsFeatureAvailable(currentLogstashVersion, PipelineGraphAPIsAvailableVersion)
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
