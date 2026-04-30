// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package node_stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func clearCache() {
	cache.PreviousCache = nil
	cache.PreviousTimestamp = 0
}

func initCache(previousCache map[string]mapstr.M, previousSeconds int64) {
	cache.NewTimestamp = time.Now().UnixMilli()

	cache.PreviousCache = previousCache
	cache.PreviousTimestamp = cache.NewTimestamp - (previousSeconds * 1_000)
}

func getNodeStatsForNode(nodeIndex int64) mapstr.M {
	return mapstr.M{
		"indices": mapstr.M{
			"indexing": mapstr.M{
				"index_failed":         10 + nodeIndex,
				"index_total":          20 + nodeIndex,
				"index_time_in_millis": 50 + nodeIndex,
			},
			"merges": mapstr.M{
				"total":                30 + nodeIndex,
				"total_time_in_millis": 60 + nodeIndex,
			},
			"search": mapstr.M{
				"query_time_in_millis": 70 + nodeIndex,
				"query_total":          40 + nodeIndex,
			},
		},
	}
}

func getNodeStats() map[string]mapstr.M {
	return map[string]mapstr.M{
		"node1": getNodeStatsForNode(0),
		"node2": getNodeStatsForNode(1),
	}
}

func TestEnrichNodeStatsWithoutCache(t *testing.T) {
	clearCache()

	nodeStatsMap := getNodeStats()
	nodeStatsNode1 := nodeStatsMap["node1"]

	enrichNodeStats("node1", &nodeStatsNode1, 0)

	// rates / latencies from new node unknown for one pass
	require.Nil(t, nodeStatsNode1["index_failed_rate_per_second"])
	require.Nil(t, nodeStatsNode1["index_rate_per_second"])
	require.Nil(t, nodeStatsNode1["merge_rate_per_second"])
	require.Nil(t, nodeStatsNode1["search_rate_per_second"])
	require.Nil(t, nodeStatsNode1["index_latency_in_millis"])
	require.Nil(t, nodeStatsNode1["merge_latency_in_millis"])
	require.Nil(t, nodeStatsNode1["search_latency_in_millis"])
}

func TestEnrichNodeStatsWithoutCachedValues(t *testing.T) {
	// empty, but not nil cache; 10s ago
	initCache(map[string]mapstr.M{}, 10)

	nodeStatsMap := getNodeStats()
	nodeStatsNode1 := nodeStatsMap["node1"]

	enrichNodeStats("node1", &nodeStatsNode1, 0)

	// rates / latencies from new node unknown for one pass
	require.Nil(t, nodeStatsNode1["index_failed_rate_per_second"])
	require.Nil(t, nodeStatsNode1["index_rate_per_second"])
	require.Nil(t, nodeStatsNode1["merge_rate_per_second"])
	require.Nil(t, nodeStatsNode1["search_rate_per_second"])
	require.Nil(t, nodeStatsNode1["index_latency_in_millis"])
	require.Nil(t, nodeStatsNode1["merge_latency_in_millis"])
	require.Nil(t, nodeStatsNode1["search_latency_in_millis"])
}

func TestEnrichNodeStatsWithCachedValues(t *testing.T) {
	// 10s ago cache
	initCache(getNodeStats(), 10)

	nodeStatsMap := getNodeStats()

	for key, nodeStats := range nodeStatsMap {
		nodeStats["indices.indexing.index_failed"] = getValue(&nodeStats, "indices.indexing.index_failed") + 30
		nodeStats["indices.indexing.index_total"] = getValue(&nodeStats, "indices.indexing.index_total") + 20
		nodeStats["indices.indexing.index_time_in_millis"] = getValue(&nodeStats, "indices.indexing.index_time_in_millis") + 10
		nodeStats["indices.merges.total"] = getValue(&nodeStats, "indices.merges.total") + 40
		nodeStats["indices.merges.total_time_in_millis"] = getValue(&nodeStats, "indices.merges.total_time_in_millis") + 40
		nodeStats["indices.search.query_total"] = getValue(&nodeStats, "indices.search.query_total") + 60
		nodeStats["indices.search.query_time_in_millis"] = getValue(&nodeStats, "indices.search.query_time_in_millis") + 120

		nodeStatsMap[key] = nodeStats
	}

	nodeStatsNode1 := nodeStatsMap["node1"]
	enrichNodeStats("node1", &nodeStatsNode1, 10000)
	nodeStatsMap["node1"] = nodeStatsNode1

	nodeStatsNode2 := nodeStatsMap["node2"]
	enrichNodeStats("node2", &nodeStatsNode2, 10000)
	nodeStatsMap["node2"] = nodeStatsNode2

	for _, nodeStats := range nodeStatsMap {
		// rates
		require.EqualValues(t, 2, nodeStats["index_rate_per_second"])
		require.EqualValues(t, 3, nodeStats["index_failed_rate_per_second"])
		require.EqualValues(t, 4, nodeStats["merge_rate_per_second"])
		require.EqualValues(t, 6, nodeStats["search_rate_per_second"])
		// latencies
		require.EqualValues(t, 0.5, nodeStats["index_latency_in_millis"])
		require.EqualValues(t, 1, nodeStats["merge_latency_in_millis"])
		require.EqualValues(t, 2, nodeStats["search_latency_in_millis"])
	}
}

func TestEnrichNodeStatsWithCachedValuesWithNoChange(t *testing.T) {
	// 10s ago cache
	initCache(getNodeStats(), 10)

	nodeStatsMap := getNodeStats()

	nodeStatsNode1 := nodeStatsMap["node1"]
	enrichNodeStats("node1", &nodeStatsNode1, 10000)
	nodeStatsMap["node1"] = nodeStatsNode1

	nodeStatsNode2 := nodeStatsMap["node2"]
	enrichNodeStats("node2", &nodeStatsNode2, 10000)
	nodeStatsMap["node2"] = nodeStatsNode2

	for _, nodeStats := range nodeStatsMap {
		// rates
		require.EqualValues(t, 0, nodeStats["index_rate_per_second"])
		require.EqualValues(t, 0, nodeStats["index_failed_rate_per_second"])
		require.EqualValues(t, 0, nodeStats["merge_rate_per_second"])
		require.EqualValues(t, 0, nodeStats["search_rate_per_second"])
		// latencies
		require.EqualValues(t, 0, nodeStats["index_latency_in_millis"])
		require.EqualValues(t, 0, nodeStats["merge_latency_in_millis"])
		require.EqualValues(t, 0, nodeStats["search_latency_in_millis"])
	}
}

func TestEnrichNodeStatsWithCachedValuesWithHoles(t *testing.T) {
	// 10s ago cache
	initCache(getNodeStats(), 10)

	nodeStatsMap := getNodeStats()

	for key, nodeStats := range nodeStatsMap {
		nodeStatsMap[key] = nodeStats

		if key == "node2" {
			nodeStats["indices.indexing.index_total"] = getValue(&nodeStats, "indices.indexing.index_total") + 20
			nodeStats["indices.indexing.index_time_in_millis"] = getValue(&nodeStats, "indices.indexing.index_time_in_millis") + 10
		} else {
			nodeStats.Delete("indices.indexing.index_total")
			nodeStats.Delete("indices.indexing.index_time_in_millis")
		}

		nodeStats["indices.indexing.index_failed"] = getValue(&nodeStats, "indices.indexing.index_failed") + 30
		nodeStats["indices.merges.total"] = getValue(&nodeStats, "indices.merges.total") + 40
		nodeStats["indices.merges.total_time_in_millis"] = getValue(&nodeStats, "indices.merges.total_time_in_millis") + 40
		nodeStats["indices.search.query_total"] = getValue(&nodeStats, "indices.search.query_total") + 60
		nodeStats["indices.search.query_time_in_millis"] = getValue(&nodeStats, "indices.search.query_time_in_millis") + 120

		nodeStatsMap[key] = nodeStats
	}

	nodeStatsNode1 := nodeStatsMap["node1"]
	enrichNodeStats("node1", &nodeStatsNode1, 10000)
	nodeStatsMap["node1"] = nodeStatsNode1

	nodeStatsNode2 := nodeStatsMap["node2"]
	enrichNodeStats("node2", &nodeStatsNode2, 10000)
	nodeStatsMap["node2"] = nodeStatsNode2

	for key, nodeStats := range nodeStatsMap {
		// rates
		if key == "node2" {
			require.EqualValues(t, 2, nodeStats["index_rate_per_second"])
		} else {
			require.Nil(t, nodeStats["index_rate_per_second"])
		}

		require.EqualValues(t, 3, nodeStats["index_failed_rate_per_second"])
		require.EqualValues(t, 4, nodeStats["merge_rate_per_second"])
		require.EqualValues(t, 6, nodeStats["search_rate_per_second"])

		// latencies
		if key == "node2" {
			require.EqualValues(t, 0.5, nodeStats["index_latency_in_millis"])
		} else {
			require.Nil(t, nodeStats["index_latency_in_millis"])
		}
		require.EqualValues(t, 1, nodeStats["merge_latency_in_millis"])
		require.EqualValues(t, 2, nodeStats["search_latency_in_millis"])
	}
}

func TestEnrichNodeIndexShardsWithCachedValuesWithNewNodeAndIndex(t *testing.T) {
	// 10s ago cache
	initCache(getNodeStats(), 10)

	nodeStatsMap := getNodeStats()

	for key, nodeStats := range nodeStatsMap {
		nodeStats["indices.indexing.index_failed"] = getValue(&nodeStats, "indices.indexing.index_failed") + 30
		nodeStats["indices.indexing.index_total"] = getValue(&nodeStats, "indices.indexing.index_total") + 20
		nodeStats["indices.indexing.index_time_in_millis"] = getValue(&nodeStats, "indices.indexing.index_time_in_millis") + 10
		nodeStats["indices.merges.total"] = getValue(&nodeStats, "indices.merges.total") + 40
		nodeStats["indices.merges.total_time_in_millis"] = getValue(&nodeStats, "indices.merges.total_time_in_millis") + 40
		nodeStats["indices.search.query_total"] = getValue(&nodeStats, "indices.search.query_total") + 60
		nodeStats["indices.search.query_time_in_millis"] = getValue(&nodeStats, "indices.search.query_time_in_millis") + 120

		nodeStatsMap[key] = nodeStats
	}

	nodeStatsMap["node3"] = getNodeStatsForNode(2)

	nodeStatsNode1 := nodeStatsMap["node1"]
	enrichNodeStats("node1", &nodeStatsNode1, 10000)
	nodeStatsMap["node1"] = nodeStatsNode1

	nodeStatsNode2 := nodeStatsMap["node2"]
	enrichNodeStats("node2", &nodeStatsNode2, 10000)
	nodeStatsMap["node2"] = nodeStatsNode2

	nodeStatsNode3 := nodeStatsMap["node3"]
	enrichNodeStats("node3", &nodeStatsNode3, 10000)
	nodeStatsMap["node3"] = nodeStatsNode3

	for key, nodeStats := range nodeStatsMap {
		if key != "node3" {
			// rates
			require.EqualValues(t, 2, nodeStats["index_rate_per_second"])
			require.EqualValues(t, 3, nodeStats["index_failed_rate_per_second"])
			require.EqualValues(t, 4, nodeStats["merge_rate_per_second"])
			require.EqualValues(t, 6, nodeStats["search_rate_per_second"])
			// latencies
			require.EqualValues(t, 0.5, nodeStats["index_latency_in_millis"])
			require.EqualValues(t, 1, nodeStats["merge_latency_in_millis"])
			require.EqualValues(t, 2, nodeStats["search_latency_in_millis"])
		} else {
			// rates / latencies from new node unknown for one pass
			require.Nil(t, nodeStats["index_failed_rate_per_second"])
			require.Nil(t, nodeStats["index_rate_per_second"])
			require.Nil(t, nodeStats["merge_rate_per_second"])
			require.Nil(t, nodeStats["search_rate_per_second"])
			require.Nil(t, nodeStats["index_latency_in_millis"])
			require.Nil(t, nodeStats["merge_latency_in_millis"])
			require.Nil(t, nodeStats["search_latency_in_millis"])
		}
	}
}
