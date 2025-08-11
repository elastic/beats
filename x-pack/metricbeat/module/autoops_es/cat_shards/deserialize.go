// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cat_shards

import (
	"encoding/json"
)

func toInt32(num json.Number) *int32 {
	if val, err := num.Int64(); err == nil {
		//nolint:gosec // disable G115
		val32 := int32(val)

		return &val32
	}

	return nil
}

func toInt64(num json.Number) *int64 {
	if val, err := num.Int64(); err == nil {
		return &val
	}

	return nil
}

func deserializeShard(jsonShard JSONShard) Shard {
	shard := Shard{
		shard:   *toInt32(jsonShard.ShardId),
		primary: jsonShard.PrimaryOrReplica == "p",
		state:   jsonShard.State,
	}

	// if a shard is UNASSIGNED, then it will only have the index name, shard ID, and primary / replica status
	// and the unassigned reasons
	if shard.state == UNASSIGNED {
		shard.node_id = shard.state
		shard.node_name = shard.state

		// both can be nil:
		shard.unassigned_reason = jsonShard.UnassignedReason
		shard.unassigned_details = jsonShard.UnassignedDetails

		return shard
	}

	shard.node_id = jsonShard.NodeId
	shard.node_name = jsonShard.NodeName

	shard.docs = toInt64(jsonShard.Docs)
	shard.get_missing_time = toInt64(jsonShard.GetMissingTime)
	shard.get_missing_total = toInt64(jsonShard.GetMissingTotal)
	shard.indexing_index_failed = toInt64(jsonShard.IndexingIndexFailed)
	shard.indexing_index_time = toInt64(jsonShard.IndexingIndexTime)
	shard.indexing_index_total = toInt64(jsonShard.IndexingIndexTotal)
	shard.merges_total = toInt64(jsonShard.MergeTotal)
	shard.merges_total_time = toInt64(jsonShard.MergeTotalTime)
	shard.store = toInt64(jsonShard.Store)
	shard.segments_count = toInt64(jsonShard.SegmentsCount)
	shard.search_query_time = toInt64(jsonShard.SearchQueryTime)
	shard.search_query_total = toInt64(jsonShard.SearchQueryTotal)

	return shard
}
