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

package elasticsearch_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"

	// Make sure metricsets are registered in mb.Registry
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/ccr"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/cluster_stats"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/enrich"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/index"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/index_recovery"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/index_summary"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/ml_job"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/node_stats"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/shard"
)

func TestXPackEnabledMetricsets(t *testing.T) {
	config := map[string]interface{}{
		"module":        elasticsearch.ModuleName,
		"hosts":         []string{"foobar:9200"},
		"xpack.enabled": true,
	}

	metricSets := mbtest.NewReportingMetricSetV2Errors(t, config)
	require.Len(t, metricSets, 9)
	for _, ms := range metricSets {
		name := ms.Name()
		switch name {
		case "ccr", "enrich", "cluster_stats", "index", "index_recovery",
			"index_summary", "ml_job", "node_stats", "shard":
		default:
			t.Errorf("unexpected metricset name = %v", name)
		}
	}
}
