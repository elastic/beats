// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package frozen_cache_stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func clearCache() {
	frozenCacheStatsCache.PreviousCache = nil
	frozenCacheStatsCache.PreviousTimestamp = 0
}

func initCache(previousCache map[string]mapstr.M, previousSeconds int64) {
	frozenCacheStatsCache.NewTimestamp = time.Now().UnixMilli()
	frozenCacheStatsCache.PreviousCache = previousCache
	frozenCacheStatsCache.PreviousTimestamp = frozenCacheStatsCache.NewTimestamp - (previousSeconds * 1_000)
}

func getNodeCacheStatsForNode(reads, writes, evictions, bytesWritten, bytesRead int64) mapstr.M {
	return mapstr.M{
		"shared_cache": mapstr.M{
			"reads":                  reads,
			"writes":                 writes,
			"evictions":              evictions,
			"bytes_written_in_bytes": bytesWritten,
			"bytes_read_in_bytes":    bytesRead,
		},
	}
}

func getDefaultNodeCacheStats() map[string]mapstr.M {
	return map[string]mapstr.M{
		"node1": getNodeCacheStatsForNode(1000, 500, 10, 102400, 204800),
		"node2": getNodeCacheStatsForNode(2000, 1000, 20, 204800, 409600),
	}
}

func TestEnrichNodeCacheStatsWithoutCache(t *testing.T) {
	clearCache()

	stats := getNodeCacheStatsForNode(1000, 500, 10, 102400, 204800)
	enrichNodeCacheStats("node1", &stats, 0)

	require.Nil(t, stats["shared_cache_reads_per_second"])
	require.Nil(t, stats["shared_cache_writes_per_second"])
	require.Nil(t, stats["shared_cache_evictions_per_second"])
	require.Nil(t, stats["shared_cache_bytes_written_per_second"])
	require.Nil(t, stats["shared_cache_bytes_read_per_second"])
}

func TestEnrichNodeCacheStatsWithoutCachedValues(t *testing.T) {
	initCache(map[string]mapstr.M{}, 10)

	stats := getNodeCacheStatsForNode(1000, 500, 10, 102400, 204800)
	enrichNodeCacheStats("node1", &stats, 0)

	require.Nil(t, stats["shared_cache_reads_per_second"])
	require.Nil(t, stats["shared_cache_writes_per_second"])
	require.Nil(t, stats["shared_cache_evictions_per_second"])
	require.Nil(t, stats["shared_cache_bytes_written_per_second"])
	require.Nil(t, stats["shared_cache_bytes_read_per_second"])
}

func TestEnrichNodeCacheStatsWithCachedValues(t *testing.T) {
	initCache(getDefaultNodeCacheStats(), 10)

	current := map[string]mapstr.M{
		"node1": getNodeCacheStatsForNode(1000+200, 500+100, 10+5, 102400+20480, 204800+40960),
		"node2": getNodeCacheStatsForNode(2000+200, 1000+100, 20+5, 204800+20480, 409600+40960),
	}

	for key, stats := range current {
		enrichNodeCacheStats(key, &stats, 10_000)
		current[key] = stats
	}

	for _, stats := range current {
		require.EqualValues(t, 20, stats["shared_cache_reads_per_second"])
		require.EqualValues(t, 10, stats["shared_cache_writes_per_second"])
		require.EqualValues(t, 0.5, stats["shared_cache_evictions_per_second"])
		require.EqualValues(t, 2048, stats["shared_cache_bytes_written_per_second"])
		require.EqualValues(t, 4096, stats["shared_cache_bytes_read_per_second"])
	}
}

func TestEnrichNodeCacheStatsWithCachedValuesWithNoChange(t *testing.T) {
	initCache(getDefaultNodeCacheStats(), 10)

	current := getDefaultNodeCacheStats()

	for key, stats := range current {
		enrichNodeCacheStats(key, &stats, 10_000)
		current[key] = stats
	}

	for _, stats := range current {
		require.EqualValues(t, 0, stats["shared_cache_reads_per_second"])
		require.EqualValues(t, 0, stats["shared_cache_writes_per_second"])
		require.EqualValues(t, 0, stats["shared_cache_evictions_per_second"])
		require.EqualValues(t, 0, stats["shared_cache_bytes_written_per_second"])
		require.EqualValues(t, 0, stats["shared_cache_bytes_read_per_second"])
	}
}

func TestEnrichNodeCacheStatsNewNodeHasNoRates(t *testing.T) {
	initCache(getDefaultNodeCacheStats(), 10)

	stats := getNodeCacheStatsForNode(500, 250, 5, 51200, 102400)
	enrichNodeCacheStats("node3", &stats, 10_000)

	require.Nil(t, stats["shared_cache_reads_per_second"])
	require.Nil(t, stats["shared_cache_writes_per_second"])
	require.Nil(t, stats["shared_cache_evictions_per_second"])
	require.Nil(t, stats["shared_cache_bytes_written_per_second"])
	require.Nil(t, stats["shared_cache_bytes_read_per_second"])
}
