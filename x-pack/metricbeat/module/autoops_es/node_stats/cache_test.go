// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package node_stats

import (
	"math"
	"testing"
	"testing/quick"
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
			"docs": mapstr.M{
				"count": 100 + nodeIndex,
			},
			"store": mapstr.M{
				"size_in_bytes": 200 + nodeIndex,
			},
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
			"bulk": mapstr.M{
				"total_size_in_bytes": 300 + nodeIndex,
				"total_operations":    400 + nodeIndex,
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
	require.Nil(t, nodeStatsNode1["ingest_docs_per_second"])
	require.Nil(t, nodeStatsNode1["ingest_bytes_per_second"])
	require.Nil(t, nodeStatsNode1["bulk_bytes_per_second"])
	require.Nil(t, nodeStatsNode1["bulk_operations_per_second"])
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
	require.Nil(t, nodeStatsNode1["ingest_docs_per_second"])
	require.Nil(t, nodeStatsNode1["ingest_bytes_per_second"])
	require.Nil(t, nodeStatsNode1["bulk_bytes_per_second"])
	require.Nil(t, nodeStatsNode1["bulk_operations_per_second"])
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
		nodeStats["indices.docs.count"] = getValue(&nodeStats, "indices.docs.count") + 10
		nodeStats["indices.store.size_in_bytes"] = getValue(&nodeStats, "indices.store.size_in_bytes") + 20
		nodeStats["indices.bulk.total_size_in_bytes"] = getValue(&nodeStats, "indices.bulk.total_size_in_bytes") + 100
		nodeStats["indices.bulk.total_operations"] = getValue(&nodeStats, "indices.bulk.total_operations") + 50

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
		require.EqualValues(t, 1, nodeStats["ingest_docs_per_second"])
		require.EqualValues(t, 2, nodeStats["ingest_bytes_per_second"])
		require.EqualValues(t, 10, nodeStats["bulk_bytes_per_second"])
		require.EqualValues(t, 5, nodeStats["bulk_operations_per_second"])
		// latencies
		require.EqualValues(t, 0.5, nodeStats["index_latency_in_millis"])
		require.EqualValues(t, 1, nodeStats["merge_latency_in_millis"])
		require.EqualValues(t, 2, nodeStats["search_latency_in_millis"])
	}
}

func TestEnrichNodeStatsSearchLatencyClampsToInterval(t *testing.T) {
	// 10s sampling interval
	initCache(getNodeStats(), 10)

	nodeStatsMap := getNodeStats()

	for key, nodeStats := range nodeStatsMap {
		// Small deltas for index and merge — raw latencies well below 10 000 ms
		nodeStats["indices.indexing.index_total"] = getValue(&nodeStats, "indices.indexing.index_total") + 3
		nodeStats["indices.indexing.index_time_in_millis"] = getValue(&nodeStats, "indices.indexing.index_time_in_millis") + 30
		nodeStats["indices.indexing.index_failed"] = getValue(&nodeStats, "indices.indexing.index_failed") + 3
		nodeStats["indices.merges.total"] = getValue(&nodeStats, "indices.merges.total") + 3
		nodeStats["indices.merges.total_time_in_millis"] = getValue(&nodeStats, "indices.merges.total_time_in_millis") + 30
		// Reproduces #2471: 3 search ops with combined query_time > interval
		// raw latency = 120 000 / 3 = 40 000 ms/op; interval = 10 000 ms → clamped
		nodeStats["indices.search.query_total"] = getValue(&nodeStats, "indices.search.query_total") + 3
		nodeStats["indices.search.query_time_in_millis"] = getValue(&nodeStats, "indices.search.query_time_in_millis") + 120_000

		nodeStatsMap[key] = nodeStats
	}

	nodeStatsNode1 := nodeStatsMap["node1"]
	enrichNodeStats("node1", &nodeStatsNode1, 10_000)
	nodeStatsMap["node1"] = nodeStatsNode1

	nodeStatsNode2 := nodeStatsMap["node2"]
	enrichNodeStats("node2", &nodeStatsNode2, 10_000)
	nodeStatsMap["node2"] = nodeStatsNode2

	for _, nodeStats := range nodeStatsMap {
		require.EqualValues(t, 10_000, nodeStats["search_latency_in_millis"],
			"search latency exceeding the sampling interval should be clamped to the interval")

		// Index and merge latencies are well below the interval and must not be clamped
		require.InDelta(t, 10, nodeStats["index_latency_in_millis"], 0.01,
			"index latency below interval should not be clamped")
		require.InDelta(t, 10, nodeStats["merge_latency_in_millis"], 0.01,
			"merge latency below interval should not be clamped")

		// No increments for ingest/bulk counters → rates are zero
		require.EqualValues(t, 0, nodeStats["ingest_docs_per_second"])
		require.EqualValues(t, 0, nodeStats["ingest_bytes_per_second"])
		require.EqualValues(t, 0, nodeStats["bulk_bytes_per_second"])
		require.EqualValues(t, 0, nodeStats["bulk_operations_per_second"])
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
		require.EqualValues(t, 0, nodeStats["ingest_docs_per_second"])
		require.EqualValues(t, 0, nodeStats["ingest_bytes_per_second"])
		require.EqualValues(t, 0, nodeStats["bulk_bytes_per_second"])
		require.EqualValues(t, 0, nodeStats["bulk_operations_per_second"])
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
		nodeStats["indices.docs.count"] = getValue(&nodeStats, "indices.docs.count") + 10
		nodeStats["indices.store.size_in_bytes"] = getValue(&nodeStats, "indices.store.size_in_bytes") + 20
		nodeStats["indices.bulk.total_size_in_bytes"] = getValue(&nodeStats, "indices.bulk.total_size_in_bytes") + 100
		nodeStats["indices.bulk.total_operations"] = getValue(&nodeStats, "indices.bulk.total_operations") + 50

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
		require.EqualValues(t, 1, nodeStats["ingest_docs_per_second"])
		require.EqualValues(t, 2, nodeStats["ingest_bytes_per_second"])
		require.EqualValues(t, 10, nodeStats["bulk_bytes_per_second"])
		require.EqualValues(t, 5, nodeStats["bulk_operations_per_second"])

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

func makeCgroupNodeStats(usageNanos, periods, quotaMicros int64) mapstr.M {
	return mapstr.M{
		"os": mapstr.M{
			"cgroup": mapstr.M{
				"cpuacct": mapstr.M{
					"usage_nanos": usageNanos,
				},
				"cpu": mapstr.M{
					"cfs_quota_micros": quotaMicros,
					"stat": mapstr.M{
						"number_of_elapsed_periods": periods,
					},
				},
			},
		},
	}
}

// makeCgroupWithout returns a cgroup mapstr.M with the given key path deleted.
func makeCgroupWithout(usageNanos, periods, quotaMicros int64, deletePath string) mapstr.M {
	m := makeCgroupNodeStats(usageNanos, periods, quotaMicros)
	_ = m.Delete(deletePath)
	return m
}

func TestEnrichNodeStatsCgroupCpuUsagePercent(t *testing.T) {
	tests := map[string]struct {
		prev          mapstr.M // if nil, cache is cleared (first sample)
		curr          mapstr.M
		expectPresent bool
		expected      float64
	}{
		"happy path 50%": {
			// Δusage=500_000_000 ns, Δperiods=10, quota=100_000 µs → 50%
			prev:          makeCgroupNodeStats(1_000_000_000, 90, 100_000),
			curr:          makeCgroupNodeStats(1_500_000_000, 100, 100_000),
			expectPresent: true,
			expected:      50.0,
		},
		"burst above quota emits >100%": {
			// Δusage=3_000_000_000 ns, Δperiods=10, quota=100_000 µs → 300%
			prev:          makeCgroupNodeStats(1_000_000_000, 90, 100_000),
			curr:          makeCgroupNodeStats(4_000_000_000, 100, 100_000),
			expectPresent: true,
			expected:      300.0,
		},
		"zero usage delta emits 0%": {
			prev:          makeCgroupNodeStats(1_000_000_000, 90, 100_000),
			curr:          makeCgroupNodeStats(1_000_000_000, 100, 100_000),
			expectPresent: true,
			expected:      0.0,
		},
		"quota changed uses current quota": {
			// Even if prev had a different quota, current sample's quota is authoritative.
			// Δusage=500_000_000, Δperiods=10, current quota=200_000 → 25%
			prev:          makeCgroupNodeStats(1_000_000_000, 90, 100_000),
			curr:          makeCgroupNodeStats(1_500_000_000, 100, 200_000),
			expectPresent: true,
			expected:      25.0,
		},
		"unlimited quota (-1)": {
			prev:          makeCgroupNodeStats(1_000_000_000, 90, -1),
			curr:          makeCgroupNodeStats(1_500_000_000, 100, -1),
			expectPresent: false,
		},
		"zero quota": {
			prev:          makeCgroupNodeStats(1_000_000_000, 90, 0),
			curr:          makeCgroupNodeStats(1_500_000_000, 100, 0),
			expectPresent: false,
		},
		"first sample no previous": {
			prev:          nil,
			curr:          makeCgroupNodeStats(1_500_000_000, 100, 100_000),
			expectPresent: false,
		},
		"zero periods delta": {
			prev:          makeCgroupNodeStats(1_000_000_000, 100, 100_000),
			curr:          makeCgroupNodeStats(1_500_000_000, 100, 100_000),
			expectPresent: false,
		},
		"usage counter reset (negative delta)": {
			prev:          makeCgroupNodeStats(2_000_000_000, 90, 100_000),
			curr:          makeCgroupNodeStats(500_000_000, 100, 100_000),
			expectPresent: false,
		},
		"periods counter reset (negative delta)": {
			prev:          makeCgroupNodeStats(1_000_000_000, 100, 100_000),
			curr:          makeCgroupNodeStats(1_500_000_000, 50, 100_000),
			expectPresent: false,
		},
		"missing usage_nanos on previous": {
			prev:          makeCgroupWithout(1_000_000_000, 90, 100_000, "os.cgroup.cpuacct.usage_nanos"),
			curr:          makeCgroupNodeStats(1_500_000_000, 100, 100_000),
			expectPresent: false,
		},
		"missing usage_nanos on current": {
			prev:          makeCgroupNodeStats(1_000_000_000, 90, 100_000),
			curr:          makeCgroupWithout(1_500_000_000, 100, 100_000, "os.cgroup.cpuacct.usage_nanos"),
			expectPresent: false,
		},
		"missing periods on previous": {
			prev:          makeCgroupWithout(1_000_000_000, 90, 100_000, "os.cgroup.cpu.stat.number_of_elapsed_periods"),
			curr:          makeCgroupNodeStats(1_500_000_000, 100, 100_000),
			expectPresent: false,
		},
		"missing periods on current": {
			prev:          makeCgroupNodeStats(1_000_000_000, 90, 100_000),
			curr:          makeCgroupWithout(1_500_000_000, 100, 100_000, "os.cgroup.cpu.stat.number_of_elapsed_periods"),
			expectPresent: false,
		},
		"missing cfs_quota_micros on current": {
			prev:          makeCgroupNodeStats(1_000_000_000, 90, 100_000),
			curr:          makeCgroupWithout(1_500_000_000, 100, 100_000, "os.cgroup.cpu.cfs_quota_micros"),
			expectPresent: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.prev != nil {
				initCache(map[string]mapstr.M{"n1": tt.prev}, 10)
			} else {
				clearCache()
			}
			curr := tt.curr

			enrichNodeStats("n1", &curr, 10_000)

			if tt.expectPresent {
				require.InDelta(t, tt.expected, curr[cgroupCpuPercentField], 0.001)
			} else {
				require.Nil(t, curr[cgroupCpuPercentField])
			}
		})
	}
}

// TestEnrichNodeStatsCgroupCpuUsagePercentQuickCheck fuzzes the enricher with
// randomly-generated positive counter deltas and quota. For any valid input
// the emitted percent must be finite and non-negative.
func TestEnrichNodeStatsCgroupCpuUsagePercentQuickCheck(t *testing.T) {
	property := func(prevUsage uint32, usageDelta uint32, prevPeriods uint16, periodsDelta uint16, quotaMicros uint32) bool {
		// Guarantee the two counter deltas and quota are strictly positive so
		// the enricher can compute a value. Otherwise we assert only the skip.
		periodsDelta = (periodsDelta % 1000) + 1 // 1..1000
		quotaMicros = (quotaMicros % 500_000) + 1
		prev := makeCgroupNodeStats(int64(prevUsage), int64(prevPeriods), int64(quotaMicros))
		curr := makeCgroupNodeStats(int64(prevUsage)+int64(usageDelta), int64(prevPeriods)+int64(periodsDelta), int64(quotaMicros))
		initCache(map[string]mapstr.M{"n1": prev}, 10)

		enrichNodeStats("n1", &curr, 10_000)

		v, ok := curr[cgroupCpuPercentField].(float64)
		if !ok {
			return false
		}
		return v >= 0 && !math.IsNaN(v) && !math.IsInf(v, 0)
	}
	require.NoError(t, quick.Check(property, nil))
}

func TestEnrichNodeStatsGaugeDecreaseWritesNilIngestRate(t *testing.T) {
	initCache(getNodeStats(), 10)

	nodeStatsMap := getNodeStats()
	for key, nodeStats := range nodeStatsMap {
		nodeStats["indices.docs.count"] = getValue(&nodeStats, "indices.docs.count") - 10
		nodeStats["indices.store.size_in_bytes"] = getValue(&nodeStats, "indices.store.size_in_bytes") - 20
		nodeStats["indices.bulk.total_operations"] = getValue(&nodeStats, "indices.bulk.total_operations") + 50
		nodeStats["indices.bulk.total_size_in_bytes"] = getValue(&nodeStats, "indices.bulk.total_size_in_bytes") + 100
		nodeStatsMap[key] = nodeStats
	}

	for key, nodeStats := range nodeStatsMap {
		enrichNodeStats(key, &nodeStats, 10_000)
		// gauge-backed rates are nil on decrease (not written)
		require.Nil(t, nodeStats["ingest_docs_per_second"])
		require.Nil(t, nodeStats["ingest_bytes_per_second"])
		// counter-backed rates still report correctly
		require.EqualValues(t, 5, nodeStats["bulk_operations_per_second"])
		require.EqualValues(t, 10, nodeStats["bulk_bytes_per_second"])
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
		nodeStats["indices.docs.count"] = getValue(&nodeStats, "indices.docs.count") + 10
		nodeStats["indices.store.size_in_bytes"] = getValue(&nodeStats, "indices.store.size_in_bytes") + 20
		nodeStats["indices.bulk.total_size_in_bytes"] = getValue(&nodeStats, "indices.bulk.total_size_in_bytes") + 100
		nodeStats["indices.bulk.total_operations"] = getValue(&nodeStats, "indices.bulk.total_operations") + 50

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
			require.EqualValues(t, 1, nodeStats["ingest_docs_per_second"])
			require.EqualValues(t, 2, nodeStats["ingest_bytes_per_second"])
			require.EqualValues(t, 10, nodeStats["bulk_bytes_per_second"])
			require.EqualValues(t, 5, nodeStats["bulk_operations_per_second"])
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
			require.Nil(t, nodeStats["ingest_docs_per_second"])
			require.Nil(t, nodeStats["ingest_bytes_per_second"])
			require.Nil(t, nodeStats["bulk_bytes_per_second"])
			require.Nil(t, nodeStats["bulk_operations_per_second"])
			require.Nil(t, nodeStats["index_latency_in_millis"])
			require.Nil(t, nodeStats["merge_latency_in_millis"])
			require.Nil(t, nodeStats["search_latency_in_millis"])
		}
	}
}
