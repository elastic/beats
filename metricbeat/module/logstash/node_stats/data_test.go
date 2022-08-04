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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/beats/v7/metricbeat/module/logstash"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestEventMapping(t *testing.T) {
	// Contain pipeline hash
	containVersions := []string{}
	containVersions = append(containVersions, "710")
	containVersions = append(containVersions, "840")
	EventMappingForFiles(t, containVersions, 1, 0)
	// Don't contain pipeline hash
	dontContainVersions := []string{}
	dontContainVersions = append(dontContainVersions, "641")
	dontContainVersions = append(dontContainVersions, "650")
	dontContainVersions = append(dontContainVersions, "700")
	EventMappingForFiles(t, dontContainVersions, 1, 0)
	// Don't contain pipeline hash but should (partial)
	partialVersions := []string{}
	partialVersions = append(partialVersions, "840_partial")
	EventMappingForFiles(t, partialVersions, 0, 0)
}

func TestPipelineMarshal(t *testing.T) {
	logger := logp.NewLogger("logstash.node_stats")

	path := "./_meta/test/node_stats.710.json"
	input, err := ioutil.ReadFile(path)
	require.NoError(t, err, "error reading file %s", path)

	reporter := &mbtest.CapturingReporterV2{}
	err = eventMapping(reporter, input, true, logger)
	require.NoError(t, err, "error in event mapping for file %s", path)

	events := reporter.GetEvents()
	nodeStats := events[0].ModuleFields["node"].(mapstr.M)["stats"]
	pipeline := nodeStats.(LogstashStats).Pipelines[0]
	t.Logf("Event: %#v", pipeline)

	assert.Equal(t, "main", pipeline.ID)
	assert.Equal(t, int64(0), pipeline.Events["filtered"])
	assert.Equal(t, int64(5), pipeline.Events["duration_in_millis"])
	assert.Equal(t, "memory", pipeline.Queue.Type)
	assert.Equal(t, int64(100), pipeline.Queue.EventsCount)
	assert.Equal(t, int64(0), pipeline.Queue.QueueSizeInBytes)

	assert.Len(t, pipeline.Vertices, 2)
	assert.Equal(t, "0710cad67e8f47667bc7612580d5b91f691dd8262a4187d9eca8cf87229d04aa", pipeline.Vertices[0].ID)
	assert.Equal(t, "f4944472678ac54e7343c1a49748c402b0bafd76ebab7fe2f3930269e0e5097b", pipeline.Vertices[1].ID)
	assert.Equal(t, int64(2), pipeline.Vertices[0].QueuePushDurationInMillis)
	assert.Equal(t, int64(20), pipeline.Vertices[1].DurationInMillis)

}

func EventMappingForFiles(t *testing.T, fixtureVersions []string, expectedEvents int, expectedErrors int) {
	logger := logp.NewLogger("logstash.node_stats")

	for _, f := range fixtureVersions {
		path := fmt.Sprintf("./_meta/test/node_stats.%s.json", f)
		input, err := ioutil.ReadFile(path)
		require.NoError(t, err)

		reporter := &mbtest.CapturingReporterV2{}
		err = eventMapping(reporter, input, true, logger)
		require.NoError(t, err, "error in event mapping for file %s", path)
		require.True(t, len(reporter.GetEvents()) >= expectedEvents, f)
		require.Equal(t, expectedErrors, len(reporter.GetErrors()), f)
	}
}

func TestData(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
		}

		input, _ := ioutil.ReadFile("./_meta/test/root.710.json")
		_, err := w.Write(input)
		require.NoError(t, err, "error writing file in / for TestData")
	}))

	mux.Handle("/_node/stats", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			input, _ := ioutil.ReadFile("./_meta/test/node_stats.710.json")
			_, err := w.Write(input)
			require.NoError(t, err, "error writing file in /_node/stats for TestData")
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
					Vertices: []logstash.Vertex{
						{
							ID: "vertex_1",
						},
						{
							ID: "vertex_2",
						},
						{
							ID: "vertex_3",
						},
					},
				},
			},
			overrideClusterUUID: "prod_cluster_id",
			expectedMap: map[string][]PipelineStats{
				"prod_cluster_id": {
					{
						ID: "test_pipeline",
						Vertices: []logstash.Vertex{
							{
								ID: "vertex_1",
							},
							{
								ID: "vertex_2",
							},
							{
								ID: "vertex_3",
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
					Vertices: []logstash.Vertex{
						{
							ID:          "vertex_1",
							ClusterUUID: "es_1",
						},
						{
							ID: "vertex_2",
						},
						{
							ID: "vertex_3",
						},
					},
				},
			},
			overrideClusterUUID: "prod_cluster_id",
			expectedMap: map[string][]PipelineStats{
				"prod_cluster_id": {
					{
						ID: "test_pipeline",
						Vertices: []logstash.Vertex{
							{
								ID:          "vertex_1",
								ClusterUUID: "es_1",
							},
							{
								ID: "vertex_2",
							},
							{
								ID: "vertex_3",
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
					Vertices: []logstash.Vertex{
						{
							ID:          "vertex_1_1",
							ClusterUUID: "es_1",
						},
						{
							ID: "vertex_1_2",
						},
						{
							ID: "vertex_1_3",
						},
					},
				},
				{
					ID: "test_pipeline_2",
					Vertices: []logstash.Vertex{
						{
							ID: "vertex_2_1",
						},
						{
							ID: "vertex_2_2",
						},
						{
							ID: "vertex_2_3",
						},
					},
				},
			},
			overrideClusterUUID: "prod_cluster_id",
			expectedMap: map[string][]PipelineStats{
				"prod_cluster_id": {
					{
						ID: "test_pipeline_1",
						Vertices: []logstash.Vertex{
							{
								ID:          "vertex_1_1",
								ClusterUUID: "es_1",
							},
							{
								ID: "vertex_1_2",
							},
							{
								ID: "vertex_1_3",
							},
						},
					},
					{
						ID: "test_pipeline_2",
						Vertices: []logstash.Vertex{
							{
								ID: "vertex_2_1",
							},
							{
								ID: "vertex_2_2",
							},
							{
								ID: "vertex_2_3",
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
					Vertices: []logstash.Vertex{
						{
							ID:          "vertex_1_1",
							ClusterUUID: "es_1",
						},
						{
							ID:          "vertex_1_2",
							ClusterUUID: "es_2",
						},
						{
							ID: "vertex_1_3",
						},
					},
				},
				{
					ID: "test_pipeline_2",
					Vertices: []logstash.Vertex{
						{
							ID: "vertex_2_1",
						},
						{
							ID: "vertex_2_2",
						},
						{
							ID: "vertex_2_3",
						},
					},
				},
			},
			expectedMap: map[string][]PipelineStats{
				"es_1": {
					{
						ID: "test_pipeline_1",
						Vertices: []logstash.Vertex{
							{
								ID:          "vertex_1_1",
								ClusterUUID: "es_1",
							},
							{
								ID:          "vertex_1_2",
								ClusterUUID: "es_2",
							},
							{
								ID: "vertex_1_3",
							},
						},
					},
				},
				"es_2": {
					{
						ID: "test_pipeline_1",
						Vertices: []logstash.Vertex{
							{
								ID:          "vertex_1_1",
								ClusterUUID: "es_1",
							},
							{
								ID:          "vertex_1_2",
								ClusterUUID: "es_2",
							},
							{
								ID: "vertex_1_3",
							},
						},
					},
				},
				"": {
					{
						ID: "test_pipeline_2",
						Vertices: []logstash.Vertex{
							{
								ID: "vertex_2_1",
							},
							{
								ID: "vertex_2_2",
							},
							{
								ID: "vertex_2_3",
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
