// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cat_shards

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppendIndexShards(t *testing.T) {
	indexShards := make(map[string][]Shard, 0)

	appendIndexShards(indexShards, "my-index-1", &Shard{shard: 0})

	require.Equal(t, 1, len(indexShards))
	require.Equal(t, 1, len(indexShards["my-index-1"]))
	require.EqualValues(t, 0, indexShards["my-index-1"][0].shard)

	appendIndexShards(indexShards, "my-index-1", &Shard{shard: 1})

	require.Equal(t, 1, len(indexShards))
	require.Equal(t, 2, len(indexShards["my-index-1"]))
	require.EqualValues(t, 0, indexShards["my-index-1"][0].shard)
	require.EqualValues(t, 1, indexShards["my-index-1"][1].shard)

	appendIndexShards(indexShards, "my-index-2", &Shard{shard: 0})

	require.Equal(t, 2, len(indexShards))
	require.Equal(t, 2, len(indexShards["my-index-1"]))
	require.EqualValues(t, 0, indexShards["my-index-1"][0].shard)
	require.EqualValues(t, 1, indexShards["my-index-1"][1].shard)
	require.Equal(t, 1, len(indexShards["my-index-2"]))
	require.EqualValues(t, 0, indexShards["my-index-2"][0].shard)
}

func TestAppendNodeShards(t *testing.T) {
	nodeShards := make(map[string]NodeShardCount, 0)

	appendNodeShards(nodeShards, &Shard{node_id: "abc1", node_name: "node1", primary: true, state: STARTED})

	require.Equal(t, 1, len(nodeShards))
	require.Equal(t, "abc1", nodeShards["abc1"].NodeId)
	require.Equal(t, "node1", nodeShards["abc1"].NodeName)
	require.EqualValues(t, 1, nodeShards["abc1"].Shards)
	require.EqualValues(t, 1, nodeShards["abc1"].PrimaryShards)
	require.EqualValues(t, 0, nodeShards["abc1"].ReplicaShards)
	require.EqualValues(t, 0, nodeShards["abc1"].InitializingShards)
	require.EqualValues(t, 0, nodeShards["abc1"].InitializingPrimaryShards)
	require.EqualValues(t, 0, nodeShards["abc1"].InitializingReplicaShards)
	require.EqualValues(t, 0, nodeShards["abc1"].RelocatingShards)
	require.EqualValues(t, 0, nodeShards["abc1"].RelocatingPrimaryShards)
	require.EqualValues(t, 0, nodeShards["abc1"].RelocatingReplicaShards)

	appendNodeShards(nodeShards, &Shard{node_id: "abc2", node_name: "node2", primary: false, state: INITIALIZING})

	require.Equal(t, 2, len(nodeShards))
	require.Equal(t, "abc2", nodeShards["abc2"].NodeId)
	require.Equal(t, "node2", nodeShards["abc2"].NodeName)
	require.EqualValues(t, 1, nodeShards["abc2"].Shards)
	require.EqualValues(t, 0, nodeShards["abc2"].PrimaryShards)
	require.EqualValues(t, 1, nodeShards["abc2"].ReplicaShards)
	require.EqualValues(t, 1, nodeShards["abc2"].InitializingShards)
	require.EqualValues(t, 0, nodeShards["abc2"].InitializingPrimaryShards)
	require.EqualValues(t, 1, nodeShards["abc2"].InitializingReplicaShards)
	require.EqualValues(t, 0, nodeShards["abc2"].RelocatingShards)
	require.EqualValues(t, 0, nodeShards["abc2"].RelocatingPrimaryShards)
	require.EqualValues(t, 0, nodeShards["abc2"].RelocatingReplicaShards)

	appendNodeShards(nodeShards, &Shard{node_id: "abc1", node_name: "node1", primary: false, state: RELOCATING})

	require.Equal(t, 2, len(nodeShards))
	require.Equal(t, "abc1", nodeShards["abc1"].NodeId)
	require.Equal(t, "node1", nodeShards["abc1"].NodeName)
	require.EqualValues(t, 2, nodeShards["abc1"].Shards)
	require.EqualValues(t, 1, nodeShards["abc1"].PrimaryShards)
	require.EqualValues(t, 1, nodeShards["abc1"].ReplicaShards)
	require.EqualValues(t, 0, nodeShards["abc1"].InitializingShards)
	require.EqualValues(t, 0, nodeShards["abc1"].InitializingPrimaryShards)
	require.EqualValues(t, 0, nodeShards["abc1"].InitializingReplicaShards)
	require.EqualValues(t, 1, nodeShards["abc1"].RelocatingShards)
	require.EqualValues(t, 0, nodeShards["abc1"].RelocatingPrimaryShards)
	require.EqualValues(t, 1, nodeShards["abc1"].RelocatingReplicaShards)

	appendNodeShards(nodeShards, &Shard{node_id: "abc1", node_name: "node1", primary: true, state: STARTED})

	require.Equal(t, 2, len(nodeShards))
	require.Equal(t, "abc1", nodeShards["abc1"].NodeId)
	require.Equal(t, "node1", nodeShards["abc1"].NodeName)
	require.EqualValues(t, 3, nodeShards["abc1"].Shards)
	require.EqualValues(t, 2, nodeShards["abc1"].PrimaryShards)
	require.EqualValues(t, 1, nodeShards["abc1"].ReplicaShards)
	require.EqualValues(t, 0, nodeShards["abc1"].InitializingShards)
	require.EqualValues(t, 0, nodeShards["abc1"].InitializingPrimaryShards)
	require.EqualValues(t, 0, nodeShards["abc1"].InitializingReplicaShards)
	require.EqualValues(t, 1, nodeShards["abc1"].RelocatingShards)
	require.EqualValues(t, 0, nodeShards["abc1"].RelocatingPrimaryShards)
	require.EqualValues(t, 1, nodeShards["abc1"].RelocatingReplicaShards)
}

func TestToAssignedShard(t *testing.T) {
	var (
		sizeInBytes1   int64 = 1234567890
		docs1          int64 = 7890123456
		segmentsCount1 int64 = 56
	)

	shard1 := Shard{
		shard:          0,
		primary:        false,
		store:          &sizeInBytes1,
		docs:           &docs1,
		segments_count: &segmentsCount1,
		state:          STARTED,
	}

	assignedShard1 := toAssignedShard(shard1)

	require.EqualValues(t, shard1.shard, assignedShard1.ShardNum)
	require.Equal(t, shard1.primary, assignedShard1.Primary)
	require.Same(t, &sizeInBytes1, assignedShard1.SizeInBytes)
	require.Same(t, &docs1, assignedShard1.DocsCount)
	require.Same(t, &segmentsCount1, assignedShard1.SegmentsCount)
	require.Equal(t, STARTED, assignedShard1.State)

	var (
		sizeInBytes2   int64 = 234567890
		docs2          int64 = 890123456
		segmentsCount2 int64 = 67
	)

	shard2 := Shard{
		shard:          1,
		primary:        true,
		store:          &sizeInBytes2,
		docs:           &docs2,
		segments_count: &segmentsCount2,
		state:          RELOCATING,
	}

	assignedShard2 := toAssignedShard(shard2)

	require.EqualValues(t, shard2.shard, assignedShard2.ShardNum)
	require.Equal(t, shard2.primary, assignedShard2.Primary)
	require.Same(t, &sizeInBytes2, assignedShard2.SizeInBytes)
	require.Same(t, &docs2, assignedShard2.DocsCount)
	require.Same(t, &segmentsCount2, assignedShard2.SegmentsCount)
	require.Equal(t, RELOCATING, assignedShard2.State)

	var (
		sizeInBytes3   int64 = 34567890
		docs3          int64 = 90123456
		segmentsCount3 int64 = 78
	)

	shard3 := Shard{
		shard:          2,
		primary:        true,
		store:          &sizeInBytes3,
		docs:           &docs3,
		segments_count: &segmentsCount3,
		state:          INITIALIZING,
	}

	assignedShard3 := toAssignedShard(shard3)

	require.EqualValues(t, shard3.shard, assignedShard3.ShardNum)
	require.Equal(t, shard3.primary, assignedShard3.Primary)
	require.Same(t, &sizeInBytes3, assignedShard3.SizeInBytes)
	require.Same(t, &docs3, assignedShard3.DocsCount)
	require.Same(t, &segmentsCount3, assignedShard3.SegmentsCount)
	require.Equal(t, INITIALIZING, assignedShard3.State)
}

func TestToUnassignedShard(t *testing.T) {
	unassignedReason := "reason"
	unassignedDetails := "details"

	shard1 := Shard{
		shard:              0,
		primary:            false,
		unassigned_reason:  &unassignedReason,
		unassigned_details: &unassignedDetails,
	}

	unassignedShard1 := toUnassignedShard(shard1)

	require.EqualValues(t, shard1.shard, unassignedShard1.ShardNum)
	require.Equal(t, shard1.primary, unassignedShard1.Primary)
	require.Same(t, &unassignedReason, unassignedShard1.UnassignedReason)
	require.Same(t, &unassignedDetails, unassignedShard1.UnassignedDetails)

	shard2 := Shard{
		shard:   1,
		primary: true,
	}

	unassignedShard2 := toUnassignedShard(shard2)

	require.EqualValues(t, shard2.shard, unassignedShard2.ShardNum)
	require.Equal(t, shard2.primary, unassignedShard2.Primary)
	require.Nil(t, unassignedShard2.UnassignedReason)
	require.Nil(t, unassignedShard2.UnassignedDetails)
}

func TestIndexShardsToNodeIndexShardsUnassignedRedIndex(t *testing.T) {
	reasons := []string{"reason1", "reason2", "reason3"}
	details := []string{"details1", "details2", "details4"}

	shards := []Shard{
		{shard: 0, primary: true, state: UNASSIGNED, node_id: UNASSIGNED, node_name: UNASSIGNED, unassigned_reason: &reasons[0], unassigned_details: &details[0]},
		{shard: 0, primary: false, state: UNASSIGNED, node_id: UNASSIGNED, node_name: UNASSIGNED, unassigned_reason: &reasons[1], unassigned_details: &details[1]},
		{shard: 1, primary: true, state: UNASSIGNED, node_id: UNASSIGNED, node_name: UNASSIGNED, unassigned_reason: &reasons[2]},
		{shard: 1, primary: false, state: UNASSIGNED, node_id: UNASSIGNED, node_name: UNASSIGNED, unassigned_details: &details[2]},
	}

	nodeIndexShardsMap := make(map[string]NodeIndexShards, 0)

	indexShardsToNodeIndexShards(nodeIndexShardsMap, "my-index", shards)

	require.Equal(t, 1, len(nodeIndexShardsMap))

	nodeIndex := nodeIndexShardsMap["my-index-node_id-UNASSIGNED"]

	require.Equal(t, "my-index", nodeIndex.Index)
	require.Equal(t, "my-index-node_id-UNASSIGNED", nodeIndex.IndexNode)
	require.Nil(t, nodeIndex.Aliases)
	require.Nil(t, nodeIndex.Attributes)
	require.Equal(t, UNASSIGNED, nodeIndex.NodeId)
	require.Equal(t, UNASSIGNED, nodeIndex.NodeName)
	require.Equal(t, RED, *nodeIndex.IndexStatus)
	require.EqualValues(t, 4, nodeIndex.Shards)
	require.EqualValues(t, 2, nodeIndex.PrimaryShards)
	require.EqualValues(t, 2, nodeIndex.ReplicaShards)
	require.Nil(t, nodeIndex.TotalSegmentsCount)
	require.Nil(t, nodeIndex.TotalSizeInBytes)
	require.Nil(t, nodeIndex.TotalMaxShardSizeInBytes)
	require.Nil(t, nodeIndex.TotalMinShardSizeInBytes)
	require.Nil(t, nodeIndex.MaxShardSizeInBytes)
	require.Nil(t, nodeIndex.MinShardSizeInBytes)
	require.Nil(t, nodeIndex.SegmentsCount)
	require.Nil(t, nodeIndex.SizeInBytes)
	require.Nil(t, nodeIndex.DocsCount)
	require.Nil(t, nodeIndex.IndexingFailedIndexTotal)
	require.Nil(t, nodeIndex.IndexingIndexTotal)
	require.Nil(t, nodeIndex.IndexingIndexTotalTime)
	require.Nil(t, nodeIndex.MergesTotal)
	require.Nil(t, nodeIndex.MergesTotalTime)
	require.Nil(t, nodeIndex.GetMissingDocTotal)
	require.Nil(t, nodeIndex.GetMissingDocTotalTime)
	require.Nil(t, nodeIndex.SearchQueryTotal)
	require.Nil(t, nodeIndex.SearchQueryTime)
	require.Nil(t, nodeIndex.TotalMergesTotal)
	require.Nil(t, nodeIndex.TotalMergesTotalTime)
	require.Equal(t, 0, len(nodeIndex.AssignShards))
	require.Equal(t, 0, len(nodeIndex.InitializingShards))
	require.EqualValues(t, 0, nodeIndex.Initializing)
	require.Equal(t, 0, len(nodeIndex.RelocatingShards))
	require.EqualValues(t, 0, nodeIndex.Relocating)
	require.Equal(t, 4, len(nodeIndex.UnassignedShards))
	require.EqualValues(t, 4, nodeIndex.Unassigned)
	require.EqualValues(t, 2, nodeIndex.UnassignedPrimaryShards)
	require.EqualValues(t, 2, nodeIndex.UnassignedReplicasShards)

	for i, shard := range shards {
		unassignedShard := nodeIndex.UnassignedShards[i]

		require.Equal(t, shard.shard, unassignedShard.ShardNum)
		require.Equal(t, shard.primary, unassignedShard.Primary)
		require.Same(t, shard.unassigned_reason, unassignedShard.UnassignedReason)
		require.Same(t, shard.unassigned_details, unassignedShard.UnassignedDetails)
	}
}

func TestIndexShardsToNodeIndexShardsUnassignedYellowIndex(t *testing.T) {
	reasons := []string{"reason1", "reason2"}
	details := []string{"details1", "details2"}

	shards := []Shard{
		{shard: 0, primary: true, state: INITIALIZING, node_id: "node1", node_name: "name1"},
		{shard: 0, primary: false, state: UNASSIGNED, node_id: UNASSIGNED, node_name: UNASSIGNED, unassigned_reason: &reasons[0], unassigned_details: &details[0]},
		{shard: 1, primary: true, state: INITIALIZING, node_id: "node1", node_name: "name1"},
		{shard: 1, primary: false, state: UNASSIGNED, node_id: UNASSIGNED, node_name: UNASSIGNED, unassigned_reason: &reasons[1], unassigned_details: &details[1]},
	}

	nodeIndexShardsMap := make(map[string]NodeIndexShards, 0)

	indexShardsToNodeIndexShards(nodeIndexShardsMap, "my-index", shards)

	require.Equal(t, 2, len(nodeIndexShardsMap))

	nodeIndex := nodeIndexShardsMap["my-index-node_id-UNASSIGNED"]

	require.Equal(t, "my-index", nodeIndex.Index)
	require.Equal(t, "my-index-node_id-UNASSIGNED", nodeIndex.IndexNode)
	require.Nil(t, nodeIndex.Aliases)
	require.Nil(t, nodeIndex.Attributes)
	require.Equal(t, UNASSIGNED, nodeIndex.NodeId)
	require.Equal(t, UNASSIGNED, nodeIndex.NodeName)
	require.Equal(t, YELLOW, *nodeIndex.IndexStatus)
	require.EqualValues(t, 2, nodeIndex.Shards)
	require.EqualValues(t, 0, nodeIndex.PrimaryShards)
	require.EqualValues(t, 2, nodeIndex.ReplicaShards)
	require.Nil(t, nodeIndex.TotalSegmentsCount)
	require.Nil(t, nodeIndex.TotalSizeInBytes)
	require.Nil(t, nodeIndex.TotalMaxShardSizeInBytes)
	require.Nil(t, nodeIndex.TotalMinShardSizeInBytes)
	require.Nil(t, nodeIndex.MaxShardSizeInBytes)
	require.Nil(t, nodeIndex.MinShardSizeInBytes)
	require.Nil(t, nodeIndex.SegmentsCount)
	require.Nil(t, nodeIndex.SizeInBytes)
	require.Nil(t, nodeIndex.DocsCount)
	require.Nil(t, nodeIndex.IndexingFailedIndexTotal)
	require.Nil(t, nodeIndex.IndexingIndexTotal)
	require.Nil(t, nodeIndex.IndexingIndexTotalTime)
	require.Nil(t, nodeIndex.MergesTotal)
	require.Nil(t, nodeIndex.MergesTotalTime)
	require.Nil(t, nodeIndex.GetMissingDocTotal)
	require.Nil(t, nodeIndex.GetMissingDocTotalTime)
	require.Nil(t, nodeIndex.SearchQueryTotal)
	require.Nil(t, nodeIndex.SearchQueryTime)
	require.Nil(t, nodeIndex.TotalMergesTotal)
	require.Nil(t, nodeIndex.TotalMergesTotalTime)
	require.Equal(t, 0, len(nodeIndex.AssignShards))
	require.Equal(t, 0, len(nodeIndex.InitializingShards))
	require.EqualValues(t, 0, nodeIndex.Initializing)
	require.Equal(t, 0, len(nodeIndex.RelocatingShards))
	require.EqualValues(t, 0, nodeIndex.Relocating)
	require.Equal(t, 2, len(nodeIndex.UnassignedShards))
	require.EqualValues(t, 2, nodeIndex.Unassigned)
	require.EqualValues(t, 0, nodeIndex.UnassignedPrimaryShards)
	require.EqualValues(t, 2, nodeIndex.UnassignedReplicasShards)

	for i, shard := range shards {
		if shard.state != UNASSIGNED {
			continue
		}

		unassignedShard := nodeIndex.UnassignedShards[i/2]

		require.Equal(t, shard.shard, unassignedShard.ShardNum)
		require.Equal(t, shard.primary, unassignedShard.Primary)
		require.Same(t, shard.unassigned_reason, unassignedShard.UnassignedReason)
		require.Same(t, shard.unassigned_details, unassignedShard.UnassignedDetails)
	}

	nodeIndex = nodeIndexShardsMap["my-index-node_id-node1"]

	require.Equal(t, "my-index", nodeIndex.Index)
	require.Equal(t, "my-index-node_id-node1", nodeIndex.IndexNode)
	require.Nil(t, nodeIndex.Aliases)
	require.Nil(t, nodeIndex.Attributes)
	require.Equal(t, "node1", nodeIndex.NodeId)
	require.Equal(t, "name1", nodeIndex.NodeName)
	require.Equal(t, YELLOW, *nodeIndex.IndexStatus)
	require.EqualValues(t, 2, nodeIndex.Shards)
	require.EqualValues(t, 2, nodeIndex.PrimaryShards)
	require.EqualValues(t, 0, nodeIndex.ReplicaShards)
	require.Nil(t, nodeIndex.TotalSegmentsCount)
	require.Nil(t, nodeIndex.TotalSizeInBytes)
	require.Nil(t, nodeIndex.TotalMaxShardSizeInBytes)
	require.Nil(t, nodeIndex.TotalMinShardSizeInBytes)
	require.Nil(t, nodeIndex.MaxShardSizeInBytes)
	require.Nil(t, nodeIndex.MinShardSizeInBytes)
	require.Nil(t, nodeIndex.SegmentsCount)
	require.Nil(t, nodeIndex.SizeInBytes)
	require.Nil(t, nodeIndex.DocsCount)
	require.Nil(t, nodeIndex.IndexingFailedIndexTotal)
	require.Nil(t, nodeIndex.IndexingIndexTotal)
	require.Nil(t, nodeIndex.IndexingIndexTotalTime)
	require.Nil(t, nodeIndex.MergesTotal)
	require.Nil(t, nodeIndex.MergesTotalTime)
	require.Nil(t, nodeIndex.GetMissingDocTotal)
	require.Nil(t, nodeIndex.GetMissingDocTotalTime)
	require.Nil(t, nodeIndex.SearchQueryTotal)
	require.Nil(t, nodeIndex.SearchQueryTime)
	require.Nil(t, nodeIndex.TotalMergesTotal)
	require.Nil(t, nodeIndex.TotalMergesTotalTime)
	require.Equal(t, 0, len(nodeIndex.AssignShards))
	require.Equal(t, 2, len(nodeIndex.InitializingShards))
	require.EqualValues(t, 2, nodeIndex.Initializing)
	require.Equal(t, 0, len(nodeIndex.RelocatingShards))
	require.EqualValues(t, 0, nodeIndex.Relocating)
	require.Equal(t, 0, len(nodeIndex.UnassignedShards))
	require.EqualValues(t, 0, nodeIndex.Unassigned)
	require.EqualValues(t, 0, nodeIndex.UnassignedPrimaryShards)
	require.EqualValues(t, 0, nodeIndex.UnassignedReplicasShards)

	for i, shard := range shards {
		if shard.state == UNASSIGNED {
			continue
		}

		assignedShard := nodeIndex.InitializingShards[i/2]

		require.Equal(t, shard.shard, assignedShard.ShardNum)
		require.Equal(t, shard.primary, assignedShard.Primary)
		require.Nil(t, assignedShard.SizeInBytes)
		require.Nil(t, assignedShard.DocsCount)
		require.Nil(t, assignedShard.SegmentsCount)
		require.Equal(t, INITIALIZING, assignedShard.State)
	}
}

func TestIndexShardsToNodeIndexShardsGreenIndex(t *testing.T) {
	docs := []int64{100001, 100000, 200001, 200000}
	segments := []int64{10, 11, 12, 13}
	stores := []int64{20, 21, 22, 23}
	getMissingTotals := []int64{30, 31, 32, 33}
	getMissingTimes := []int64{34, 35, 36, 37}
	searchQueryTotals := []int64{40, 41, 42, 43}
	searchQueryTimes := []int64{44, 45, 46, 47}
	mergeTotals := []int64{50, 51, 52, 53}
	mergeTimes := []int64{54, 55, 56, 57}
	indexingFailed := []int64{60, 61, 62, 63}
	indexingTotals := []int64{70, 71, 72, 73}
	indexingTimes := []int64{74, 75, 76, 77}

	shards := []Shard{
		{shard: 0, primary: true, node_id: "node1", node_name: "name1"},
		{shard: 0, primary: false, node_id: "node2", node_name: "name2"},
		{shard: 1, primary: true, node_id: "node2", node_name: "name2"},
		{shard: 1, primary: false, node_id: "node3", node_name: "name3"},
	}

	for i, shard := range shards {
		shard.state = STARTED
		shard.docs = &docs[i]
		shard.segments_count = &segments[i]
		shard.store = &stores[i]
		shard.get_missing_total = &getMissingTotals[i]
		shard.get_missing_time = &getMissingTimes[i]
		shard.search_query_total = &searchQueryTotals[i]
		shard.search_query_time = &searchQueryTimes[i]
		shard.merges_total = &mergeTotals[i]
		shard.merges_total_time = &mergeTimes[i]
		shard.indexing_index_failed = &indexingFailed[i]
		shard.indexing_index_total = &indexingTotals[i]
		shard.indexing_index_time = &indexingTimes[i]

		shards[i] = shard
	}

	nodeIndexShardsMap := make(map[string]NodeIndexShards, 0)

	indexShardsToNodeIndexShards(nodeIndexShardsMap, "my-index", shards)

	require.Equal(t, 3, len(nodeIndexShardsMap))

	nodeIndex := nodeIndexShardsMap["my-index-node_id-node1"]

	require.Equal(t, "my-index", nodeIndex.Index)
	require.Equal(t, "my-index-node_id-node1", nodeIndex.IndexNode)
	require.Nil(t, nodeIndex.Aliases)
	require.Nil(t, nodeIndex.Attributes)
	require.Equal(t, "node1", nodeIndex.NodeId)
	require.Equal(t, "name1", nodeIndex.NodeName)
	require.Equal(t, GREEN, *nodeIndex.IndexStatus)
	require.EqualValues(t, 1, nodeIndex.Shards)
	require.EqualValues(t, 1, nodeIndex.PrimaryShards)
	require.EqualValues(t, 0, nodeIndex.ReplicaShards)
	require.Equal(t, segments[0], *nodeIndex.TotalSegmentsCount)
	require.Equal(t, stores[0], *nodeIndex.TotalSizeInBytes)
	require.Equal(t, stores[0], *nodeIndex.TotalMaxShardSizeInBytes)
	require.Equal(t, stores[0], *nodeIndex.TotalMinShardSizeInBytes)
	require.Equal(t, stores[0], *nodeIndex.MaxShardSizeInBytes)
	require.Equal(t, stores[0], *nodeIndex.MinShardSizeInBytes)
	require.Equal(t, stores[0], *nodeIndex.SizeInBytes)
	require.Equal(t, segments[0], *nodeIndex.SegmentsCount)
	require.Equal(t, docs[0], *nodeIndex.DocsCount)
	require.Equal(t, indexingFailed[0], *nodeIndex.IndexingFailedIndexTotal)
	require.Equal(t, indexingTotals[0], *nodeIndex.IndexingIndexTotal)
	require.Equal(t, indexingTimes[0], *nodeIndex.IndexingIndexTotalTime)
	require.Equal(t, mergeTotals[0], *nodeIndex.MergesTotal)
	require.Equal(t, mergeTimes[0], *nodeIndex.MergesTotalTime)
	require.Equal(t, getMissingTotals[0], *nodeIndex.GetMissingDocTotal)
	require.Equal(t, getMissingTimes[0], *nodeIndex.GetMissingDocTotalTime)
	require.Equal(t, searchQueryTotals[0], *nodeIndex.SearchQueryTotal)
	require.Equal(t, searchQueryTimes[0], *nodeIndex.SearchQueryTime)
	require.Equal(t, mergeTotals[0], *nodeIndex.TotalMergesTotal)
	require.Equal(t, mergeTimes[0], *nodeIndex.TotalMergesTotalTime)
	require.Equal(t, 1, len(nodeIndex.AssignShards))
	require.Equal(t, 0, len(nodeIndex.InitializingShards))
	require.EqualValues(t, 0, nodeIndex.Initializing)
	require.Equal(t, 0, len(nodeIndex.RelocatingShards))
	require.EqualValues(t, 0, nodeIndex.Relocating)
	require.Equal(t, 0, len(nodeIndex.UnassignedShards))
	require.EqualValues(t, 0, nodeIndex.Unassigned)
	require.EqualValues(t, 0, nodeIndex.UnassignedPrimaryShards)
	require.EqualValues(t, 0, nodeIndex.UnassignedReplicasShards)

	assignedShard1 := nodeIndex.AssignShards[0]

	require.Equal(t, shards[0].shard, assignedShard1.ShardNum)
	require.Equal(t, shards[0].primary, assignedShard1.Primary)
	require.Same(t, shards[0].store, assignedShard1.SizeInBytes)
	require.Same(t, shards[0].docs, assignedShard1.DocsCount)
	require.Same(t, shards[0].segments_count, assignedShard1.SegmentsCount)
	require.Equal(t, STARTED, assignedShard1.State)

	nodeIndex = nodeIndexShardsMap["my-index-node_id-node2"]

	require.Equal(t, "my-index", nodeIndex.Index)
	require.Equal(t, "my-index-node_id-node2", nodeIndex.IndexNode)
	require.Nil(t, nodeIndex.Aliases)
	require.Nil(t, nodeIndex.Attributes)
	require.Equal(t, "node2", nodeIndex.NodeId)
	require.Equal(t, "name2", nodeIndex.NodeName)
	require.Equal(t, GREEN, *nodeIndex.IndexStatus)
	require.EqualValues(t, 2, nodeIndex.Shards)
	require.EqualValues(t, 1, nodeIndex.PrimaryShards)
	require.EqualValues(t, 1, nodeIndex.ReplicaShards)
	require.Equal(t, segments[1]+segments[2], *nodeIndex.TotalSegmentsCount)
	require.Equal(t, stores[1]+stores[2], *nodeIndex.TotalSizeInBytes)
	require.Equal(t, stores[2], *nodeIndex.TotalMaxShardSizeInBytes)
	require.Equal(t, stores[1], *nodeIndex.TotalMinShardSizeInBytes)
	require.Equal(t, stores[2], *nodeIndex.MaxShardSizeInBytes)
	require.Equal(t, stores[2], *nodeIndex.MinShardSizeInBytes)
	require.Equal(t, stores[2], *nodeIndex.SizeInBytes)
	require.Equal(t, segments[2], *nodeIndex.SegmentsCount)
	require.Equal(t, docs[2], *nodeIndex.DocsCount)
	require.Equal(t, indexingFailed[2], *nodeIndex.IndexingFailedIndexTotal)
	require.Equal(t, indexingTotals[2], *nodeIndex.IndexingIndexTotal)
	require.Equal(t, indexingTimes[2], *nodeIndex.IndexingIndexTotalTime)
	require.Equal(t, mergeTotals[2], *nodeIndex.MergesTotal)
	require.Equal(t, mergeTimes[2], *nodeIndex.MergesTotalTime)
	require.Equal(t, getMissingTotals[1]+getMissingTotals[2], *nodeIndex.GetMissingDocTotal)
	require.Equal(t, getMissingTimes[1]+getMissingTimes[2], *nodeIndex.GetMissingDocTotalTime)
	require.Equal(t, searchQueryTotals[1]+searchQueryTotals[2], *nodeIndex.SearchQueryTotal)
	require.Equal(t, searchQueryTimes[1]+searchQueryTimes[2], *nodeIndex.SearchQueryTime)
	require.Equal(t, mergeTotals[1]+mergeTotals[2], *nodeIndex.TotalMergesTotal)
	require.Equal(t, mergeTimes[1]+mergeTimes[2], *nodeIndex.TotalMergesTotalTime)
	require.Equal(t, 2, len(nodeIndex.AssignShards))
	require.Equal(t, 0, len(nodeIndex.InitializingShards))
	require.EqualValues(t, 0, nodeIndex.Initializing)
	require.Equal(t, 0, len(nodeIndex.RelocatingShards))
	require.EqualValues(t, 0, nodeIndex.Relocating)
	require.Equal(t, 0, len(nodeIndex.UnassignedShards))
	require.EqualValues(t, 0, nodeIndex.Unassigned)
	require.EqualValues(t, 0, nodeIndex.UnassignedPrimaryShards)
	require.EqualValues(t, 0, nodeIndex.UnassignedReplicasShards)

	assignedShard2 := nodeIndex.AssignShards[0]

	require.Equal(t, shards[1].shard, assignedShard2.ShardNum)
	require.Equal(t, shards[1].primary, assignedShard2.Primary)
	require.Same(t, shards[1].store, assignedShard2.SizeInBytes)
	require.Same(t, shards[1].docs, assignedShard2.DocsCount)
	require.Same(t, shards[1].segments_count, assignedShard2.SegmentsCount)
	require.Equal(t, STARTED, assignedShard2.State)

	assignedShard3 := nodeIndex.AssignShards[1]

	require.Equal(t, shards[2].shard, assignedShard3.ShardNum)
	require.Equal(t, shards[2].primary, assignedShard3.Primary)
	require.Same(t, shards[2].store, assignedShard3.SizeInBytes)
	require.Same(t, shards[2].docs, assignedShard3.DocsCount)
	require.Same(t, shards[2].segments_count, assignedShard3.SegmentsCount)
	require.Equal(t, STARTED, assignedShard3.State)

	nodeIndex = nodeIndexShardsMap["my-index-node_id-node3"]

	require.Equal(t, "my-index", nodeIndex.Index)
	require.Equal(t, "my-index-node_id-node3", nodeIndex.IndexNode)
	require.Nil(t, nodeIndex.Aliases)
	require.Nil(t, nodeIndex.Attributes)
	require.Equal(t, "node3", nodeIndex.NodeId)
	require.Equal(t, "name3", nodeIndex.NodeName)
	require.Equal(t, GREEN, *nodeIndex.IndexStatus)
	require.EqualValues(t, 1, nodeIndex.Shards)
	require.EqualValues(t, 0, nodeIndex.PrimaryShards)
	require.EqualValues(t, 1, nodeIndex.ReplicaShards)
	require.Equal(t, segments[3], *nodeIndex.TotalSegmentsCount)
	require.Equal(t, stores[3], *nodeIndex.TotalSizeInBytes)
	require.Equal(t, stores[3], *nodeIndex.TotalMaxShardSizeInBytes)
	require.Equal(t, stores[3], *nodeIndex.TotalMinShardSizeInBytes)
	require.Nil(t, nodeIndex.MaxShardSizeInBytes)
	require.Nil(t, nodeIndex.MinShardSizeInBytes)
	require.Nil(t, nodeIndex.SizeInBytes)
	require.Nil(t, nodeIndex.SegmentsCount)
	require.Nil(t, nodeIndex.DocsCount)
	require.Nil(t, nodeIndex.IndexingFailedIndexTotal)
	require.Nil(t, nodeIndex.IndexingIndexTotal)
	require.Nil(t, nodeIndex.IndexingIndexTotalTime)
	require.Nil(t, nodeIndex.MergesTotal)
	require.Nil(t, nodeIndex.MergesTotalTime)
	require.Equal(t, getMissingTotals[3], *nodeIndex.GetMissingDocTotal)
	require.Equal(t, getMissingTimes[3], *nodeIndex.GetMissingDocTotalTime)
	require.Equal(t, searchQueryTotals[3], *nodeIndex.SearchQueryTotal)
	require.Equal(t, searchQueryTimes[3], *nodeIndex.SearchQueryTime)
	require.Equal(t, mergeTotals[3], *nodeIndex.TotalMergesTotal)
	require.Equal(t, mergeTimes[3], *nodeIndex.TotalMergesTotalTime)
	require.Equal(t, 1, len(nodeIndex.AssignShards))
	require.Equal(t, 0, len(nodeIndex.InitializingShards))
	require.EqualValues(t, 0, nodeIndex.Initializing)
	require.Equal(t, 0, len(nodeIndex.RelocatingShards))
	require.EqualValues(t, 0, nodeIndex.Relocating)
	require.Equal(t, 0, len(nodeIndex.UnassignedShards))
	require.EqualValues(t, 0, nodeIndex.Unassigned)
	require.EqualValues(t, 0, nodeIndex.UnassignedPrimaryShards)
	require.EqualValues(t, 0, nodeIndex.UnassignedReplicasShards)

	assignedShard4 := nodeIndex.AssignShards[0]

	require.Equal(t, shards[3].shard, assignedShard4.ShardNum)
	require.Equal(t, shards[3].primary, assignedShard4.Primary)
	require.Same(t, shards[3].store, assignedShard4.SizeInBytes)
	require.Same(t, shards[3].docs, assignedShard4.DocsCount)
	require.Same(t, shards[3].segments_count, assignedShard4.SegmentsCount)
	require.Equal(t, STARTED, assignedShard4.State)
}

func TestIndexShardsToNodeIndexShardsRelocating(t *testing.T) {
	docs := int64(100001)
	segments := int64(10)
	stores := int64(20)
	getMissingTotals := int64(30)
	getMissingTimes := int64(31)
	searchQueryTotals := int64(40)
	searchQueryTimes := int64(41)
	mergeTotals := int64(50)
	mergeTimes := int64(51)
	indexingFailed := int64(60)
	indexingTotals := int64(70)
	indexingTimes := int64(71)

	shards := []Shard{
		{
			shard:                 0,
			primary:               true,
			state:                 RELOCATING,
			node_id:               "node1",
			node_name:             "name1",
			docs:                  &docs,
			segments_count:        &segments,
			store:                 &stores,
			get_missing_total:     &getMissingTotals,
			get_missing_time:      &getMissingTimes,
			search_query_total:    &searchQueryTotals,
			search_query_time:     &searchQueryTimes,
			merges_total:          &mergeTotals,
			merges_total_time:     &mergeTimes,
			indexing_index_failed: &indexingFailed,
			indexing_index_total:  &indexingTotals,
			indexing_index_time:   &indexingTimes,
		},
	}

	nodeIndexShardsMap := make(map[string]NodeIndexShards, 0)

	indexShardsToNodeIndexShards(nodeIndexShardsMap, "my-index", shards)

	require.Equal(t, 1, len(nodeIndexShardsMap))

	nodeIndex := nodeIndexShardsMap["my-index-node_id-node1"]

	require.Equal(t, "my-index", nodeIndex.Index)
	require.Equal(t, "my-index-node_id-node1", nodeIndex.IndexNode)
	require.Nil(t, nodeIndex.Aliases)
	require.Nil(t, nodeIndex.Attributes)
	require.Equal(t, "node1", nodeIndex.NodeId)
	require.Equal(t, "name1", nodeIndex.NodeName)
	require.Equal(t, GREEN, *nodeIndex.IndexStatus)
	require.EqualValues(t, 1, nodeIndex.Shards)
	require.EqualValues(t, 1, nodeIndex.PrimaryShards)
	require.EqualValues(t, 0, nodeIndex.ReplicaShards)
	require.Equal(t, segments, *nodeIndex.TotalSegmentsCount)
	require.Equal(t, stores, *nodeIndex.TotalSizeInBytes)
	require.Equal(t, stores, *nodeIndex.TotalMaxShardSizeInBytes)
	require.Equal(t, stores, *nodeIndex.TotalMinShardSizeInBytes)
	require.Equal(t, stores, *nodeIndex.MaxShardSizeInBytes)
	require.Equal(t, stores, *nodeIndex.MinShardSizeInBytes)
	require.Equal(t, stores, *nodeIndex.SizeInBytes)
	require.Equal(t, segments, *nodeIndex.SegmentsCount)
	require.Equal(t, docs, *nodeIndex.DocsCount)
	require.Equal(t, indexingFailed, *nodeIndex.IndexingFailedIndexTotal)
	require.Equal(t, indexingTotals, *nodeIndex.IndexingIndexTotal)
	require.Equal(t, indexingTimes, *nodeIndex.IndexingIndexTotalTime)
	require.Equal(t, mergeTotals, *nodeIndex.MergesTotal)
	require.Equal(t, mergeTimes, *nodeIndex.MergesTotalTime)
	require.Equal(t, getMissingTotals, *nodeIndex.GetMissingDocTotal)
	require.Equal(t, getMissingTimes, *nodeIndex.GetMissingDocTotalTime)
	require.Equal(t, searchQueryTotals, *nodeIndex.SearchQueryTotal)
	require.Equal(t, searchQueryTimes, *nodeIndex.SearchQueryTime)
	require.Equal(t, mergeTotals, *nodeIndex.TotalMergesTotal)
	require.Equal(t, mergeTimes, *nodeIndex.TotalMergesTotalTime)
	require.Equal(t, 0, len(nodeIndex.AssignShards))
	require.Equal(t, 0, len(nodeIndex.InitializingShards))
	require.EqualValues(t, 0, nodeIndex.Initializing)
	require.Equal(t, 1, len(nodeIndex.RelocatingShards))
	require.EqualValues(t, 1, nodeIndex.Relocating)
	require.Equal(t, 0, len(nodeIndex.UnassignedShards))
	require.EqualValues(t, 0, nodeIndex.Unassigned)
	require.EqualValues(t, 0, nodeIndex.UnassignedPrimaryShards)
	require.EqualValues(t, 0, nodeIndex.UnassignedReplicasShards)

	assignedShard := nodeIndex.RelocatingShards[0]

	require.Equal(t, shards[0].shard, assignedShard.ShardNum)
	require.Equal(t, shards[0].primary, assignedShard.Primary)
	require.Same(t, shards[0].store, assignedShard.SizeInBytes)
	require.Same(t, shards[0].docs, assignedShard.DocsCount)
	require.Same(t, shards[0].segments_count, assignedShard.SegmentsCount)
	require.Equal(t, RELOCATING, assignedShard.State)
}
