// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cat_shards

import (
	"time"

	"golang.org/x/exp/maps"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

var (
	cache = utils.EnrichedCache[NodeIndexShards]{
		Enrichers: []utils.EnrichedType[NodeIndexShards]{
			// RATES:
			{
				CalculateValue: utils.CalculateRate,
				ConvertTime:    utils.MillisToSeconds,
				GetTime:        utils.UseTimestamp[*NodeIndexShards],
				GetValue:       func(obj *NodeIndexShards) int64 { return *obj.GetMissingDocTotal },
				IsUsable:       func(obj *NodeIndexShards) bool { return obj.GetMissingDocTotal != nil },
				WriteValue:     func(obj *NodeIndexShards, value float64) { obj.GetMissingDocRatePerSecond = &value },
			},
			{
				CalculateValue: utils.CalculateRate,
				ConvertTime:    utils.MillisToSeconds,
				GetTime:        utils.UseTimestamp[*NodeIndexShards],
				GetValue:       func(obj *NodeIndexShards) int64 { return *obj.IndexingIndexTotal },
				IsUsable:       func(obj *NodeIndexShards) bool { return obj.IndexingIndexTotal != nil },
				WriteValue:     func(obj *NodeIndexShards, value float64) { obj.IndexRatePerSecond = &value },
			},
			{
				CalculateValue: utils.CalculateRate,
				ConvertTime:    utils.MillisToSeconds,
				GetTime:        utils.UseTimestamp[*NodeIndexShards],
				GetValue:       func(obj *NodeIndexShards) int64 { return *obj.IndexingFailedIndexTotal },
				IsUsable:       func(obj *NodeIndexShards) bool { return obj.IndexingFailedIndexTotal != nil },
				WriteValue:     func(obj *NodeIndexShards, value float64) { obj.IndexFailedRatePerSecond = &value },
			},
			{
				CalculateValue: utils.CalculateRate,
				ConvertTime:    utils.MillisToSeconds,
				GetTime:        utils.UseTimestamp[*NodeIndexShards],
				GetValue:       func(obj *NodeIndexShards) int64 { return *obj.MergesTotal },
				IsUsable:       func(obj *NodeIndexShards) bool { return obj.MergesTotal != nil },
				WriteValue:     func(obj *NodeIndexShards, value float64) { obj.MergeRatePerSecond = &value },
			},
			{
				CalculateValue: utils.CalculateRate,
				ConvertTime:    utils.MillisToSeconds,
				GetTime:        utils.UseTimestamp[*NodeIndexShards],
				GetValue:       func(obj *NodeIndexShards) int64 { return *obj.SearchQueryTotal },
				IsUsable:       func(obj *NodeIndexShards) bool { return obj.SearchQueryTotal != nil },
				WriteValue:     func(obj *NodeIndexShards, value float64) { obj.SearchRatePerSecond = &value },
			},
			// LATENCIES:
			{
				CalculateValue: utils.CalculateLatency,
				ConvertTime:    utils.UseTimeInMillis,
				GetTime:        func(obj *NodeIndexShards, _ int64) int64 { return *obj.IndexingIndexTotalTime },
				GetValue:       func(obj *NodeIndexShards) int64 { return *obj.IndexingIndexTotal },
				IsUsable: func(obj *NodeIndexShards) bool {
					return obj.IndexingIndexTotal != nil && obj.IndexingIndexTotalTime != nil
				},
				WriteValue: func(obj *NodeIndexShards, value float64) { obj.IndexLatencyInMillis = &value },
			},
			{
				CalculateValue: utils.CalculateLatency,
				ConvertTime:    utils.UseTimeInMillis,
				GetTime:        func(obj *NodeIndexShards, _ int64) int64 { return *obj.MergesTotalTime },
				GetValue:       func(obj *NodeIndexShards) int64 { return *obj.MergesTotal },
				IsUsable: func(obj *NodeIndexShards) bool {
					return obj.MergesTotal != nil && obj.MergesTotalTime != nil
				},
				WriteValue: func(obj *NodeIndexShards, value float64) { obj.MergeLatencyInMillis = &value },
			},
			{
				CalculateValue: utils.CalculateLatency,
				ConvertTime:    utils.UseTimeInMillis,
				GetTime:        func(obj *NodeIndexShards, _ int64) int64 { return *obj.SearchQueryTime },
				GetValue:       func(obj *NodeIndexShards) int64 { return *obj.SearchQueryTotal },
				IsUsable: func(obj *NodeIndexShards) bool {
					return obj.SearchQueryTotal != nil && obj.SearchQueryTime != nil
				},
				WriteValue: func(obj *NodeIndexShards, value float64) { obj.SearchLatencyInMillis = &value },
			},
		},
	}
)

func enrichNodeIndexShards(nodeIndexShardsMap map[string]NodeIndexShards, indexMetadata map[string]IndexMetadata) []NodeIndexShards {
	var timestampDiff *int64
	//nolint:gosec // disable G115
	size := int32(len(nodeIndexShardsMap))

	if cache.PreviousCache != nil && cache.PreviousTimestamp != 0 {
		diff := cache.NewTimestamp - cache.PreviousTimestamp
		timestampDiff = &diff
	}

	nodeIndexShardsList := maps.Values(nodeIndexShardsMap)

	for i := range nodeIndexShardsList {
		nodeIndexShardsList[i].TotalFractions = size

		if timestampDiff != nil {
			if prevNodeIndexShards, exists := cache.PreviousCache[nodeIndexShardsList[i].IndexNode]; exists {
				utils.EnrichObject(&nodeIndexShardsList[i], &prevNodeIndexShards, cache)
				nodeIndexShardsList[i].TimestampDiff = timestampDiff
			}
		}

		enrichIndexMetadata(&nodeIndexShardsList[i], indexMetadata)
	}

	return nodeIndexShardsList
}

func enrichIndexMetadata(nodeIndexShards *NodeIndexShards, indexMetadata map[string]IndexMetadata) {
	if metadata, found := indexMetadata[nodeIndexShards.Index]; found {
		nodeIndexShards.Aliases = metadata.aliases
		nodeIndexShards.Attributes = metadata.attributes
		nodeIndexShards.IndexType = &metadata.indexType
		nodeIndexShards.IsHidden = &metadata.hidden
		nodeIndexShards.IsOpen = &metadata.open
		nodeIndexShards.IsSystem = &metadata.system
	}
}

// Convert the indexToShardList into a list of NodeIndexShards docs
func convertToNodeIndexShards(indexToShardList map[string][]Shard, indexMetadata map[string]IndexMetadata) []NodeIndexShards {
	nodeIndexShardsMap := make(map[string]NodeIndexShards, len(cache.PreviousCache))

	// track latest timestamp
	cache.NewTimestamp = time.Now().UnixMilli()

	for index, shards := range indexToShardList {
		indexShardsToNodeIndexShards(nodeIndexShardsMap, index, shards)
	}

	nodeIndexShards := enrichNodeIndexShards(nodeIndexShardsMap, indexMetadata)

	// setup for next execution
	cache.PreviousCache = nodeIndexShardsMap
	cache.PreviousTimestamp = cache.NewTimestamp

	return nodeIndexShards
}
