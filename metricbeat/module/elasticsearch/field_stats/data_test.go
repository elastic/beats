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

package field_stats

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

var info = elasticsearch.Info{
	ClusterID:   "1234",
	ClusterName: "helloworld",
}

func TestEmptyResponseShouldGiveNoError(t *testing.T) {
	content, err := os.ReadFile("./_meta/test/empty.json")
	require.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	err = eventsMapping(reporter, info, content, true)
	require.NoError(t, err)
	require.Empty(t, reporter.GetEvents())
	require.Empty(t, reporter.GetErrors())
}

func TestSingleIndexShouldEmitFieldEvents(t *testing.T) {
	content, err := os.ReadFile("./_meta/test/field_usage_stats.json")
	require.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	err = eventsMapping(reporter, info, content, false)
	require.NoError(t, err)

	events := reporter.GetEvents()
	// 1 index, 1 shard, 2 fields = 2 events
	require.Equal(t, 2, len(events))
	require.Empty(t, reporter.GetErrors())

	// Verify common fields on first event
	ev := events[0]
	serviceName, _ := ev.RootFields.GetValue("service.name")
	require.Equal(t, "elasticsearch", serviceName)

	clusterName, _ := ev.ModuleFields.GetValue("cluster.name")
	require.Equal(t, "helloworld", clusterName)

	clusterID, _ := ev.ModuleFields.GetValue("cluster.id")
	require.Equal(t, "1234", clusterID)

	indexName, _ := ev.ModuleFields.GetValue("index.name")
	require.Equal(t, "my-index-000001", indexName)

	// Verify field name is present
	fieldName, _ := ev.MetricSetFields.GetValue("name")
	require.NotEmpty(t, fieldName)

	// Verify shard routing info is present
	shardNode, _ := ev.MetricSetFields.GetValue("shard.routing.node")
	require.Equal(t, "node-1", shardNode)

	shardPrimary, _ := ev.MetricSetFields.GetValue("shard.routing.primary")
	require.Equal(t, true, shardPrimary)
}

func TestMultiIndexShouldEmitEventsForEachIndex(t *testing.T) {
	content, err := os.ReadFile("./_meta/test/multi_index.json")
	require.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	err = eventsMapping(reporter, info, content, false)
	require.NoError(t, err)

	events := reporter.GetEvents()
	// 2 indices, 1 shard each, 1 field each = 2 events
	require.Equal(t, 2, len(events))
	require.Empty(t, reporter.GetErrors())
}

func TestXpackEnabledSetsMonitoringIndex(t *testing.T) {
	content, err := os.ReadFile("./_meta/test/field_usage_stats.json")
	require.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	err = eventsMapping(reporter, info, content, true)
	require.NoError(t, err)

	events := reporter.GetEvents()
	require.NotEmpty(t, events)
	for _, ev := range events {
		require.NotEmpty(t, ev.Index)
	}
}

func TestInvalidJsonShouldReturnError(t *testing.T) {
	content := []byte(`{invalid json}`)

	reporter := &mbtest.CapturingReporterV2{}
	err := eventsMapping(reporter, info, content, false)
	require.Error(t, err)
}

func TestFieldStatsValues(t *testing.T) {
	content, err := os.ReadFile("./_meta/test/field_usage_stats.json")
	require.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	err = eventsMapping(reporter, info, content, false)
	require.NoError(t, err)

	events := reporter.GetEvents()
	require.Equal(t, 2, len(events))

	// Find the "message" field event
	var messageEvent *mb.Event
	for i, ev := range events {
		name, _ := ev.MetricSetFields.GetValue("name")
		if name == "message" {
			messageEvent = &events[i]
			break
		}
	}
	require.NotNil(t, messageEvent, "expected event for field 'message'")

	any, _ := messageEvent.MetricSetFields.GetValue("any")
	require.Equal(t, 80, any)

	storedFields, _ := messageEvent.MetricSetFields.GetValue("stored_fields")
	require.Equal(t, 40, storedFields)

	docValues, _ := messageEvent.MetricSetFields.GetValue("doc_values")
	require.Equal(t, 0, docValues)

	norms, _ := messageEvent.MetricSetFields.GetValue("norms")
	require.Equal(t, 10, norms)

	terms, _ := messageEvent.MetricSetFields.GetValue("inverted_index.terms")
	require.Equal(t, 30, terms)

	postings, _ := messageEvent.MetricSetFields.GetValue("inverted_index.postings")
	require.Equal(t, 25, postings)

	proximity, _ := messageEvent.MetricSetFields.GetValue("inverted_index.proximity")
	require.Equal(t, 5, proximity)

	trackingID, _ := messageEvent.MetricSetFields.GetValue("shard.tracking_id")
	require.Equal(t, "A_LkEJn8TYWkWJn0Fq-xhA", trackingID)
}
