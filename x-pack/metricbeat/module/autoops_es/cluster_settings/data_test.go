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

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
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
	require.NotNil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "defaults"))

	// filename includes the "display_name" as the second part
	displayName := strings.Split(data.File, ".")[1]

	require.Equal(t, displayName, auto_ops_testing.GetObjectValue(event.MetricSetFields, "persistent.cluster.metadata.display_name"))
	require.ElementsMatch(t, []string{"/app/data"}, auto_ops_testing.GetObjectValue(event.MetricSetFields, "defaults.path.data"))

	// schema is expected to drop this field if it appears (it does in one file)
	require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "ignored_field"))
}

// Tests that the schema rejects the data
func expectError(t *testing.T, data metricset.FetcherData[map[string]interface{}]) {
	require.ErrorContains(t, data.Error, "failed applying cluster settings schema")
}

// Expect a valid response from Elasticsearch to create a single event
func TestProperlyHandlesResponse(t *testing.T) {
	metricset.RunTestsForServerlessMetricSetWithGlobFiles(t, "./_meta/test/cluster_settings.*.json", ClusterSettingsMetricSet, eventsMapping, expectValidParsedData)
}

// Expect a corrupt response from Elasticsearch to trigger an error while applying the schema
func TestProperlyFailsOnBadResponse(t *testing.T) {
	metricset.RunTestsForServerlessMetricSetWithGlobFiles[map[string]interface{}](t, "./_meta/test/no_*.cluster_settings.*.json", ClusterSettingsMetricSet, eventsMapping, expectError)
}
