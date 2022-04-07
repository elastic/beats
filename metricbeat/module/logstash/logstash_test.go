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

package logstash_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"
	"github.com/elastic/beats/v8/metricbeat/module/logstash"

	// Make sure metricsets are registered in mb.Registry
	_ "github.com/elastic/beats/v8/metricbeat/module/logstash/node"
	_ "github.com/elastic/beats/v8/metricbeat/module/logstash/node_stats"
)

func TestGetVertexClusterUUID(t *testing.T) {
	tests := map[string]struct {
		vertex              map[string]interface{}
		overrideClusterUUID string
		expectedClusterUUID string
	}{
		"vertex_and_override": {
			map[string]interface{}{
				"cluster_uuid": "v",
			},
			"o",
			"v",
		},
		"vertex_only": {
			vertex: map[string]interface{}{
				"cluster_uuid": "v",
			},
			expectedClusterUUID: "v",
		},
		"override_only": {
			overrideClusterUUID: "o",
			expectedClusterUUID: "o",
		},
		"none": {
			expectedClusterUUID: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.expectedClusterUUID, logstash.GetVertexClusterUUID(test.vertex, test.overrideClusterUUID))
		})
	}
}

func TestXPackEnabledMetricSets(t *testing.T) {
	config := map[string]interface{}{
		"module":        logstash.ModuleName,
		"hosts":         []string{"foobar:9600"},
		"xpack.enabled": true,
	}

	metricSets := mbtest.NewReportingMetricSetV2Errors(t, config)
	require.Len(t, metricSets, 2)
	for _, ms := range metricSets {
		name := ms.Name()
		switch name {
		case "node", "node_stats":
		default:
			t.Errorf("unexpected metricset name = %v", name)
		}
	}
}
