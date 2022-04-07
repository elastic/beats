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

//go:build !integration
// +build !integration

package node

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v8/metricbeat/mb"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/metricbeat/module/logstash"
)

func TestEventMapping(t *testing.T) {

	files, err := filepath.Glob("./_meta/test/node.*.json")
	require.NoError(t, err)

	for _, f := range files {
		input, err := ioutil.ReadFile(f)
		require.NoError(t, err)

		var data map[string]interface{}
		err = json.Unmarshal(input, &data)
		require.NoError(t, err)

		event := mb.Event{}
		err = commonFieldsMapping(&event, data)
		require.NoError(t, err, f)
	}
}

func TestMakeClusterToPipelinesMap(t *testing.T) {
	tests := map[string]struct {
		pipelines           []logstash.PipelineState
		overrideClusterUUID string
		expectedMap         map[string][]logstash.PipelineState
	}{
		"no_vertex_cluster_id": {
			pipelines: []logstash.PipelineState{
				{
					ID: "test_pipeline",
					Graph: &logstash.GraphContainer{
						Graph: &logstash.Graph{
							Vertices: []map[string]interface{}{
								{
									"id": "vertex_1",
								},
								{
									"id": "vertex_2",
								},
								{
									"id": "vertex_3",
								},
							},
						},
					},
				},
			},
			overrideClusterUUID: "prod_cluster_id",
			expectedMap: map[string][]logstash.PipelineState{
				"prod_cluster_id": {
					{
						ID: "test_pipeline",
						Graph: &logstash.GraphContainer{
							Graph: &logstash.Graph{
								Vertices: []map[string]interface{}{
									{
										"id": "vertex_1",
									},
									{
										"id": "vertex_2",
									},
									{
										"id": "vertex_3",
									},
								},
							},
						},
					},
				},
			},
		},
		"one_vertex_cluster_id": {
			pipelines: []logstash.PipelineState{
				{
					ID: "test_pipeline",
					Graph: &logstash.GraphContainer{
						Graph: &logstash.Graph{
							Vertices: []map[string]interface{}{
								{
									"id":           "vertex_1",
									"cluster_uuid": "es_1",
								},
								{
									"id": "vertex_2",
								},
								{
									"id": "vertex_3",
								},
							},
						},
					},
				},
			},
			overrideClusterUUID: "prod_cluster_id",
			expectedMap: map[string][]logstash.PipelineState{
				"prod_cluster_id": {
					{
						ID: "test_pipeline",
						Graph: &logstash.GraphContainer{
							Graph: &logstash.Graph{
								Vertices: []map[string]interface{}{
									{
										"id":           "vertex_1",
										"cluster_uuid": "es_1",
									},
									{
										"id": "vertex_2",
									},
									{
										"id": "vertex_3",
									},
								},
							},
						},
					},
				},
			},
		},
		"two_pipelines": {
			pipelines: []logstash.PipelineState{
				{
					ID: "test_pipeline_1",
					Graph: &logstash.GraphContainer{
						Graph: &logstash.Graph{
							Vertices: []map[string]interface{}{
								{
									"id":           "vertex_1_1",
									"cluster_uuid": "es_1",
								},
								{
									"id": "vertex_1_2",
								},
								{
									"id": "vertex_1_3",
								},
							},
						},
					},
				},
				{
					ID: "test_pipeline_2",
					Graph: &logstash.GraphContainer{
						Graph: &logstash.Graph{
							Vertices: []map[string]interface{}{
								{
									"id": "vertex_2_1",
								},
								{
									"id": "vertex_2_2",
								},
								{
									"id": "vertex_2_3",
								},
							},
						},
					},
				},
			},
			overrideClusterUUID: "prod_cluster_id",
			expectedMap: map[string][]logstash.PipelineState{
				"prod_cluster_id": {
					{
						ID: "test_pipeline_1",
						Graph: &logstash.GraphContainer{
							Graph: &logstash.Graph{
								Vertices: []map[string]interface{}{
									{
										"id":           "vertex_1_1",
										"cluster_uuid": "es_1",
									},
									{
										"id": "vertex_1_2",
									},
									{
										"id": "vertex_1_3",
									},
								},
							},
						},
					},
					{
						ID: "test_pipeline_2",
						Graph: &logstash.GraphContainer{
							Graph: &logstash.Graph{
								Vertices: []map[string]interface{}{
									{
										"id": "vertex_2_1",
									},
									{
										"id": "vertex_2_2",
									},
									{
										"id": "vertex_2_3",
									},
								},
							},
						},
					},
				},
			},
		},
		"no_override_cluster_id": {
			pipelines: []logstash.PipelineState{
				{
					ID: "test_pipeline_1",
					Graph: &logstash.GraphContainer{
						Graph: &logstash.Graph{
							Vertices: []map[string]interface{}{
								{
									"id":           "vertex_1_1",
									"cluster_uuid": "es_1",
								},
								{
									"id":           "vertex_1_2",
									"cluster_uuid": "es_2",
								},
								{
									"id": "vertex_1_3",
								},
							},
						},
					},
				},
				{
					ID: "test_pipeline_2",
					Graph: &logstash.GraphContainer{
						Graph: &logstash.Graph{
							Vertices: []map[string]interface{}{
								{
									"id": "vertex_2_1",
								},
								{
									"id": "vertex_2_2",
								},
								{
									"id": "vertex_2_3",
								},
							},
						},
					},
				},
			},
			overrideClusterUUID: "",
			expectedMap: map[string][]logstash.PipelineState{
				"es_1": {
					{
						ID: "test_pipeline_1",
						Graph: &logstash.GraphContainer{
							Graph: &logstash.Graph{
								Vertices: []map[string]interface{}{
									{
										"id":           "vertex_1_1",
										"cluster_uuid": "es_1",
									},
									{
										"id":           "vertex_1_2",
										"cluster_uuid": "es_2",
									},
									{
										"id": "vertex_1_3",
									},
								},
							},
						},
					},
				},
				"es_2": {
					{
						ID: "test_pipeline_1",
						Graph: &logstash.GraphContainer{
							Graph: &logstash.Graph{
								Vertices: []map[string]interface{}{
									{
										"id":           "vertex_1_1",
										"cluster_uuid": "es_1",
									},
									{
										"id":           "vertex_1_2",
										"cluster_uuid": "es_2",
									},
									{
										"id": "vertex_1_3",
									},
								},
							},
						},
					},
				},
				"": {
					{
						ID: "test_pipeline_2",
						Graph: &logstash.GraphContainer{
							Graph: &logstash.Graph{
								Vertices: []map[string]interface{}{
									{
										"id": "vertex_2_1",
									},
									{
										"id": "vertex_2_2",
									},
									{
										"id": "vertex_2_3",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actualMap := makeClusterToPipelinesMap(test.pipelines, test.overrideClusterUUID)
			require.Equal(t, test.expectedMap, actualMap)
		})
	}
}
