// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package node_stats

import (
	"slices"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func expectValidParsedData(t *testing.T, data metricset.FetcherData[NodesStats]) {
	require.NoError(t, data.Error)
	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	// 2 <= len(events)
	require.LessOrEqual(t, 2, len(data.Reporter.GetEvents()))

	events := data.Reporter.GetEvents()

	auto_ops_testing.CheckAllEventsUseSameTransactionId(t, events)

	nodeList := auto_ops_testing.GetEventsWithField(t, events, "subType")

	require.Equal(t, 1, len(nodeList))
	require.LessOrEqual(t, 1, len(auto_ops_testing.GetObjectValue(nodeList[0].MetricSetFields, "nodes").(map[string]string)))

	nodeStats := auto_ops_testing.GetEventsWithField(t, events, "name")

	require.LessOrEqual(t, 1, len(nodeStats))
}

func expectValidParsedDetailed(t *testing.T, data metricset.FetcherData[NodesStats]) {
	expectValidParsedData(t, data)

	events := data.Reporter.GetEvents()

	nodeListEvents := auto_ops_testing.GetEventsWithField(t, events, "subType")
	nodeListEvent := nodeListEvents[0]
	nodeList := auto_ops_testing.GetObjectValue(nodeListEvent.MetricSetFields, "nodes").(map[string]string)

	// TODO: Update the indexer to use these from the metricset and remove this from being checked
	require.Equal(t, "list", nodeListEvent.RootFields["subType"])
	require.Equal(t, nodeList, nodeListEvent.ModuleFields["nodes"])

	if data.Version == "7.17.0" {
		require.Equal(t, 1, len(nodeList))
		require.Equal(t, "instance-0000000001", nodeList["deX3GDaCSQSINcDCm-AtDw"])
	} else if data.Version == "8.15.3" {
		require.Equal(t, 59, len(nodeList))
		require.Equal(t, "instance-0000000001", nodeList["deX3GDaCSQSINcDCm-AtDw"])
		require.Equal(t, "instance-0000000105", nodeList["AwqTc41oSDqGpaGKgdBGpA"])
	}

	nodeStatsEvents := auto_ops_testing.GetEventsWithField(t, events, "name")
	node1 := nodeStatsEvents[slices.IndexFunc(nodeStatsEvents, func(event mb.Event) bool { return event.MetricSetFields["name"] == "instance-0000000001" })]
	node1MetricSet := node1.MetricSetFields

	require.Equal(t, "deX3GDaCSQSINcDCm-AtDw", node1MetricSet["id"])
	require.Equal(t, "instance-0000000001", node1MetricSet["name"])

	// TODO: Update the indexer and remove these from the module fields (and thus stop checking they're there)
	node1ModuleFields := node1.ModuleFields
	require.Equal(t, "deX3GDaCSQSINcDCm-AtDw", auto_ops_testing.GetObjectValue(node1ModuleFields, "node.id"))
	require.Equal(t, "instance-0000000001", auto_ops_testing.GetObjectValue(node1ModuleFields, "node.name"))

	if data.Version == "7.17.0" {
		require.Equal(t, 1, len(nodeStatsEvents))
		require.EqualValues(t, 1, nodeStatsEvents[0].ModuleFields["totalAmountOfFractions"])

		// TODO: Remove module fields
		require.Equal(t, "10.42.0.2", auto_ops_testing.GetObjectValue(node1ModuleFields, "node.host"))
		require.Equal(t, true, auto_ops_testing.GetObjectValue(node1ModuleFields, "node.is_elected_master"))
		require.ElementsMatch(t, []string{"data_content", "data_hot", "ingest", "master", "remote_cluster_client", "transform"}, auto_ops_testing.GetObjectValue(node1ModuleFields, "node.roles"))

		// metricset fields
		require.Equal(t, "10.42.0.2", node1MetricSet["host"])
		require.Equal(t, true, node1MetricSet["is_elected_master"])
		require.ElementsMatch(t, []string{"data_content", "data_hot", "ingest", "master", "remote_cluster_client", "transform"}, node1MetricSet["roles"])
		require.EqualValues(t, 2337, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.docs.count"))
		require.EqualValues(t, 45203023, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.store.size_in_bytes"))
		require.EqualValues(t, 1390859, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.indexing.index_total"))
		require.EqualValues(t, 942011, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.indexing.index_time_in_millis"))
		require.EqualValues(t, 164, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.indexing.index_failed"))
		require.EqualValues(t, 73560, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.merges.total"))
		require.EqualValues(t, 1101515, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.merges.total_time_in_millis"))
		require.EqualValues(t, 2387966, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.search.query_total"))
		require.EqualValues(t, 1742827, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.search.query_time_in_millis"))
		require.EqualValues(t, 50, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.segments.count"))
		require.EqualValues(t, 2, auto_ops_testing.GetObjectValue(node1MetricSet, "thread_pool.write.threads"))
		require.EqualValues(t, 1777, auto_ops_testing.GetObjectValue(node1MetricSet, "thread_pool.write.completed"))
	} else if data.Version == "8.15.3" {
		require.Equal(t, 59, len(nodeStatsEvents))
		require.EqualValues(t, 59, nodeStatsEvents[0].ModuleFields["totalAmountOfFractions"])

		// TODO: Remove module fields
		require.Equal(t, "172.22.238.181", auto_ops_testing.GetObjectValue(node1ModuleFields, "node.host"))
		require.Equal(t, false, auto_ops_testing.GetObjectValue(node1ModuleFields, "node.is_elected_master"))
		require.ElementsMatch(t, []string{"data_content", "data_hot", "ingest", "remote_cluster_client", "transform"}, auto_ops_testing.GetObjectValue(node1ModuleFields, "node.roles"))

		// metricset fields
		require.Equal(t, "172.22.238.181", node1MetricSet["host"])
		require.Equal(t, false, node1MetricSet["is_elected_master"])
		require.ElementsMatch(t, []string{"data_content", "data_hot", "ingest", "remote_cluster_client", "transform"}, node1MetricSet["roles"])
		require.EqualValues(t, 3836668558, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.docs.count"))
		require.EqualValues(t, 814301334447, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.store.size_in_bytes"))
		require.EqualValues(t, 187857964532, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.indexing.index_total"))
		require.EqualValues(t, 23116646135, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.indexing.index_time_in_millis"))
		require.EqualValues(t, 266, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.indexing.index_failed"))
		require.EqualValues(t, 36933162, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.merges.total"))
		require.EqualValues(t, 34264942295, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.merges.total_time_in_millis"))
		require.EqualValues(t, 175109606, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.search.query_total"))
		require.EqualValues(t, 3464297906, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.search.query_time_in_millis"))
		require.EqualValues(t, 5358, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.segments.count"))
		require.EqualValues(t, 32, auto_ops_testing.GetObjectValue(node1MetricSet, "thread_pool.write.threads"))
		require.EqualValues(t, 24175874622, auto_ops_testing.GetObjectValue(node1MetricSet, "thread_pool.write.completed"))
	}

	// some ignored values
	require.Nil(t, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.shard_stats"))
	require.Nil(t, auto_ops_testing.GetObjectValue(node1MetricSet, "indices.store.total_data_set_size_in_bytes"))
}

func expectValidParsedDetailedWithNoCache(t *testing.T, data metricset.FetcherData[NodesStats]) {
	expectValidParsedDetailed(t, data)

	nodeStatsEvents := auto_ops_testing.GetEventsWithField(t, data.Reporter.GetEvents(), "name")
	node1 := nodeStatsEvents[slices.IndexFunc(nodeStatsEvents, func(event mb.Event) bool { return event.MetricSetFields["name"] == "instance-0000000001" })]
	node1MetricSet := node1.MetricSetFields

	require.Nil(t, node1MetricSet["index_failed_rate_per_second"])
	require.Nil(t, node1MetricSet["index_rate_per_second"])
	require.Nil(t, node1MetricSet["merge_rate_per_second"])
	require.Nil(t, node1MetricSet["search_rate_per_second"])
	require.Nil(t, node1MetricSet["index_latency_in_millis"])
	require.Nil(t, node1MetricSet["merge_latency_in_millis"])
	require.Nil(t, node1MetricSet["search_latency_in_millis"])
}

func expectValidParsedDetailedWithCache(t *testing.T, data metricset.FetcherData[NodesStats]) {
	expectValidParsedDetailed(t, data)

	nodeStatsEvents := auto_ops_testing.GetEventsWithField(t, data.Reporter.GetEvents(), "name")
	node1 := nodeStatsEvents[slices.IndexFunc(nodeStatsEvents, func(event mb.Event) bool { return event.MetricSetFields["name"] == "instance-0000000001" })]
	node1MetricSet := node1.MetricSetFields

	// cache_test.go checks for actual caching with exact timing
	require.NotNil(t, node1MetricSet["index_failed_rate_per_second"])
	require.NotNil(t, node1MetricSet["index_rate_per_second"])
	require.NotNil(t, node1MetricSet["merge_rate_per_second"])
	require.NotNil(t, node1MetricSet["search_rate_per_second"])
	require.NotNil(t, node1MetricSet["index_latency_in_millis"])
	require.NotNil(t, node1MetricSet["merge_latency_in_millis"])
	require.NotNil(t, node1MetricSet["search_latency_in_millis"])
}

// Tests that Cluster Info is consistently reported and the Node Stats is properly reported
func expectMixedValidParsedData(t *testing.T, data metricset.FetcherData[NodesStats]) {
	require.ErrorContains(t, data.Error, "failed applying nodes_stats schema")

	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	require.Equal(t, 3, len(data.Reporter.GetEvents()))

	events := data.Reporter.GetEvents()

	auto_ops_testing.CheckAllEventsUseSameTransactionId(t, events)

	event := auto_ops_testing.GetEventByName(t, events, "name", "instance-0000000001")

	auto_ops_testing.CheckEvent(t, event, data.ClusterInfo)

	// metrics exist
	require.True(t, len(*event.MetricSetFields.FlattenKeys()) > 2)
	require.Equal(t, "deX3GDaCSQSINcDCm-AtDw", auto_ops_testing.GetObjectValue(event.MetricSetFields, "id"))
	require.Equal(t, "instance-0000000001", auto_ops_testing.GetObjectValue(event.MetricSetFields, "name"))
	require.Equal(t, "10.42.0.2", auto_ops_testing.GetObjectValue(event.MetricSetFields, "host"))
	require.Equal(t, true, auto_ops_testing.GetObjectValue(event.MetricSetFields, "is_elected_master"))
	require.ElementsMatch(t, []string{"data_content", "data_hot", "ingest", "master", "remote_cluster_client", "transform"}, auto_ops_testing.GetObjectValue(event.MetricSetFields, "roles"))

	// schema is expected to drop unknown fields
	require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "ignored_field"))

	event = auto_ops_testing.GetEventsWithField(t, events, "nodes")[0]

	require.Equal(t, map[string]string{"deX3GDaCSQSINcDCm-AtDw": "instance-0000000001"}, auto_ops_testing.GetObjectValue(event.MetricSetFields, "nodes"))
}

// Tests that the schema rejects the data
func expectError(t *testing.T, data metricset.FetcherData[NodesStats]) {
	require.ErrorContains(t, data.Error, "failed applying nodes_stats schema")
}

// Expect a valid response from Elasticsearch to create N events
func TestProperlyHandlesResponse(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFilesAndSetup(t, "./_meta/test/nodes_stats.*.json", setupSuccessfulServer(), useNamedMetricSet, expectValidParsedData, clearCache)
}

// Expect a valid response from Elasticsearch to create N events
func TestProperlyHandlesResponseWithDetails(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFilesAndSetup(t, "./_meta/test/nodes_stats.*.json", setupSuccessfulServer(), useNamedMetricSet, expectValidParsedDetailedWithNoCache, clearCache)
}

// Expect a valid response from Elasticsearch to create N events that run after a previous run that sets up the cache.
func TestProperlyHandlesResponseWithDetailsWithCache(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFilesAndSetup(t, "./_meta/test/nodes_stats.*.json", setupSuccessfulServer(), useNamedMetricSet, expectValidParsedDetailedWithCache, func() {
		initCache(map[string]mapstr.M{
			"deX3GDaCSQSINcDCm-AtDw": getNodeStatsForNode(0),
		}, 10)
	})
}

// Expect a valid mixed with invalid response from Elasticsearch to create N events without bad data
func TestProperlyHandlesInnerErrorsInResponse(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFilesAndSetup(t, "./_meta/test/mixed.nodes_stats.*.json", setupSuccessfulServer(), useNamedMetricSet, expectMixedValidParsedData, clearCache)
}

// Expect a corrupt response from Elasticsearch to trigger an error while applying the schema
func TestProperlyFailsOnBadResponse(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFilesAndSetup(t, "./_meta/test/no_*.nodes_stats.*.json", setupSuccessfulServer(), useNamedMetricSet, expectError, clearCache)
}
