// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package frozen_cache_stats

import (
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	e "github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type SharedCacheStats struct {
	Reads               int64 `json:"reads"`
	Writes              int64 `json:"writes"`
	Evictions           int64 `json:"evictions"`
	SizeInBytes         int64 `json:"size_in_bytes"`
	RegionSizeInBytes   int64 `json:"region_size_in_bytes"`
	NumRegions          int64 `json:"num_regions"`
	BytesWrittenInBytes int64 `json:"bytes_written_in_bytes"`
	BytesReadInBytes    int64 `json:"bytes_read_in_bytes"`
}

type NodeCacheStats struct {
	SharedCache SharedCacheStats `json:"shared_cache"`
}

type CacheStatsResponse struct {
	Nodes map[string]NodeCacheStats `json:"nodes"`
}

func toMapStr(id string, node NodeCacheStats) mapstr.M {
	return mapstr.M{
		"id": id,
		"shared_cache": mapstr.M{
			"reads":                  node.SharedCache.Reads,
			"writes":                 node.SharedCache.Writes,
			"evictions":              node.SharedCache.Evictions,
			"size_in_bytes":          node.SharedCache.SizeInBytes,
			"region_size_in_bytes":   node.SharedCache.RegionSizeInBytes,
			"num_regions":            node.SharedCache.NumRegions,
			"bytes_written_in_bytes": node.SharedCache.BytesWrittenInBytes,
			"bytes_read_in_bytes":    node.SharedCache.BytesReadInBytes,
		},
	}
}

func eventsMapping(r mb.ReporterV2, info *utils.ClusterInfo, response *CacheStatsResponse) error {
	if len(response.Nodes) == 0 {
		return nil
	}

	transactionId := utils.NewUUID()
	metricSets := make([]mapstr.M, 0, len(response.Nodes))
	enrichedStats := make(map[string]mapstr.M, len(response.Nodes))
	timestampDiff := int64(0)

	frozenCacheStatsCache.NewTimestamp = time.Now().UnixMilli()

	if frozenCacheStatsCache.PreviousCache != nil && frozenCacheStatsCache.PreviousTimestamp != 0 {
		timestampDiff = frozenCacheStatsCache.NewTimestamp - frozenCacheStatsCache.PreviousTimestamp
	}

	for id, node := range response.Nodes {
		if node.SharedCache.NumRegions == 0 && node.SharedCache.SizeInBytes == 0 {
			continue
		}

		metricSet := toMapStr(id, node)

		if timestampDiff != 0 {
			enrichNodeCacheStats(id, &metricSet, timestampDiff)
		}

		enrichedStats[id] = metricSet
		metricSets = append(metricSets, metricSet)
	}

	frozenCacheStatsCache.PreviousCache = enrichedStats
	frozenCacheStatsCache.PreviousTimestamp = frozenCacheStatsCache.NewTimestamp

	e.CreateAndReportEvents(r, info, metricSets, transactionId)

	return nil
}
