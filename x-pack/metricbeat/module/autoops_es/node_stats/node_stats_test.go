// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package node_stats

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	auto_ops_events "github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
)

var (
	useNamedMetricSet = auto_ops_testing.UseNamedMetricSet(NodesStatsMetricSet)
)

func TestSuccessfulFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/nodes_stats.*.json", setupSuccessfulServer(), useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[NodesStats]) {
		require.NoError(t, data.Error)

		require.LessOrEqual(t, 2, len(data.Reporter.GetEvents()))
	})
}

func TestFailedClusterInfoFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/nodes_stats.*.json", auto_ops_testing.SetupClusterInfoErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[NodesStats]) {
		require.ErrorContains(t, data.Error, "failed to get cluster info from cluster, node_stats metricset")
	})
}

func TestFailedNodeStatsFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/nodes_stats.*.json", auto_ops_testing.SetupDataErrorServer(NodesStatsPath), useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[NodesStats]) {
		require.ErrorContains(t, data.Error, "failed to get data, node_stats metricset")
	})
}

func TestFailedNodeStatsMasterNode(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/nodes_stats.*.json", setupMasterNodeErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[NodesStats]) {
		require.Error(t, data.Error)
	})
}

func TestSendErrorEventWhenFailedNodeStatsMasterNode(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/error_nodes_stats.*.json", setupMasterNodeErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[NodesStats]) {
		require.Error(t, data.Error)

		// Check error event
		event := data.Reporter.GetEvents()[2]
		_, ok := event.MetricSetFields["error"].(auto_ops_events.ErrorEvent)
		require.True(t, ok, "expected error event to be of type auto_ops_events.ErrorEvent")
	})
}
