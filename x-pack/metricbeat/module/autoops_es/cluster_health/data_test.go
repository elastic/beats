// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cluster_health

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
	require.True(t, len(*event.MetricSetFields.FlattenKeys()) > 2)
	require.Equal(t, data.ClusterInfo.ClusterName, auto_ops_testing.GetObjectValue(event.MetricSetFields, "cluster_name"))

	// filename includes the status as the second part
	status := strings.Split(data.File, ".")[1]

	require.Equal(t, status, auto_ops_testing.GetObjectValue(event.MetricSetFields, "status"))

	// schema is expected to drop this field if it appears (it does in one file)
	require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "ignored_field"))
}

// Tests that the schema rejects the data
func expectError(t *testing.T, data metricset.FetcherData[map[string]interface{}]) {
	require.ErrorContains(t, data.Error, "failed applying cluster health schema")
}

// Expect a valid response from Elasticsearch to create a single event
func TestProperlyHandlesResponse(t *testing.T) {
	metricset.RunTestsForServerlessMetricSetWithGlobFiles(t, "./_meta/test/cluster_health.*.json", ClusterHealthMetricSet, eventsMapping, expectValidParsedData)
}

// Expect a corrupt response from Elasticsearch to trigger an error while applying the schema
func TestProperlyFailsOnBadResponse(t *testing.T) {
	metricset.RunTestsForServerlessMetricSetWithGlobFiles(t, "./_meta/test/no_*.cluster_health.*.json", ClusterHealthMetricSet, eventsMapping, expectError)
}
