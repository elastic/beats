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

package node_stats

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/menderesk/beats/v7/metricbeat/module/logstash"

	"github.com/stretchr/testify/require"

	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
)

func TestEventMapping(t *testing.T) {
	files, err := filepath.Glob("./_meta/test/node_stats.*.json")
	require.NoError(t, err)

	for _, f := range files {
		input, err := ioutil.ReadFile(f)
		require.NoError(t, err)

		reporter := &mbtest.CapturingReporterV2{}
		err = eventMapping(reporter, input, true)

		require.NoError(t, err, f)
		require.True(t, len(reporter.GetEvents()) >= 1, f)
		require.Equal(t, 0, len(reporter.GetErrors()), f)
	}
}

func TestData(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
		}

		input, _ := ioutil.ReadFile("./_meta/test/root.710.json")
		w.Write(input)
	}))

	mux.Handle("/_node/stats", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			input, _ := ioutil.ReadFile("./_meta/test/node_stats.710.json")
			w.Write(input)
		}))

	server := httptest.NewServer(mux)
	defer server.Close()

	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(server.URL))
	if err := mbtest.WriteEventsReporterV2Error(ms, t, ""); err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     logstash.ModuleName,
		"metricsets": []string{"node_stats"},
		"hosts":      []string{host},
	}
}

func TestMakeClusterToPipelinesMap(t *testing.T) {
	tests := map[string]struct {
		pipelines           []PipelineStats
		overrideClusterUUID string
		expectedMap         map[string][]PipelineStats
	}{
		"no_vertex_cluster_id": {
			pipelines: []PipelineStats{
				{
					ID: "test_pipeline",
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
			overrideClusterUUID: "prod_cluster_id",
			expectedMap: map[string][]PipelineStats{
				"prod_cluster_id": {
					{
						ID: "test_pipeline",
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
		"one_vertex_cluster_id": {
			pipelines: []PipelineStats{
				{
					ID: "test_pipeline",
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
			overrideClusterUUID: "prod_cluster_id",
			expectedMap: map[string][]PipelineStats{
				"prod_cluster_id": {
					{
						ID: "test_pipeline",
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
		"two_pipelines": {
			pipelines: []PipelineStats{
				{
					ID: "test_pipeline_1",
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
				{
					ID: "test_pipeline_2",
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
			overrideClusterUUID: "prod_cluster_id",
			expectedMap: map[string][]PipelineStats{
				"prod_cluster_id": {
					{
						ID: "test_pipeline_1",
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
					{
						ID: "test_pipeline_2",
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
		"no_override_cluster_id": {
			pipelines: []PipelineStats{
				{
					ID: "test_pipeline_1",
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
				{
					ID: "test_pipeline_2",
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
			expectedMap: map[string][]PipelineStats{
				"es_1": {
					{
						ID: "test_pipeline_1",
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
				"es_2": {
					{
						ID: "test_pipeline_1",
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
				"": {
					{
						ID: "test_pipeline_2",
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
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actualMap := makeClusterToPipelinesMap(test.pipelines, test.overrideClusterUUID)
			require.Equal(t, test.expectedMap, actualMap)
		})
	}
}
