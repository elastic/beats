// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cat_shards

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v9/x-pack/metricbeat/module/autoops_es/utils"
)

type jsonNumberTestType struct {
	JsonNumber json.Number `json:"n"`
}

func getJsonNumber(t *testing.T, json string) json.Number {
	data, err := utils.DeserializeData[jsonNumberTestType]([]byte(json))

	require.NoError(t, err)

	return data.JsonNumber
}

func TestToInt32ReturnsNil(t *testing.T) {
	nilValue := getJsonNumber(t, `{}`)
	explicitNilValue := getJsonNumber(t, `{"n":null}`)
	floatValue := getJsonNumber(t, `{"n":123.456}`)

	require.Nil(t, toInt32(nilValue))
	require.Nil(t, toInt32(explicitNilValue))
	require.Nil(t, toInt32(floatValue))
}

func TestToInt32ReturnsValue(t *testing.T) {
	require.EqualValues(t, 0, *toInt32("0"))
	require.EqualValues(t, 1, *toInt32("1"))
	require.EqualValues(t, -1, *toInt32("-1"))
	require.EqualValues(t, 123, *toInt32("123"))
	require.EqualValues(t, -2147483648, *toInt32("-2147483648"))
	require.EqualValues(t, 2147483647, *toInt32("2147483647"))
}

func TestToInt64ReturnsNil(t *testing.T) {
	nilValue := getJsonNumber(t, `{}`)
	explicitNilValue := getJsonNumber(t, `{"n":null}`)
	floatValue := getJsonNumber(t, `{"n":123.456}`)

	require.Nil(t, toInt32(nilValue))
	require.Nil(t, toInt32(explicitNilValue))
	require.Nil(t, toInt32(floatValue))
}

func TestToInt64ReturnsValue(t *testing.T) {
	require.EqualValues(t, 0, *toInt64("0"))
	require.EqualValues(t, 1, *toInt64("1"))
	require.EqualValues(t, -1, *toInt64("-1"))
	require.EqualValues(t, 123, *toInt64("123"))
	require.EqualValues(t, -9223372036854775808, *toInt64("-9223372036854775808"))
	require.EqualValues(t, 9223372036854775807, *toInt64("9223372036854775807"))
}

func TestDeserializeShardUnassigned(t *testing.T) {
	unassignedReason := "reason"
	unassignedDetails := "details"

	unassignedShard1 := deserializeShard(JSONShard{
		ShardId:          "123",
		PrimaryOrReplica: "p",
		State:            UNASSIGNED,
	})

	require.EqualValues(t, 123, unassignedShard1.shard)
	require.Equal(t, true, unassignedShard1.primary)
	require.Equal(t, UNASSIGNED, unassignedShard1.state)
	require.Equal(t, UNASSIGNED, unassignedShard1.node_id)
	require.Equal(t, UNASSIGNED, unassignedShard1.node_name)
	require.Nil(t, unassignedShard1.unassigned_reason)
	require.Nil(t, unassignedShard1.unassigned_details)

	unassignedShard2 := deserializeShard(JSONShard{
		ShardId:          "456",
		PrimaryOrReplica: "r",
		State:            UNASSIGNED,
		UnassignedReason: &unassignedReason,
	})

	require.EqualValues(t, 456, unassignedShard2.shard)
	require.Equal(t, false, unassignedShard2.primary)
	require.Equal(t, UNASSIGNED, unassignedShard2.state)
	require.Equal(t, UNASSIGNED, unassignedShard2.node_id)
	require.Equal(t, UNASSIGNED, unassignedShard2.node_name)
	require.Same(t, &unassignedReason, unassignedShard2.unassigned_reason)
	require.Nil(t, unassignedShard2.unassigned_details)

	unassignedShard3 := deserializeShard(JSONShard{
		ShardId:           "789",
		PrimaryOrReplica:  "p",
		State:             UNASSIGNED,
		UnassignedDetails: &unassignedDetails,
	})

	require.EqualValues(t, 789, unassignedShard3.shard)
	require.Equal(t, true, unassignedShard3.primary)
	require.Equal(t, UNASSIGNED, unassignedShard3.state)
	require.Equal(t, UNASSIGNED, unassignedShard3.node_id)
	require.Equal(t, UNASSIGNED, unassignedShard3.node_name)
	require.Nil(t, unassignedShard3.unassigned_reason)
	require.Same(t, &unassignedDetails, unassignedShard3.unassigned_details)

	unassignedShard4 := deserializeShard(JSONShard{
		ShardId:           "0",
		PrimaryOrReplica:  "r",
		State:             UNASSIGNED,
		UnassignedReason:  &unassignedReason,
		UnassignedDetails: &unassignedDetails,
	})

	require.EqualValues(t, 0, unassignedShard4.shard)
	require.Equal(t, false, unassignedShard4.primary)
	require.Equal(t, UNASSIGNED, unassignedShard4.state)
	require.Equal(t, UNASSIGNED, unassignedShard4.node_id)
	require.Equal(t, UNASSIGNED, unassignedShard4.node_name)
	require.Same(t, &unassignedReason, unassignedShard4.unassigned_reason)
	require.Same(t, &unassignedDetails, unassignedShard4.unassigned_details)
}

func TestDeserializeShard(t *testing.T) {
	shard1 := deserializeShard(JSONShard{
		NodeId:   "node1",
		NodeName: "name1",

		ShardId:          "123",
		PrimaryOrReplica: "p",
		State:            STARTED,

		Docs:                "123456",
		GetMissingTime:      "1",
		GetMissingTotal:     "2",
		IndexingIndexFailed: "3",
		IndexingIndexTime:   "4",
		IndexingIndexTotal:  "5",
		MergeTotal:          "6",
		MergeTotalTime:      "7",
		Store:               "8",
		SegmentsCount:       "9",
		SearchQueryTime:     "10",
		SearchQueryTotal:    "11",
	})

	require.EqualValues(t, 123, shard1.shard)
	require.Equal(t, true, shard1.primary)
	require.Equal(t, STARTED, shard1.state)
	require.Equal(t, "node1", shard1.node_id)
	require.Equal(t, "name1", shard1.node_name)
	require.Nil(t, shard1.unassigned_reason)
	require.Nil(t, shard1.unassigned_details)
	require.EqualValues(t, 123456, *shard1.docs)
	require.EqualValues(t, 1, *shard1.get_missing_time)
	require.EqualValues(t, 2, *shard1.get_missing_total)
	require.EqualValues(t, 3, *shard1.indexing_index_failed)
	require.EqualValues(t, 4, *shard1.indexing_index_time)
	require.EqualValues(t, 5, *shard1.indexing_index_total)
	require.EqualValues(t, 6, *shard1.merges_total)
	require.EqualValues(t, 7, *shard1.merges_total_time)
	require.EqualValues(t, 8, *shard1.store)
	require.EqualValues(t, 9, *shard1.segments_count)
	require.EqualValues(t, 10, *shard1.search_query_time)
	require.EqualValues(t, 11, *shard1.search_query_total)

	shard2 := deserializeShard(JSONShard{
		NodeId:   "node2",
		NodeName: "name2",

		ShardId:          "456",
		PrimaryOrReplica: "r",
		State:            INITIALIZING,
	})

	require.EqualValues(t, 456, shard2.shard)
	require.Equal(t, false, shard2.primary)
	require.Equal(t, INITIALIZING, shard2.state)
	require.Equal(t, "node2", shard2.node_id)
	require.Equal(t, "name2", shard2.node_name)
	require.Nil(t, shard2.unassigned_reason)
	require.Nil(t, shard2.unassigned_details)
	require.Nil(t, shard2.docs)
	require.Nil(t, shard2.get_missing_time)
	require.Nil(t, shard2.get_missing_total)
	require.Nil(t, shard2.indexing_index_failed)
	require.Nil(t, shard2.indexing_index_time)
	require.Nil(t, shard2.indexing_index_total)
	require.Nil(t, shard2.merges_total)
	require.Nil(t, shard2.merges_total_time)
	require.Nil(t, shard2.store)
	require.Nil(t, shard2.segments_count)
	require.Nil(t, shard2.search_query_time)
	require.Nil(t, shard2.search_query_total)

	shard3 := deserializeShard(JSONShard{
		NodeId:   "node3",
		NodeName: "name3",

		ShardId:          "1",
		PrimaryOrReplica: "p",
		State:            RELOCATING,

		// It's valid for a mix of values to be null
		Docs:          "1",
		Store:         "2",
		SegmentsCount: "3",
	})

	require.EqualValues(t, 1, shard3.shard)
	require.Equal(t, true, shard3.primary)
	require.Equal(t, RELOCATING, shard3.state)
	require.Equal(t, "node3", shard3.node_id)
	require.Equal(t, "name3", shard3.node_name)
	require.Nil(t, shard3.unassigned_reason)
	require.Nil(t, shard3.unassigned_details)
	require.EqualValues(t, 1, *shard3.docs)
	require.EqualValues(t, 2, *shard3.store)
	require.EqualValues(t, 3, *shard3.segments_count)
	require.Nil(t, shard3.get_missing_time)
	require.Nil(t, shard3.get_missing_total)
	require.Nil(t, shard3.indexing_index_failed)
	require.Nil(t, shard3.indexing_index_time)
	require.Nil(t, shard3.indexing_index_total)
	require.Nil(t, shard3.merges_total)
	require.Nil(t, shard3.merges_total_time)
	require.Nil(t, shard3.search_query_time)
	require.Nil(t, shard3.search_query_total)
}
