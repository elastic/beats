// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cluster_health

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	autoopsevents "github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
)

var (
	setupClusterHealthErrorServer = auto_ops_testing.SetupDataErrorServer(ClusterHealthPath)
	setupSuccessfulServer         = auto_ops_testing.SetupSuccessfulServer(ClusterHealthPath)
	useNamedMetricSet             = auto_ops_testing.UseNamedMetricSet(ClusterHealthMetricSet)
)

func TestSuccessfulFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cluster_health.*.json", setupSuccessfulServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[map[string]interface{}]) {
		require.NoError(t, data.Error)

		require.Equal(t, 1, len(data.Reporter.GetEvents()))
	})
}

func TestFailedClusterInfoFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cluster_health.*.json", auto_ops_testing.SetupClusterInfoErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[map[string]interface{}]) {
		require.ErrorContains(t, data.Error, "failed to get cluster info from cluster, cluster_health metricset")
	})
}

func TestFailedClusterHealthFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cluster_health.*.json", setupClusterHealthErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[map[string]interface{}]) {
		require.ErrorContains(t, data.Error, "failed to get data, cluster_health metricset")
	})
}

func TestFailedClusterHealthFetchEventsMapping(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/no_*.cluster_health.*.json", setupSuccessfulServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[map[string]interface{}]) {
		require.Error(t, data.Error)
		require.Equal(t, 1, len(data.Reporter.GetEvents()))

		// Check error event
		event := data.Reporter.GetEvents()[0]
		_, ok := event.MetricSetFields["error"].(autoopsevents.ErrEvent)
		require.True(t, ok, "error field should be of type error.ErrEvent")
	})
}
