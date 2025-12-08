// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cat_shards

import (
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

type Shard struct {
	node_id   string
	node_name string
	shard     int32
	primary   bool
	state     string

	// optional; set if state != "UNASSIGNED"
	docs                  *int64
	store                 *int64
	segments_count        *int64
	search_query_total    *int64
	search_query_time     *int64
	indexing_index_total  *int64
	indexing_index_time   *int64
	indexing_index_failed *int64
	merges_total          *int64
	merges_total_time     *int64
	get_missing_time      *int64
	get_missing_total     *int64
	unassigned_reason     *string
	unassigned_details    *string
}

type AssignedShard struct {
	ShardNum      int32  `json:"shard_num"`
	Primary       bool   `json:"primary"`
	SizeInBytes   *int64 `json:"size_in_bytes"`
	DocsCount     *int64 `json:"docs_count"`
	SegmentsCount *int64 `json:"segments_count"`
	State         string `json:"state"`
}

type UnassignedShard struct {
	ShardNum          int32   `json:"shard_num"`
	Primary           bool    `json:"primary"`
	UnassignedReason  *string `json:"unassigned_reason"`
	UnassignedDetails *string `json:"unassigned_details"`
}

type IndexMetadata struct {
	indexType  string
	aliases    []string
	hidden     bool
	system     bool
	open       bool
	attributes []string
}

type NodeIndexShards struct {
	TotalFractions int32 `json:"total_fractions"`

	Index                    string            `json:"index"`
	IndexNode                string            `json:"index_node"`
	IndexStatus              *string           `json:"index_status"`
	IndexType                *string           `json:"index_type"`
	Aliases                  []string          `json:"aliases"`
	Attributes               []string          `json:"attributes"`
	IsHidden                 *bool             `json:"is_hidden"`
	IsOpen                   *bool             `json:"is_open"`
	IsSystem                 *bool             `json:"is_system"`
	NodeId                   string            `json:"node_id"`
	NodeName                 string            `json:"node_name"`
	AssignShards             []AssignedShard   `json:"assign_shards"`
	InitializingShards       []AssignedShard   `json:"initializing_shards"`
	RelocatingShards         []AssignedShard   `json:"relocating_shards"`
	UnassignedShards         []UnassignedShard `json:"unassign_shards"`
	Shards                   int32             `json:"shards_count"`
	PrimaryShards            int32             `json:"primary_shards_count"`
	ReplicaShards            int32             `json:"replica_shards_count"`
	Initializing             int32             `json:"initializing_shards_count"`
	Relocating               int32             `json:"relocating_shards_count"`
	Unassigned               int32             `json:"unassigned_shards_count"`
	UnassignedPrimaryShards  int32             `json:"total_unassigned_primary_shards"`
	UnassignedReplicasShards int32             `json:"total_unassigned_replicas_shards"`
	SegmentsCount            *int64            `json:"segments_count"`
	SizeInBytes              *int64            `json:"size_in_bytes"`
	TotalSegmentsCount       *int64            `json:"total_segments_count"` // includes replicas
	TotalSizeInBytes         *int64            `json:"total_size_in_bytes"`  // includes replicas
	MaxShardSizeInBytes      *int64            `json:"max_shard_size_in_bytes"`
	MinShardSizeInBytes      *int64            `json:"min_shard_size_in_bytes"`
	TotalMaxShardSizeInBytes *int64            `json:"total_max_shard_size_in_bytes"` // includes replicas
	TotalMinShardSizeInBytes *int64            `json:"total_min_shard_size_in_bytes"` // includes replicas

	// indexing metrics only consider primary shards!

	DocsCount                  *int64   `json:"docs_count"`
	IndexFailedRatePerSecond   *float64 `json:"index_failed_rate_per_second"`
	IndexLatencyInMillis       *float64 `json:"index_latency_in_millis"`
	IndexRatePerSecond         *float64 `json:"index_rate_per_second"`
	IndexingFailedIndexTotal   *int64   `json:"indexing_failed_index_total"`
	IndexingIndexTotal         *int64   `json:"indexing_index_total"`
	IndexingIndexTotalTime     *int64   `json:"indexing_index_total_time"`
	GetMissingDocTotal         *int64   `json:"get_missing_doc_total"`           // includes replicas
	GetMissingDocTotalTime     *int64   `json:"get_missing_doc_total_time"`      // includes replicas
	GetMissingDocRatePerSecond *float64 `json:"get_missing_doc_rate_per_second"` // includes replicas
	MergeLatencyInMillis       *float64 `json:"merge_latency_in_millis"`
	MergeRatePerSecond         *float64 `json:"merge_rate_per_second"`
	MergesTotal                *int64   `json:"merges_total"`
	MergesTotalTime            *int64   `json:"merges_total_time"`
	SearchLatencyInMillis      *float64 `json:"search_latency_in_millis"` // includes replicas
	SearchQueryTime            *int64   `json:"search_query_time"`        // includes replicas
	SearchQueryTotal           *int64   `json:"search_query_total"`       // includes replicas
	SearchRatePerSecond        *float64 `json:"search_rate_per_second"`   // includes replicas
	TotalMergesTotal           *int64   `json:"total_merges_total"`       // includes replicas
	TotalMergesTotalTime       *int64   `json:"total_merges_total_time"`  // includes replicas
	TimestampDiff              *int64   `json:"timestamp_diff"`
}

type NodeShardCount struct {
	NodeId                    string `json:"node_id"`
	NodeName                  string `json:"node_name"`
	Shards                    int32  `json:"shards_count"`
	PrimaryShards             int32  `json:"primary_shards"`
	ReplicaShards             int32  `json:"replica_shards"`
	InitializingShards        int32  `json:"initializing_shards"`
	InitializingPrimaryShards int32  `json:"initializing_primary_shards"`
	InitializingReplicaShards int32  `json:"initializing_replica_shards"`
	RelocatingShards          int32  `json:"relocating_shards"`
	RelocatingPrimaryShards   int32  `json:"relocating_primary_shards"`
	RelocatingReplicaShards   int32  `json:"relocating_replica_shards"`
}

type ShardInfo struct {
	ShardNum          string `json:"shard_num"`
	ShardId           string `json:"shard_id"`
	Primary           bool   `json:"primary"`
	SizeInBytes       uint64 `json:"size_in_bytes"`
	DocsCount         uint64 `json:"docs_count"`
	UnAssignedReason  string `json:"unassigned_reason"`
	UnAssignedDetails string `json:"unassigned_details"`
}

func toAssignedShard(shard Shard) AssignedShard {
	return AssignedShard{
		ShardNum:      shard.shard,
		Primary:       shard.primary,
		SizeInBytes:   shard.store,
		DocsCount:     shard.docs,
		SegmentsCount: shard.segments_count,
		State:         shard.state,
	}
}

func toUnassignedShard(shard Shard) UnassignedShard {
	return UnassignedShard{
		ShardNum:          shard.shard,
		Primary:           shard.primary,
		UnassignedReason:  shard.unassigned_reason,
		UnassignedDetails: shard.unassigned_details,
	}
}

func appendIndexShards(indexShards map[string][]Shard, index string, shard *Shard) {
	shards, found := indexShards[index]

	if !found {
		shards = make([]Shard, 0, 1)
	}

	indexShards[index] = append(shards, *shard)
}

func appendNodeShards(nodeShards map[string]NodeShardCount, shard *Shard) {
	node, found := nodeShards[shard.node_id]

	if !found {
		node = NodeShardCount{
			NodeId:   shard.node_id,
			NodeName: shard.node_name,
		}
	}

	var (
		primaryShard int32 = 0
		replicaShard int32 = 0
	)

	node.Shards++

	if shard.primary {
		primaryShard = 1
		node.PrimaryShards++
	} else {
		replicaShard = 1
		node.ReplicaShards++
	}

	switch shard.state {
	case INITIALIZING:
		node.InitializingShards += 1
		node.InitializingPrimaryShards += primaryShard
		node.InitializingReplicaShards += replicaShard
	case RELOCATING:
		node.RelocatingShards += 1
		node.RelocatingPrimaryShards += primaryShard
		node.RelocatingReplicaShards += replicaShard
	}

	nodeShards[shard.node_id] = node
}

func indexShardsToNodeIndexShards(nodeIndexShardsMap map[string]NodeIndexShards, index string, shards []Shard) {
	status := GREEN
	indexStatus := &status

	for _, shard := range shards {
		indexNodeId := index + "-node_id-" + shard.node_id
		nodeIndex, found := nodeIndexShardsMap[indexNodeId]

		// initial setup for this node + index
		if !found {
			nodeIndex.Index = index
			nodeIndex.IndexNode = indexNodeId
			nodeIndex.IndexStatus = indexStatus
			nodeIndex.Aliases = nil
			nodeIndex.Attributes = nil
			nodeIndex.NodeId = shard.node_id
			nodeIndex.NodeName = shard.node_name
		}

		nodeIndex.Shards++

		if shard.primary {
			nodeIndex.PrimaryShards++
		} else {
			nodeIndex.ReplicaShards++
		}

		if shard.state != UNASSIGNED {
			// store related data:
			nodeIndex.TotalSegmentsCount = utils.AddInt64OrNull(nodeIndex.TotalSegmentsCount, shard.segments_count)
			nodeIndex.TotalSizeInBytes = utils.AddInt64OrNull(nodeIndex.TotalSizeInBytes, shard.store)

			if nodeIndex.TotalMaxShardSizeInBytes == nil {
				if shard.store != nil {
					totalMax := *shard.store
					nodeIndex.TotalMaxShardSizeInBytes = &totalMax
				}
			} else if shard.store != nil {
				*nodeIndex.TotalMaxShardSizeInBytes = max(*nodeIndex.TotalMaxShardSizeInBytes, *shard.store)
			}

			// _no_ assigned shard is 0 bytes, so we don't need any weird checks here
			if nodeIndex.TotalMinShardSizeInBytes == nil {
				if shard.store != nil {
					totalMin := *shard.store
					nodeIndex.TotalMinShardSizeInBytes = &totalMin
				}
			} else if shard.store != nil {
				*nodeIndex.TotalMinShardSizeInBytes = min(*nodeIndex.TotalMinShardSizeInBytes, *shard.store)
			}

			// index stats:
			nodeIndex.GetMissingDocTotal = utils.AddInt64OrNull(nodeIndex.GetMissingDocTotal, shard.get_missing_total)
			nodeIndex.GetMissingDocTotalTime = utils.AddInt64OrNull(nodeIndex.GetMissingDocTotalTime, shard.get_missing_time)
			nodeIndex.SearchQueryTime = utils.AddInt64OrNull(nodeIndex.SearchQueryTime, shard.search_query_time)
			nodeIndex.SearchQueryTotal = utils.AddInt64OrNull(nodeIndex.SearchQueryTotal, shard.search_query_total)
			nodeIndex.TotalMergesTotal = utils.AddInt64OrNull(nodeIndex.TotalMergesTotal, shard.merges_total)
			nodeIndex.TotalMergesTotalTime = utils.AddInt64OrNull(nodeIndex.TotalMergesTotalTime, shard.merges_total_time)

			// store / index stats that we only care about from primary shards:
			if shard.primary {
				if nodeIndex.MaxShardSizeInBytes == nil {
					nodeIndex.MaxShardSizeInBytes = shard.store
				} else if shard.store != nil {
					*nodeIndex.MaxShardSizeInBytes = max(*nodeIndex.MaxShardSizeInBytes, *shard.store)
				}

				if nodeIndex.MinShardSizeInBytes == nil {
					nodeIndex.MinShardSizeInBytes = shard.store
				} else if shard.store != nil {
					*nodeIndex.MinShardSizeInBytes = min(*nodeIndex.MinShardSizeInBytes, *shard.store)
				}

				nodeIndex.SegmentsCount = utils.AddInt64OrNull(nodeIndex.SegmentsCount, shard.segments_count)
				nodeIndex.SizeInBytes = utils.AddInt64OrNull(nodeIndex.SizeInBytes, shard.store)

				nodeIndex.DocsCount = utils.AddInt64OrNull(nodeIndex.DocsCount, shard.docs)
				nodeIndex.IndexingFailedIndexTotal = utils.AddInt64OrNull(nodeIndex.IndexingFailedIndexTotal, shard.indexing_index_failed)
				nodeIndex.IndexingIndexTotal = utils.AddInt64OrNull(nodeIndex.IndexingIndexTotal, shard.indexing_index_total)
				nodeIndex.IndexingIndexTotalTime = utils.AddInt64OrNull(nodeIndex.IndexingIndexTotalTime, shard.indexing_index_time)
				nodeIndex.MergesTotal = utils.AddInt64OrNull(nodeIndex.MergesTotal, shard.merges_total)
				nodeIndex.MergesTotalTime = utils.AddInt64OrNull(nodeIndex.MergesTotalTime, shard.merges_total_time)
			}

			assignedShard := toAssignedShard(shard)

			switch shard.state {
			case STARTED:
				nodeIndex.AssignShards = append(nodeIndex.AssignShards, assignedShard)
			case INITIALIZING:
				nodeIndex.InitializingShards = append(nodeIndex.InitializingShards, assignedShard)
				nodeIndex.Initializing++
			case RELOCATING:
				nodeIndex.RelocatingShards = append(nodeIndex.RelocatingShards, assignedShard)
				nodeIndex.Relocating++
			}
		} else {
			// if it's already red, we don't need to look
			if status != RED {
				if shard.primary {
					status = RED
				} else {
					status = YELLOW
				}
			}

			nodeIndex.UnassignedShards = append(nodeIndex.UnassignedShards, toUnassignedShard(shard))
			nodeIndex.Unassigned++

			if shard.primary {
				nodeIndex.UnassignedPrimaryShards++
			} else {
				nodeIndex.UnassignedReplicasShards++
			}
		}

		nodeIndexShardsMap[indexNodeId] = nodeIndex
	}
}
