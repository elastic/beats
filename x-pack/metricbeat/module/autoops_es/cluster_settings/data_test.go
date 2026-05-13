// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cluster_settings

import (
	"strings"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"

	"github.com/stretchr/testify/require"
)

// Tests that Cluster Info is consistently reported, the Cluster Name, and the dynamic status from the filename
func expectValidParsedData(t *testing.T, data metricset.FetcherData[map[string]interface{}]) {
	require.NoError(t, data.Error)
	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	require.Equal(t, 1, len(data.Reporter.GetEvents()))

	event := data.Reporter.GetEvents()[0]

	auto_ops_testing.CheckEventWithRandomTransactionId(t, event, data.ClusterInfo)

	// metrics exist
	require.True(t, len(*event.MetricSetFields.FlattenKeys()) > 3)

	// filename includes the "display_name" as the second part
	displayName := strings.Split(data.File, ".")[1]

	// Settings are flattened with precedence: transient > persistent > defaults
	require.Equal(t, "4000", auto_ops_testing.GetObjectValue(event.MetricSetFields, "cluster.max_shards_per_node"))
	require.Equal(t, displayName, auto_ops_testing.GetObjectValue(event.MetricSetFields, "cluster.metadata.display_name"))
	require.ElementsMatch(t, []string{"/app/data"}, auto_ops_testing.GetObjectValue(event.MetricSetFields, "path.data"))
	require.Equal(t, "3", auto_ops_testing.GetObjectValue(event.MetricSetFields, "serverless.search.search_power_min"))

	// schema is expected to drop this field if it appears (it does in one file)
	require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "ignored_field"))
}

// Expect a valid response from Elasticsearch to create a single event
func TestProperlyHandlesResponse(t *testing.T) {
	metricset.RunTestsForServerlessMetricSetWithGlobFiles(t, "./_meta/test/cluster_settings.*.json", ClusterSettingsMetricSet, eventsMapping, expectValidParsedData)
}
