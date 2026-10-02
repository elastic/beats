// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package frozen_cache_stats

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func expectValidParsedData(t *testing.T, data metricset.FetcherData[CacheStatsResponse]) {
	require.NoError(t, data.Error)
	require.Equal(t, 0, len(data.Reporter.GetErrors()))

	events := data.Reporter.GetEvents()
	require.LessOrEqual(t, 1, len(events))

	auto_ops_testing.CheckAllEventsUseSameTransactionId(t, events)

	nodeEvents := auto_ops_testing.GetEventsWithField(t, events, "id")
	require.LessOrEqual(t, 1, len(nodeEvents))
}

func expectValidParsedDetailed(t *testing.T, data metricset.FetcherData[CacheStatsResponse]) {
	expectValidParsedData(t, data)

	events := data.Reporter.GetEvents()
	nodeEvents := auto_ops_testing.GetEventsWithField(t, events, "id")

	if data.Version == "8.15.3" {
		require.Equal(t, 2, len(nodeEvents))

		node1 := auto_ops_testing.GetEventByName(t, nodeEvents, "id", "eerrtBMtQEisohZzxBLUSw")
		m := node1.MetricSetFields

		require.Equal(t, "eerrtBMtQEisohZzxBLUSw", m["id"])
		require.EqualValues(t, 6051, auto_ops_testing.GetObjectValue(m, "shared_cache.reads"))
		require.EqualValues(t, 37, auto_ops_testing.GetObjectValue(m, "shared_cache.writes"))
		require.EqualValues(t, 5, auto_ops_testing.GetObjectValue(m, "shared_cache.evictions"))
		require.EqualValues(t, 1099511627776, auto_ops_testing.GetObjectValue(m, "shared_cache.size_in_bytes"))
		require.EqualValues(t, 16777216, auto_ops_testing.GetObjectValue(m, "shared_cache.region_size_in_bytes"))
		require.EqualValues(t, 65536, auto_ops_testing.GetObjectValue(m, "shared_cache.num_regions"))
		require.EqualValues(t, 1208320, auto_ops_testing.GetObjectValue(m, "shared_cache.bytes_written_in_bytes"))
		require.EqualValues(t, 5448829, auto_ops_testing.GetObjectValue(m, "shared_cache.bytes_read_in_bytes"))
	}
}

func expectValidParsedDetailedWithNoCache(t *testing.T, data metricset.FetcherData[CacheStatsResponse]) {
	expectValidParsedDetailed(t, data)

	nodeEvents := auto_ops_testing.GetEventsWithField(t, data.Reporter.GetEvents(), "id")
	node1 := auto_ops_testing.GetEventByName(t, nodeEvents, "id", "eerrtBMtQEisohZzxBLUSw")
	m := node1.MetricSetFields

	require.Nil(t, m["shared_cache_reads_per_second"])
	require.Nil(t, m["shared_cache_writes_per_second"])
	require.Nil(t, m["shared_cache_evictions_per_second"])
	require.Nil(t, m["shared_cache_bytes_written_per_second"])
	require.Nil(t, m["shared_cache_bytes_read_per_second"])
}

func expectValidParsedDetailedWithCache(t *testing.T, data metricset.FetcherData[CacheStatsResponse]) {
	expectValidParsedDetailed(t, data)

	nodeEvents := auto_ops_testing.GetEventsWithField(t, data.Reporter.GetEvents(), "id")
	node1 := auto_ops_testing.GetEventByName(t, nodeEvents, "id", "eerrtBMtQEisohZzxBLUSw")
	m := node1.MetricSetFields

	require.NotNil(t, m["shared_cache_reads_per_second"])
	require.NotNil(t, m["shared_cache_writes_per_second"])
	require.NotNil(t, m["shared_cache_evictions_per_second"])
	require.NotNil(t, m["shared_cache_bytes_written_per_second"])
	require.NotNil(t, m["shared_cache_bytes_read_per_second"])
}

func TestSkipsNodesWithNoSharedCache(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFilesAndSetup(t, "./_meta/test/no_frozen_nodes.frozen_cache_stats.*.json", auto_ops_testing.SetupSuccessfulServer(FrozenCacheStatsPath), useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[CacheStatsResponse]) {
		require.NoError(t, data.Error)
		require.Equal(t, 0, len(data.Reporter.GetEvents()))
	}, clearCache)
}

func TestProperlyHandlesResponse(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFilesAndSetup(t, "./_meta/test/frozen_cache_stats.*.json", auto_ops_testing.SetupSuccessfulServer(FrozenCacheStatsPath), useNamedMetricSet, expectValidParsedData, clearCache)
}

func TestProperlyHandlesResponseWithDetails(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFilesAndSetup(t, "./_meta/test/frozen_cache_stats.*.json", auto_ops_testing.SetupSuccessfulServer(FrozenCacheStatsPath), useNamedMetricSet, expectValidParsedDetailedWithNoCache, clearCache)
}

func TestProperlyHandlesResponseWithDetailsAndCache(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFilesAndSetup(t, "./_meta/test/frozen_cache_stats.*.json", auto_ops_testing.SetupSuccessfulServer(FrozenCacheStatsPath), useNamedMetricSet, expectValidParsedDetailedWithCache, func() {
		initCache(map[string]mapstr.M{
			"eerrtBMtQEisohZzxBLUSw": getNodeCacheStatsForNode(5000, 25, 3, 800000, 4000000),
		}, 10)
	})
}
