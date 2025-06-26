// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cat_shards

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func clearCache() {
	cache.PreviousCache = nil
	cache.PreviousTimestamp = 0
}

func initCache(previousCache map[string]NodeIndexShards, previousSeconds int64) {
	cache.NewTimestamp = time.Now().UnixMilli()

	cache.PreviousCache = previousCache
	cache.PreviousTimestamp = cache.NewTimestamp - (previousSeconds * 1_000)
}

func getUnassignedShard(shardId int32, primary bool) Shard {
	return Shard{
		node_id:   UNASSIGNED,
		node_name: UNASSIGNED,
		shard:     shardId,
		primary:   primary,
		state:     UNASSIGNED,
	}
}

func getShard(nodeId string, nodeName string, shardId int32, primary bool, state string, index int) Shard {
	shard := Shard{
		node_id:   nodeId,
		node_name: nodeName,
		shard:     shardId,
		primary:   primary,
		state:     state,
	}

	docs := []int64{1, 2, 3, 4}
	store := []int64{10, 11, 12, 13}
	segments_count := []int64{20, 21, 22, 23}
	search_query_total := []int64{30, 31, 32, 33}
	search_query_time := []int64{40, 41, 42, 43}
	indexing_index_total := []int64{50, 51, 52, 53}
	indexing_index_time := []int64{60, 61, 62, 63}
	indexing_index_failed := []int64{70, 71, 72, 73}
	merges_total := []int64{80, 81, 82, 83}
	merges_total_time := []int64{90, 91, 92, 93}
	get_missing_time := []int64{100, 101, 102, 103}
	get_missing_total := []int64{110, 111, 112, 113}

	shard.docs = &docs[index]
	shard.store = &store[index]
	shard.segments_count = &segments_count[index]
	shard.search_query_total = &search_query_total[index]
	shard.search_query_time = &search_query_time[index]
	shard.indexing_index_total = &indexing_index_total[index]
	shard.indexing_index_time = &indexing_index_time[index]
	shard.indexing_index_failed = &indexing_index_failed[index]
	shard.merges_total = &merges_total[index]
	shard.merges_total_time = &merges_total_time[index]
	shard.get_missing_time = &get_missing_time[index]
	shard.get_missing_total = &get_missing_total[index]

	return shard
}

func getNodeIndexShards() map[string]NodeIndexShards {
	getMissingDocTotals := []int64{1, 2}
	indexingIndexTotal := []int64{10, 11}
	indexingFailedIndexTotal := []int64{20, 21}
	mergesTotal := []int64{30, 31}
	searchQueryTotal := []int64{40, 41}
	indexingIndexTotalTime := []int64{50, 51}
	mergesTotalTime := []int64{60, 61}
	searchQueryTime := []int64{70, 71}

	return map[string]NodeIndexShards{
		"my-index-node_id-node1": {
			Index:                    "my-index",
			IndexNode:                "my-index-node_id-node1",
			NodeId:                   "node1",
			GetMissingDocTotal:       &getMissingDocTotals[0],
			IndexingIndexTotal:       &indexingIndexTotal[0],
			IndexingIndexTotalTime:   &indexingIndexTotalTime[0],
			IndexingFailedIndexTotal: &indexingFailedIndexTotal[0],
			MergesTotalTime:          &mergesTotalTime[0],
			MergesTotal:              &mergesTotal[0],
			SearchQueryTime:          &searchQueryTime[0],
			SearchQueryTotal:         &searchQueryTotal[0],
		},
		"my-index-node_id-node2": {
			Index:                    "my-index",
			IndexNode:                "my-index-node_id-node2",
			NodeId:                   "node2",
			GetMissingDocTotal:       &getMissingDocTotals[1],
			IndexingIndexTotal:       &indexingIndexTotal[1],
			IndexingIndexTotalTime:   &indexingIndexTotalTime[1],
			IndexingFailedIndexTotal: &indexingFailedIndexTotal[1],
			MergesTotalTime:          &mergesTotalTime[1],
			MergesTotal:              &mergesTotal[1],
			SearchQueryTime:          &searchQueryTime[1],
			SearchQueryTotal:         &searchQueryTotal[1],
		},
	}
}

func getIndexMetadata() map[string]IndexMetadata {
	return map[string]IndexMetadata{
		"my-index": {
			aliases:    []string{"alias1", "alias2"},
			attributes: []string{"attribute1"},
			indexType:  "index",
			hidden:     false,
			open:       true,
			system:     false,
		},
	}
}

func findNodeIndexShards(t *testing.T, nodeIndexShardsList []NodeIndexShards, indexNode string) NodeIndexShards {
	for _, nodeIndexShards := range nodeIndexShardsList {
		if nodeIndexShards.IndexNode == indexNode {
			return nodeIndexShards
		}
	}

	t.Fatalf("Unable to find NodeIndexShards for %v", indexNode)
	return NodeIndexShards{}
}

func TestEnrichNodeIndexShardsWithoutCache(t *testing.T) {
	clearCache()

	nodeIndexShardsMap := getNodeIndexShards()
	indexMetadata := map[string]IndexMetadata{}
	nodeIndexShardsList := enrichNodeIndexShards(nodeIndexShardsMap, indexMetadata)

	require.Equal(t, len(nodeIndexShardsMap), len(nodeIndexShardsList))

	for _, nodeIndexShards := range nodeIndexShardsList {
		require.EqualValues(t, len(nodeIndexShardsList), nodeIndexShards.TotalFractions)

		// rates / latencies from new node unknown for one pass
		require.Nil(t, nodeIndexShards.GetMissingDocRatePerSecond)
		require.Nil(t, nodeIndexShards.IndexRatePerSecond)
		require.Nil(t, nodeIndexShards.IndexFailedRatePerSecond)
		require.Nil(t, nodeIndexShards.MergeRatePerSecond)
		require.Nil(t, nodeIndexShards.SearchRatePerSecond)
		require.Nil(t, nodeIndexShards.IndexLatencyInMillis)
		require.Nil(t, nodeIndexShards.MergeLatencyInMillis)
		require.Nil(t, nodeIndexShards.SearchLatencyInMillis)
		// unknown index metadata
		require.Nil(t, nodeIndexShards.Aliases)
		require.Nil(t, nodeIndexShards.Attributes)
		require.Nil(t, nodeIndexShards.IndexType)
		require.Nil(t, nodeIndexShards.IsHidden)
		require.Nil(t, nodeIndexShards.IsOpen)
		require.Nil(t, nodeIndexShards.IsSystem)
	}
}

func TestEnrichNodeIndexShardsWithoutCachedValues(t *testing.T) {
	// empty, but not nil cache; 10s ago
	initCache(map[string]NodeIndexShards{}, 10)

	nodeIndexShardsMap := getNodeIndexShards()
	indexMetadata := map[string]IndexMetadata{}
	nodeIndexShardsList := enrichNodeIndexShards(nodeIndexShardsMap, indexMetadata)

	require.Equal(t, len(nodeIndexShardsMap), len(nodeIndexShardsList))

	for _, nodeIndexShards := range nodeIndexShardsList {
		require.EqualValues(t, len(nodeIndexShardsList), nodeIndexShards.TotalFractions)

		// rates / latencies from new node unknown for one pass
		require.Nil(t, nodeIndexShards.GetMissingDocRatePerSecond)
		require.Nil(t, nodeIndexShards.IndexRatePerSecond)
		require.Nil(t, nodeIndexShards.IndexFailedRatePerSecond)
		require.Nil(t, nodeIndexShards.MergeRatePerSecond)
		require.Nil(t, nodeIndexShards.SearchRatePerSecond)
		require.Nil(t, nodeIndexShards.IndexLatencyInMillis)
		require.Nil(t, nodeIndexShards.MergeLatencyInMillis)
		require.Nil(t, nodeIndexShards.SearchLatencyInMillis)
		// unknown index metadata
		require.Nil(t, nodeIndexShards.Aliases)
		require.Nil(t, nodeIndexShards.Attributes)
		require.Nil(t, nodeIndexShards.IndexType)
		require.Nil(t, nodeIndexShards.IsHidden)
		require.Nil(t, nodeIndexShards.IsOpen)
		require.Nil(t, nodeIndexShards.IsSystem)
	}
}

func TestEnrichNodeIndexShardsWithCachedValues(t *testing.T) {
	// 10s ago cache
	initCache(getNodeIndexShards(), 10)

	indexMetadata := getIndexMetadata()
	nodeIndexShardsMap := getNodeIndexShards()

	for key, nodeIndexShards := range nodeIndexShardsMap {
		*nodeIndexShards.GetMissingDocTotal += 10
		*nodeIndexShards.IndexingIndexTotal += 20
		*nodeIndexShards.IndexingIndexTotalTime += 10
		*nodeIndexShards.IndexingFailedIndexTotal += 30
		*nodeIndexShards.MergesTotal += 40
		*nodeIndexShards.MergesTotalTime += 40
		*nodeIndexShards.SearchQueryTotal += 60
		*nodeIndexShards.SearchQueryTime += 120

		nodeIndexShardsMap[key] = nodeIndexShards
	}

	nodeIndexShardsList := enrichNodeIndexShards(nodeIndexShardsMap, indexMetadata)

	require.Equal(t, len(nodeIndexShardsMap), len(nodeIndexShardsList))

	for _, nodeIndexShards := range nodeIndexShardsList {
		require.EqualValues(t, len(nodeIndexShardsList), nodeIndexShards.TotalFractions)

		// rates
		require.EqualValues(t, 1, *nodeIndexShards.GetMissingDocRatePerSecond)
		require.EqualValues(t, 2, *nodeIndexShards.IndexRatePerSecond)
		require.EqualValues(t, 3, *nodeIndexShards.IndexFailedRatePerSecond)
		require.EqualValues(t, 4, *nodeIndexShards.MergeRatePerSecond)
		require.EqualValues(t, 6, *nodeIndexShards.SearchRatePerSecond)
		// latencies
		require.EqualValues(t, 0.5, *nodeIndexShards.IndexLatencyInMillis)
		require.EqualValues(t, 1, *nodeIndexShards.MergeLatencyInMillis)
		require.EqualValues(t, 2, *nodeIndexShards.SearchLatencyInMillis)
		// index metadata
		metadata := indexMetadata[nodeIndexShards.Index]

		require.ElementsMatch(t, metadata.aliases, nodeIndexShards.Aliases)
		require.ElementsMatch(t, metadata.attributes, nodeIndexShards.Attributes)
		require.Equal(t, metadata.indexType, *nodeIndexShards.IndexType)
		require.Equal(t, metadata.hidden, *nodeIndexShards.IsHidden)
		require.Equal(t, metadata.open, *nodeIndexShards.IsOpen)
		require.Equal(t, metadata.system, *nodeIndexShards.IsSystem)
	}
}

func TestEnrichNodeIndexShardsWithCachedValuesWithNoChange(t *testing.T) {
	// 10s ago cache
	initCache(getNodeIndexShards(), 10)

	indexMetadata := getIndexMetadata()
	nodeIndexShardsMap := getNodeIndexShards()
	nodeIndexShardsList := enrichNodeIndexShards(nodeIndexShardsMap, indexMetadata)

	require.Equal(t, len(nodeIndexShardsMap), len(nodeIndexShardsList))

	for _, nodeIndexShards := range nodeIndexShardsList {
		require.EqualValues(t, len(nodeIndexShardsList), nodeIndexShards.TotalFractions)

		// rates
		require.EqualValues(t, 0, *nodeIndexShards.GetMissingDocRatePerSecond)
		require.EqualValues(t, 0, *nodeIndexShards.IndexRatePerSecond)
		require.EqualValues(t, 0, *nodeIndexShards.IndexFailedRatePerSecond)
		require.EqualValues(t, 0, *nodeIndexShards.MergeRatePerSecond)
		require.EqualValues(t, 0, *nodeIndexShards.SearchRatePerSecond)
		// latencies
		require.EqualValues(t, 0, *nodeIndexShards.IndexLatencyInMillis)
		require.EqualValues(t, 0, *nodeIndexShards.MergeLatencyInMillis)
		require.EqualValues(t, 0, *nodeIndexShards.SearchLatencyInMillis)
		// index metadata
		metadata := indexMetadata[nodeIndexShards.Index]

		require.ElementsMatch(t, metadata.aliases, nodeIndexShards.Aliases)
		require.ElementsMatch(t, metadata.attributes, nodeIndexShards.Attributes)
		require.Equal(t, metadata.indexType, *nodeIndexShards.IndexType)
		require.Equal(t, metadata.hidden, *nodeIndexShards.IsHidden)
		require.Equal(t, metadata.open, *nodeIndexShards.IsOpen)
		require.Equal(t, metadata.system, *nodeIndexShards.IsSystem)
	}
}

func TestEnrichNodeIndexShardsWithCachedValuesWithHoles(t *testing.T) {
	// 10s ago cache
	initCache(getNodeIndexShards(), 10)

	indexMetadata := getIndexMetadata()
	nodeIndexShardsMap := getNodeIndexShards()

	for key, nodeIndexShards := range nodeIndexShardsMap {
		if nodeIndexShards.NodeId == "node2" {
			*nodeIndexShards.GetMissingDocTotal += 10
			*nodeIndexShards.IndexingIndexTotal += 20
			*nodeIndexShards.IndexingIndexTotalTime += 10
		} else {
			nodeIndexShards.GetMissingDocTotal = nil
			nodeIndexShards.IndexingIndexTotal = nil
			nodeIndexShards.IndexingIndexTotalTime = nil
		}

		*nodeIndexShards.IndexingFailedIndexTotal += 30
		*nodeIndexShards.MergesTotal += 40
		*nodeIndexShards.MergesTotalTime += 40
		*nodeIndexShards.SearchQueryTotal += 60
		*nodeIndexShards.SearchQueryTime += 120

		nodeIndexShardsMap[key] = nodeIndexShards
	}

	nodeIndexShardsList := enrichNodeIndexShards(nodeIndexShardsMap, indexMetadata)

	require.Equal(t, len(nodeIndexShardsMap), len(nodeIndexShardsList))

	for _, nodeIndexShards := range nodeIndexShardsList {
		require.EqualValues(t, len(nodeIndexShardsList), nodeIndexShards.TotalFractions)

		// rates
		if nodeIndexShards.NodeId == "node2" {
			require.EqualValues(t, 1, *nodeIndexShards.GetMissingDocRatePerSecond)
			require.EqualValues(t, 2, *nodeIndexShards.IndexRatePerSecond)
		} else {
			require.Nil(t, nodeIndexShards.GetMissingDocRatePerSecond)
			require.Nil(t, nodeIndexShards.IndexRatePerSecond)
		}

		require.EqualValues(t, 3, *nodeIndexShards.IndexFailedRatePerSecond)
		require.EqualValues(t, 4, *nodeIndexShards.MergeRatePerSecond)
		require.EqualValues(t, 6, *nodeIndexShards.SearchRatePerSecond)
		// latencies
		if nodeIndexShards.NodeId == "node2" {
			require.EqualValues(t, 0.5, *nodeIndexShards.IndexLatencyInMillis)
		} else {
			require.Nil(t, nodeIndexShards.IndexLatencyInMillis)
		}
		require.EqualValues(t, 1, *nodeIndexShards.MergeLatencyInMillis)
		require.EqualValues(t, 2, *nodeIndexShards.SearchLatencyInMillis)

		// index metadata
		metadata := indexMetadata[nodeIndexShards.Index]

		require.ElementsMatch(t, metadata.aliases, nodeIndexShards.Aliases)
		require.ElementsMatch(t, metadata.attributes, nodeIndexShards.Attributes)
		require.Equal(t, metadata.indexType, *nodeIndexShards.IndexType)
		require.Equal(t, metadata.hidden, *nodeIndexShards.IsHidden)
		require.Equal(t, metadata.open, *nodeIndexShards.IsOpen)
		require.Equal(t, metadata.system, *nodeIndexShards.IsSystem)
	}
}

func TestEnrichNodeIndexShardsWithCachedValuesWithNewNodeAndIndex(t *testing.T) {
	// 10s ago cache
	initCache(getNodeIndexShards(), 10)

	indexMetadata := getIndexMetadata()
	nodeIndexShardsMap := getNodeIndexShards()

	for key, nodeIndexShards := range nodeIndexShardsMap {
		*nodeIndexShards.GetMissingDocTotal += 10
		*nodeIndexShards.IndexingIndexTotal += 20
		*nodeIndexShards.IndexingIndexTotalTime += 10
		*nodeIndexShards.IndexingFailedIndexTotal += 30
		*nodeIndexShards.MergesTotal += 40
		*nodeIndexShards.MergesTotalTime += 40
		*nodeIndexShards.SearchQueryTotal += 60
		*nodeIndexShards.SearchQueryTime += 120

		nodeIndexShardsMap[key] = nodeIndexShards
	}

	newNode := getNodeIndexShards()["my-index-node_id-node2"]

	newNode.Index = "my-other-index"
	newNode.NodeId = "node3"
	newNode.IndexNode = "my-other-index-node_id-node3"

	nodeIndexShardsMap["my-other-index-node_id-node3"] = newNode

	nodeIndexShardsList := enrichNodeIndexShards(nodeIndexShardsMap, indexMetadata)

	require.Equal(t, len(nodeIndexShardsMap), len(nodeIndexShardsList))

	for _, nodeIndexShards := range nodeIndexShardsList {
		require.EqualValues(t, len(nodeIndexShardsList), nodeIndexShards.TotalFractions)

		if nodeIndexShards.NodeId != "node3" {
			// rates
			require.EqualValues(t, 1, *nodeIndexShards.GetMissingDocRatePerSecond)
			require.EqualValues(t, 2, *nodeIndexShards.IndexRatePerSecond)
			require.EqualValues(t, 3, *nodeIndexShards.IndexFailedRatePerSecond)
			require.EqualValues(t, 4, *nodeIndexShards.MergeRatePerSecond)
			require.EqualValues(t, 6, *nodeIndexShards.SearchRatePerSecond)
			// latencies
			require.EqualValues(t, 0.5, *nodeIndexShards.IndexLatencyInMillis)
			require.EqualValues(t, 1, *nodeIndexShards.MergeLatencyInMillis)
			require.EqualValues(t, 2, *nodeIndexShards.SearchLatencyInMillis)
			// index metadata
			metadata := indexMetadata[nodeIndexShards.Index]

			require.ElementsMatch(t, metadata.aliases, nodeIndexShards.Aliases)
			require.ElementsMatch(t, metadata.attributes, nodeIndexShards.Attributes)
			require.Equal(t, metadata.indexType, *nodeIndexShards.IndexType)
			require.Equal(t, metadata.hidden, *nodeIndexShards.IsHidden)
			require.Equal(t, metadata.open, *nodeIndexShards.IsOpen)
			require.Equal(t, metadata.system, *nodeIndexShards.IsSystem)
		} else {
			// rates / latencies from new node unknown for one pass
			require.Nil(t, nodeIndexShards.GetMissingDocRatePerSecond)
			require.Nil(t, nodeIndexShards.IndexRatePerSecond)
			require.Nil(t, nodeIndexShards.IndexFailedRatePerSecond)
			require.Nil(t, nodeIndexShards.MergeRatePerSecond)
			require.Nil(t, nodeIndexShards.SearchRatePerSecond)
			require.Nil(t, nodeIndexShards.IndexLatencyInMillis)
			require.Nil(t, nodeIndexShards.MergeLatencyInMillis)
			require.Nil(t, nodeIndexShards.SearchLatencyInMillis)
			// unknown index metadata
			require.Nil(t, nodeIndexShards.Aliases)
			require.Nil(t, nodeIndexShards.Attributes)
			require.Nil(t, nodeIndexShards.IndexType)
			require.Nil(t, nodeIndexShards.IsHidden)
			require.Nil(t, nodeIndexShards.IsOpen)
			require.Nil(t, nodeIndexShards.IsSystem)
		}
	}
}

func TestConvertToNodeIndexShardsReturnsEmpty(t *testing.T) {
	clearCache()

	nodeIndexShards := convertToNodeIndexShards(map[string][]Shard{}, map[string]IndexMetadata{})

	require.Equal(t, 0, len(nodeIndexShards))
	require.NotNil(t, cache.PreviousCache)
	require.Equal(t, cache.PreviousTimestamp, cache.NewTimestamp)
}

func TestConvertToNodeIndexShardsUncached(t *testing.T) {
	clearCache()

	indexToShardsList := map[string][]Shard{
		"my-index": { // green-index, but we can reuse the indexMetadata
			getShard("node1", "name1", 0, true, STARTED, 0),
			getShard("node3", "name3", 0, false, STARTED, 1),
		},
		"yellow-index": {
			getShard("node1", "name1", 0, true, STARTED, 0),
			getShard("node2", "name2", 0, false, STARTED, 1),
			getShard("node2", "name2", 1, true, STARTED, 2),
			getUnassignedShard(1, false),
		},
		"red-index": {
			getUnassignedShard(0, true),
			getUnassignedShard(0, false),
		},
	}

	nodeIndexShards := convertToNodeIndexShards(indexToShardsList, getIndexMetadata())

	// 3 indexes on 3 total nodes (and 3 unassigned shards)
	require.Equal(t, 6, len(nodeIndexShards))

	greenNode1 := findNodeIndexShards(t, nodeIndexShards, "my-index-node_id-node1")

	require.EqualValues(t, 6, greenNode1.TotalFractions)
	require.Equal(t, "my-index", greenNode1.Index)
	require.Equal(t, GREEN, *greenNode1.IndexStatus)
	require.Equal(t, "index", *greenNode1.IndexType)
	require.ElementsMatch(t, []string{"alias1", "alias2"}, greenNode1.Aliases)
	require.ElementsMatch(t, []string{"attribute1"}, greenNode1.Attributes)
	require.Equal(t, false, *greenNode1.IsHidden)
	require.Equal(t, true, *greenNode1.IsOpen)
	require.Equal(t, false, *greenNode1.IsSystem)
	require.Equal(t, "node1", greenNode1.NodeId)
	require.Equal(t, "name1", greenNode1.NodeName)
	require.ElementsMatch(t, []AssignedShard{toAssignedShard(indexToShardsList["my-index"][0])}, greenNode1.AssignShards)
	require.Equal(t, 0, len(greenNode1.InitializingShards))
	require.Equal(t, 0, len(greenNode1.RelocatingShards))
	require.Equal(t, 0, len(greenNode1.UnassignedShards))
	require.EqualValues(t, 1, greenNode1.Shards)
	require.EqualValues(t, 1, greenNode1.PrimaryShards)
	require.EqualValues(t, 0, greenNode1.ReplicaShards)
	require.EqualValues(t, 0, greenNode1.Initializing)
	require.EqualValues(t, 0, greenNode1.Relocating)
	require.EqualValues(t, 0, greenNode1.Unassigned)
	require.EqualValues(t, 0, greenNode1.UnassignedPrimaryShards)
	require.EqualValues(t, 0, greenNode1.UnassignedReplicasShards)
	require.Equal(t, *indexToShardsList["my-index"][0].segments_count, *greenNode1.SegmentsCount)
	require.Equal(t, *indexToShardsList["my-index"][0].segments_count, *greenNode1.TotalSegmentsCount)
	require.Equal(t, *indexToShardsList["my-index"][0].store, *greenNode1.SizeInBytes)
	require.Equal(t, *indexToShardsList["my-index"][0].store, *greenNode1.TotalSizeInBytes)
	require.Equal(t, *indexToShardsList["my-index"][0].store, *greenNode1.MaxShardSizeInBytes)
	require.Equal(t, *indexToShardsList["my-index"][0].store, *greenNode1.MinShardSizeInBytes)
	require.Equal(t, *indexToShardsList["my-index"][0].store, *greenNode1.TotalMaxShardSizeInBytes)
	require.Equal(t, *indexToShardsList["my-index"][0].store, *greenNode1.TotalMinShardSizeInBytes)
	require.Equal(t, *indexToShardsList["my-index"][0].docs, *greenNode1.DocsCount)
	require.Equal(t, *indexToShardsList["my-index"][0].indexing_index_failed, *greenNode1.IndexingFailedIndexTotal)
	require.Equal(t, *indexToShardsList["my-index"][0].indexing_index_total, *greenNode1.IndexingIndexTotal)
	require.Equal(t, *indexToShardsList["my-index"][0].indexing_index_time, *greenNode1.IndexingIndexTotalTime)
	require.Equal(t, *indexToShardsList["my-index"][0].get_missing_total, *greenNode1.GetMissingDocTotal)
	require.Equal(t, *indexToShardsList["my-index"][0].get_missing_time, *greenNode1.GetMissingDocTotalTime)
	require.Equal(t, *indexToShardsList["my-index"][0].merges_total, *greenNode1.MergesTotal)
	require.Equal(t, *indexToShardsList["my-index"][0].merges_total_time, *greenNode1.MergesTotalTime)
	require.Equal(t, *indexToShardsList["my-index"][0].search_query_total, *greenNode1.SearchQueryTotal)
	require.Equal(t, *indexToShardsList["my-index"][0].search_query_time, *greenNode1.SearchQueryTime)
	require.Equal(t, *indexToShardsList["my-index"][0].merges_total, *greenNode1.TotalMergesTotal)
	require.Equal(t, *indexToShardsList["my-index"][0].merges_total_time, *greenNode1.TotalMergesTotalTime)
	require.Nil(t, greenNode1.IndexFailedRatePerSecond)
	require.Nil(t, greenNode1.IndexLatencyInMillis)
	require.Nil(t, greenNode1.IndexRatePerSecond)
	require.Nil(t, greenNode1.GetMissingDocRatePerSecond)
	require.Nil(t, greenNode1.MergeLatencyInMillis)
	require.Nil(t, greenNode1.MergeRatePerSecond)
	require.Nil(t, greenNode1.SearchLatencyInMillis)
	require.Nil(t, greenNode1.SearchRatePerSecond)
	require.Nil(t, greenNode1.TimestampDiff)

	// ensure it exists
	findNodeIndexShards(t, nodeIndexShards, "my-index-node_id-node3")
	findNodeIndexShards(t, nodeIndexShards, "yellow-index-node_id-node1")
	findNodeIndexShards(t, nodeIndexShards, "yellow-index-node_id-UNASSIGNED")

	yellowNode2 := findNodeIndexShards(t, nodeIndexShards, "yellow-index-node_id-node2")

	require.EqualValues(t, 6, yellowNode2.TotalFractions)
	require.Equal(t, "yellow-index", yellowNode2.Index)
	require.Equal(t, YELLOW, *yellowNode2.IndexStatus)
	require.Nil(t, yellowNode2.IndexType)
	require.Nil(t, yellowNode2.Aliases)
	require.Nil(t, yellowNode2.Attributes)
	require.Nil(t, yellowNode2.IsHidden)
	require.Nil(t, yellowNode2.IsOpen)
	require.Nil(t, yellowNode2.IsSystem)
	require.Equal(t, "node2", yellowNode2.NodeId)
	require.Equal(t, "name2", yellowNode2.NodeName)
	require.ElementsMatch(t, []AssignedShard{
		toAssignedShard(indexToShardsList["yellow-index"][1]),
		toAssignedShard(indexToShardsList["yellow-index"][2])},
		yellowNode2.AssignShards)
	require.Equal(t, 0, len(yellowNode2.InitializingShards))
	require.Equal(t, 0, len(yellowNode2.RelocatingShards))
	require.Equal(t, 0, len(yellowNode2.UnassignedShards))
	require.EqualValues(t, 2, yellowNode2.Shards)
	require.EqualValues(t, 1, yellowNode2.PrimaryShards)
	require.EqualValues(t, 1, yellowNode2.ReplicaShards)
	require.EqualValues(t, 0, yellowNode2.Initializing)
	require.EqualValues(t, 0, yellowNode2.Relocating)
	require.EqualValues(t, 0, yellowNode2.Unassigned)
	require.EqualValues(t, 0, yellowNode2.UnassignedPrimaryShards)
	require.EqualValues(t, 0, yellowNode2.UnassignedReplicasShards)
	require.Equal(t, *indexToShardsList["yellow-index"][2].segments_count, *yellowNode2.SegmentsCount)
	require.Equal(t, *indexToShardsList["yellow-index"][1].segments_count+*indexToShardsList["yellow-index"][2].segments_count, *yellowNode2.TotalSegmentsCount)
	require.Equal(t, *indexToShardsList["yellow-index"][2].store, *yellowNode2.SizeInBytes)
	require.Equal(t, *indexToShardsList["yellow-index"][1].store+*indexToShardsList["yellow-index"][2].store, *yellowNode2.TotalSizeInBytes)
	require.Equal(t, *indexToShardsList["yellow-index"][2].store, *yellowNode2.MaxShardSizeInBytes)
	require.Equal(t, *indexToShardsList["yellow-index"][2].store, *yellowNode2.MinShardSizeInBytes)
	require.Equal(t, *indexToShardsList["yellow-index"][2].store, *yellowNode2.TotalMaxShardSizeInBytes)
	require.Equal(t, *indexToShardsList["yellow-index"][1].store, *yellowNode2.TotalMinShardSizeInBytes)
	require.Equal(t, *indexToShardsList["yellow-index"][2].docs, *yellowNode2.DocsCount)
	require.Equal(t, *indexToShardsList["yellow-index"][2].indexing_index_failed, *yellowNode2.IndexingFailedIndexTotal)
	require.Equal(t, *indexToShardsList["yellow-index"][2].indexing_index_total, *yellowNode2.IndexingIndexTotal)
	require.Equal(t, *indexToShardsList["yellow-index"][2].indexing_index_time, *yellowNode2.IndexingIndexTotalTime)
	require.Equal(t, *indexToShardsList["yellow-index"][1].get_missing_total+*indexToShardsList["yellow-index"][2].get_missing_total, *yellowNode2.GetMissingDocTotal)
	require.Equal(t, *indexToShardsList["yellow-index"][1].get_missing_time+*indexToShardsList["yellow-index"][2].get_missing_time, *yellowNode2.GetMissingDocTotalTime)
	require.Equal(t, *indexToShardsList["yellow-index"][2].merges_total, *yellowNode2.MergesTotal)
	require.Equal(t, *indexToShardsList["yellow-index"][2].merges_total_time, *yellowNode2.MergesTotalTime)
	require.Equal(t, *indexToShardsList["yellow-index"][1].search_query_total+*indexToShardsList["yellow-index"][2].search_query_total, *yellowNode2.SearchQueryTotal)
	require.Equal(t, *indexToShardsList["yellow-index"][1].search_query_time+*indexToShardsList["yellow-index"][2].search_query_time, *yellowNode2.SearchQueryTime)
	require.Equal(t, *indexToShardsList["yellow-index"][1].merges_total+*indexToShardsList["yellow-index"][2].merges_total, *yellowNode2.TotalMergesTotal)
	require.Equal(t, *indexToShardsList["yellow-index"][1].merges_total_time+*indexToShardsList["yellow-index"][2].merges_total_time, *yellowNode2.TotalMergesTotalTime)
	require.Nil(t, yellowNode2.IndexFailedRatePerSecond)
	require.Nil(t, yellowNode2.IndexLatencyInMillis)
	require.Nil(t, yellowNode2.IndexRatePerSecond)
	require.Nil(t, yellowNode2.GetMissingDocRatePerSecond)
	require.Nil(t, yellowNode2.MergeLatencyInMillis)
	require.Nil(t, yellowNode2.MergeRatePerSecond)
	require.Nil(t, yellowNode2.SearchLatencyInMillis)
	require.Nil(t, yellowNode2.SearchRatePerSecond)
	require.Nil(t, yellowNode2.TimestampDiff)

	redIndex := findNodeIndexShards(t, nodeIndexShards, "red-index-node_id-UNASSIGNED")

	require.EqualValues(t, 6, redIndex.TotalFractions)
	require.Equal(t, "red-index", redIndex.Index)
	require.Equal(t, RED, *redIndex.IndexStatus)
	require.Nil(t, redIndex.IndexType)
	require.Nil(t, redIndex.Aliases)
	require.Nil(t, redIndex.Attributes)
	require.Nil(t, redIndex.IsHidden)
	require.Nil(t, redIndex.IsOpen)
	require.Nil(t, redIndex.IsSystem)
	require.Equal(t, UNASSIGNED, redIndex.NodeId)
	require.Equal(t, UNASSIGNED, redIndex.NodeName)
	require.Equal(t, 0, len(redIndex.AssignShards))
	require.Equal(t, 0, len(redIndex.InitializingShards))
	require.Equal(t, 0, len(redIndex.RelocatingShards))
	require.ElementsMatch(t, []UnassignedShard{
		toUnassignedShard(indexToShardsList["red-index"][0]),
		toUnassignedShard(indexToShardsList["red-index"][1]),
	}, redIndex.UnassignedShards)
	require.EqualValues(t, 2, redIndex.Shards)
	require.EqualValues(t, 1, redIndex.PrimaryShards)
	require.EqualValues(t, 1, redIndex.ReplicaShards)
	require.EqualValues(t, 0, redIndex.Initializing)
	require.EqualValues(t, 0, redIndex.Relocating)
	require.EqualValues(t, 2, redIndex.Unassigned)
	require.EqualValues(t, 1, redIndex.UnassignedPrimaryShards)
	require.EqualValues(t, 1, redIndex.UnassignedReplicasShards)
	require.Nil(t, redIndex.SegmentsCount)
	require.Nil(t, redIndex.TotalSegmentsCount)
	require.Nil(t, redIndex.SizeInBytes)
	require.Nil(t, redIndex.TotalSizeInBytes)
	require.Nil(t, redIndex.MaxShardSizeInBytes)
	require.Nil(t, redIndex.MinShardSizeInBytes)
	require.Nil(t, redIndex.TotalMaxShardSizeInBytes)
	require.Nil(t, redIndex.TotalMinShardSizeInBytes)
	require.Nil(t, redIndex.DocsCount)
	require.Nil(t, redIndex.IndexingFailedIndexTotal)
	require.Nil(t, redIndex.IndexingIndexTotal)
	require.Nil(t, redIndex.IndexingIndexTotalTime)
	require.Nil(t, redIndex.GetMissingDocTotal)
	require.Nil(t, redIndex.GetMissingDocTotalTime)
	require.Nil(t, redIndex.MergesTotal)
	require.Nil(t, redIndex.MergesTotalTime)
	require.Nil(t, redIndex.SearchQueryTotal)
	require.Nil(t, redIndex.SearchQueryTime)
	require.Nil(t, redIndex.TotalMergesTotal)
	require.Nil(t, redIndex.TotalMergesTotalTime)
	require.Nil(t, redIndex.IndexFailedRatePerSecond)
	require.Nil(t, redIndex.IndexLatencyInMillis)
	require.Nil(t, redIndex.IndexRatePerSecond)
	require.Nil(t, redIndex.GetMissingDocRatePerSecond)
	require.Nil(t, redIndex.MergeLatencyInMillis)
	require.Nil(t, redIndex.MergeRatePerSecond)
	require.Nil(t, redIndex.SearchLatencyInMillis)
	require.Nil(t, redIndex.SearchRatePerSecond)
	require.Nil(t, redIndex.TimestampDiff)
}

func TestConvertToNodeIndexShardsWithCache(t *testing.T) {
	initCache(getNodeIndexShards(), 10)

	indexToShardsList := map[string][]Shard{
		"my-index": {
			getShard("node1", "name1", 0, true, STARTED, 0),
			getShard("node3", "name3", 0, false, STARTED, 1),
		},
	}

	nodeIndexShards := convertToNodeIndexShards(indexToShardsList, getIndexMetadata())

	// 1 indexes on 2 total nodes
	require.Equal(t, 2, len(nodeIndexShards))

	myIndexNode1 := findNodeIndexShards(t, nodeIndexShards, "my-index-node_id-node1")

	require.EqualValues(t, 2, myIndexNode1.TotalFractions)
	require.Equal(t, "my-index", myIndexNode1.Index)
	require.Equal(t, GREEN, *myIndexNode1.IndexStatus)
	require.Equal(t, "index", *myIndexNode1.IndexType)
	require.ElementsMatch(t, []string{"alias1", "alias2"}, myIndexNode1.Aliases)
	require.ElementsMatch(t, []string{"attribute1"}, myIndexNode1.Attributes)
	require.Equal(t, false, *myIndexNode1.IsHidden)
	require.Equal(t, true, *myIndexNode1.IsOpen)
	require.Equal(t, false, *myIndexNode1.IsSystem)
	require.Equal(t, "node1", myIndexNode1.NodeId)
	require.Equal(t, "name1", myIndexNode1.NodeName)
	require.ElementsMatch(t, []AssignedShard{toAssignedShard(indexToShardsList["my-index"][0])}, myIndexNode1.AssignShards)
	require.Equal(t, 0, len(myIndexNode1.InitializingShards))
	require.Equal(t, 0, len(myIndexNode1.RelocatingShards))
	require.Equal(t, 0, len(myIndexNode1.UnassignedShards))
	require.EqualValues(t, 1, myIndexNode1.Shards)
	require.EqualValues(t, 1, myIndexNode1.PrimaryShards)
	require.EqualValues(t, 0, myIndexNode1.ReplicaShards)
	require.EqualValues(t, 0, myIndexNode1.Initializing)
	require.EqualValues(t, 0, myIndexNode1.Relocating)
	require.EqualValues(t, 0, myIndexNode1.Unassigned)
	require.EqualValues(t, 0, myIndexNode1.UnassignedPrimaryShards)
	require.EqualValues(t, 0, myIndexNode1.UnassignedReplicasShards)
	require.Equal(t, *indexToShardsList["my-index"][0].segments_count, *myIndexNode1.SegmentsCount)
	require.Equal(t, *indexToShardsList["my-index"][0].segments_count, *myIndexNode1.TotalSegmentsCount)
	require.Equal(t, *indexToShardsList["my-index"][0].store, *myIndexNode1.SizeInBytes)
	require.Equal(t, *indexToShardsList["my-index"][0].store, *myIndexNode1.TotalSizeInBytes)
	require.Equal(t, *indexToShardsList["my-index"][0].store, *myIndexNode1.MaxShardSizeInBytes)
	require.Equal(t, *indexToShardsList["my-index"][0].store, *myIndexNode1.MinShardSizeInBytes)
	require.Equal(t, *indexToShardsList["my-index"][0].store, *myIndexNode1.TotalMaxShardSizeInBytes)
	require.Equal(t, *indexToShardsList["my-index"][0].store, *myIndexNode1.TotalMinShardSizeInBytes)
	require.Equal(t, *indexToShardsList["my-index"][0].docs, *myIndexNode1.DocsCount)
	require.Equal(t, *indexToShardsList["my-index"][0].indexing_index_failed, *myIndexNode1.IndexingFailedIndexTotal)
	require.Equal(t, *indexToShardsList["my-index"][0].indexing_index_total, *myIndexNode1.IndexingIndexTotal)
	require.Equal(t, *indexToShardsList["my-index"][0].indexing_index_time, *myIndexNode1.IndexingIndexTotalTime)
	require.Equal(t, *indexToShardsList["my-index"][0].get_missing_total, *myIndexNode1.GetMissingDocTotal)
	require.Equal(t, *indexToShardsList["my-index"][0].get_missing_time, *myIndexNode1.GetMissingDocTotalTime)
	require.Equal(t, *indexToShardsList["my-index"][0].merges_total, *myIndexNode1.MergesTotal)
	require.Equal(t, *indexToShardsList["my-index"][0].merges_total_time, *myIndexNode1.MergesTotalTime)
	require.Equal(t, *indexToShardsList["my-index"][0].search_query_total, *myIndexNode1.SearchQueryTotal)
	require.Equal(t, *indexToShardsList["my-index"][0].search_query_time, *myIndexNode1.SearchQueryTime)
	require.Equal(t, *indexToShardsList["my-index"][0].merges_total, *myIndexNode1.TotalMergesTotal)
	require.Equal(t, *indexToShardsList["my-index"][0].merges_total_time, *myIndexNode1.TotalMergesTotalTime)
	require.EqualValues(t, 10000, *myIndexNode1.TimestampDiff)
	require.EqualValues(t, 5, *myIndexNode1.IndexFailedRatePerSecond)
	require.EqualValues(t, 0.25, *myIndexNode1.IndexLatencyInMillis)
	require.EqualValues(t, 4, *myIndexNode1.IndexRatePerSecond)
	require.EqualValues(t, 10.9, *myIndexNode1.GetMissingDocRatePerSecond)
	require.EqualValues(t, 0.6, *myIndexNode1.MergeLatencyInMillis)
	require.EqualValues(t, 5, *myIndexNode1.MergeRatePerSecond)
	// note: these are examples of restarted values, so we blank them out rather than calculate negative or massive values
	// if you're interested: compare the `search_query_total` and `search_query_time` values from the cache and this value
	require.Nil(t, myIndexNode1.SearchLatencyInMillis)
	require.Nil(t, myIndexNode1.SearchRatePerSecond)

	// this will be a cache miss because the cache has node2 (so the shard moved)!
	myIndexNode3 := findNodeIndexShards(t, nodeIndexShards, "my-index-node_id-node3")

	require.EqualValues(t, 2, myIndexNode3.TotalFractions)
	require.Equal(t, "my-index", myIndexNode3.Index)
	require.Equal(t, GREEN, *myIndexNode3.IndexStatus)
	require.Equal(t, "index", *myIndexNode3.IndexType)
	require.ElementsMatch(t, []string{"alias1", "alias2"}, myIndexNode3.Aliases)
	require.ElementsMatch(t, []string{"attribute1"}, myIndexNode3.Attributes)
	require.Equal(t, false, *myIndexNode3.IsHidden)
	require.Equal(t, true, *myIndexNode3.IsOpen)
	require.Equal(t, false, *myIndexNode3.IsSystem)
	require.Equal(t, "node3", myIndexNode3.NodeId)
	require.Equal(t, "name3", myIndexNode3.NodeName)
	require.ElementsMatch(t, []AssignedShard{toAssignedShard(indexToShardsList["my-index"][1])}, myIndexNode3.AssignShards)
	require.Equal(t, 0, len(myIndexNode3.InitializingShards))
	require.Equal(t, 0, len(myIndexNode3.RelocatingShards))
	require.Equal(t, 0, len(myIndexNode3.UnassignedShards))
	require.EqualValues(t, 1, myIndexNode3.Shards)
	require.EqualValues(t, 0, myIndexNode3.PrimaryShards)
	require.EqualValues(t, 1, myIndexNode3.ReplicaShards)
	require.EqualValues(t, 0, myIndexNode3.Initializing)
	require.EqualValues(t, 0, myIndexNode3.Relocating)
	require.EqualValues(t, 0, myIndexNode3.Unassigned)
	require.EqualValues(t, 0, myIndexNode3.UnassignedPrimaryShards)
	require.EqualValues(t, 0, myIndexNode3.UnassignedReplicasShards)
	require.Nil(t, myIndexNode3.SegmentsCount)
	require.Equal(t, *indexToShardsList["my-index"][1].segments_count, *myIndexNode3.TotalSegmentsCount)
	require.Nil(t, myIndexNode3.SizeInBytes)
	require.Equal(t, *indexToShardsList["my-index"][1].store, *myIndexNode3.TotalSizeInBytes)
	require.Nil(t, myIndexNode3.MaxShardSizeInBytes)
	require.Nil(t, myIndexNode3.MinShardSizeInBytes)
	require.Equal(t, *indexToShardsList["my-index"][1].store, *myIndexNode3.TotalMaxShardSizeInBytes)
	require.Equal(t, *indexToShardsList["my-index"][1].store, *myIndexNode3.TotalMinShardSizeInBytes)
	require.Nil(t, myIndexNode3.DocsCount)
	require.Nil(t, myIndexNode3.IndexingFailedIndexTotal)
	require.Nil(t, myIndexNode3.IndexingIndexTotal)
	require.Nil(t, myIndexNode3.IndexingIndexTotalTime)
	require.Equal(t, *indexToShardsList["my-index"][1].get_missing_total, *myIndexNode3.GetMissingDocTotal)
	require.Equal(t, *indexToShardsList["my-index"][1].get_missing_time, *myIndexNode3.GetMissingDocTotalTime)
	require.Nil(t, myIndexNode3.MergesTotal)
	require.Nil(t, myIndexNode3.MergesTotalTime)
	require.Equal(t, *indexToShardsList["my-index"][1].search_query_total, *myIndexNode3.SearchQueryTotal)
	require.Equal(t, *indexToShardsList["my-index"][1].search_query_time, *myIndexNode3.SearchQueryTime)
	require.Equal(t, *indexToShardsList["my-index"][1].merges_total, *myIndexNode3.TotalMergesTotal)
	require.Equal(t, *indexToShardsList["my-index"][1].merges_total_time, *myIndexNode3.TotalMergesTotalTime)
	require.Nil(t, myIndexNode3.IndexFailedRatePerSecond)
	require.Nil(t, myIndexNode3.IndexLatencyInMillis)
	require.Nil(t, myIndexNode3.IndexRatePerSecond)
	require.Nil(t, myIndexNode3.GetMissingDocRatePerSecond)
	require.Nil(t, myIndexNode3.MergeLatencyInMillis)
	require.Nil(t, myIndexNode3.MergeRatePerSecond)
	require.Nil(t, myIndexNode3.SearchLatencyInMillis)
	require.Nil(t, myIndexNode3.SearchRatePerSecond)
	require.Nil(t, myIndexNode3.TimestampDiff)
}
