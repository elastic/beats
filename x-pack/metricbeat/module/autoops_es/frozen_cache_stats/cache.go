// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package frozen_cache_stats

import (
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	frozenCacheStatsCache = utils.EnrichedCache[mapstr.M]{
		Enrichers: []utils.EnrichedType[mapstr.M]{
			createRate("shared_cache_reads_per_second", "shared_cache.reads"),
			createRate("shared_cache_writes_per_second", "shared_cache.writes"),
			createRate("shared_cache_evictions_per_second", "shared_cache.evictions"),
			createRate("shared_cache_bytes_written_per_second", "shared_cache.bytes_written_in_bytes"),
			createRate("shared_cache_bytes_read_per_second", "shared_cache.bytes_read_in_bytes"),
		},
	}
)

func getValue(obj *mapstr.M, key string) int64 {
	if value, err := obj.GetValue(key); err == nil {
		if value, ok := value.(int64); ok {
			return value
		}
	}
	return 0
}

func hasKey(obj *mapstr.M, key string) bool {
	exists, _ := obj.HasKey(key)
	return exists
}

func setValue[T any](obj *mapstr.M, key string, value T) {
	(*obj)[key] = value
}

func createRate(rateKey string, key string) utils.EnrichedType[mapstr.M] {
	return utils.EnrichedType[mapstr.M]{
		CalculateValue: utils.CalculateRate,
		ConvertTime:    utils.MillisToSeconds,
		GetTime:        utils.UseTimestamp[*mapstr.M],
		GetValue:       func(obj *mapstr.M) int64 { return getValue(obj, key) },
		IsUsable:       func(obj *mapstr.M) bool { return hasKey(obj, key) },
		WriteValue:     func(obj *mapstr.M, value float64) { setValue(obj, rateKey, value) },
	}
}

func enrichNodeCacheStats(id string, nodeStats *mapstr.M, timestampDiff int64) {
	if prevStats, exists := frozenCacheStatsCache.PreviousCache[id]; exists {
		utils.EnrichObject(nodeStats, &prevStats, frozenCacheStatsCache)
		setValue(nodeStats, "timestampDiff", timestampDiff)
	}
}
