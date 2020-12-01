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

// +build !integration

package node

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/module/logstash"
)

func TestMakeClusterToPipelinesMap(t *testing.T) {
	pipelines := []logstash.PipelineState{
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
	}
	m := makeClusterToPipelinesMap(pipelines, "prod_cluster_id")
	require.Len(t, m, 1)
	for clusterID, pipelines := range m {
		require.Equal(t, "prod_cluster_id", clusterID)
		require.Len(t, pipelines, 1)
	}
}
