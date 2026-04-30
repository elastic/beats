// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package node_stats

import (
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	cache = utils.EnrichedCache[mapstr.M]{
		Enrichers: []utils.EnrichedType[mapstr.M]{
			// RATES:
			createRate("index_failed_rate_per_second", "indices.indexing.index_failed"),
			createRate("index_rate_per_second", "indices.indexing.index_total"),
			createRate("merge_rate_per_second", "indices.merges.total"),
			createRate("search_rate_per_second", "indices.search.query_total"),
			// LATENCIES:
			createLatency("index_latency_in_millis", "indices.indexing.index_total", "indices.indexing.index_time_in_millis"),
			createLatency("merge_latency_in_millis", "indices.merges.total", "indices.merges.total_time_in_millis"),
			createLatency("search_latency_in_millis", "indices.search.query_total", "indices.search.query_time_in_millis"),
		},
	}
)

// Get the value as an `int64` for the `key`. This assumes that `HasKey` returned `true`.
func getValue(obj *mapstr.M, key string) int64 {
	if value, err := obj.GetValue(key); err == nil {
		if value, ok := value.(int64); ok {
			return value
		}
	}

	return 0
}

// Determine if the `key` exists.
func hasKey(obj *mapstr.M, key string) bool {
	exists, _ := obj.HasKey(key)

	return exists
}

// Set the `value` for the `key`.
func setValue[T any](obj *mapstr.M, key string, value T) {
	(*obj)[key] = value
}

func createLatency(latencyKey string, key string, timestampKey string) utils.EnrichedType[mapstr.M] {
	return utils.EnrichedType[mapstr.M]{
		CalculateValue: utils.CalculateLatency,
		ConvertTime:    utils.UseTimeInMillis,
		GetTime:        func(obj *mapstr.M, _ int64) int64 { return getValue(obj, timestampKey) },
		GetValue:       func(obj *mapstr.M) int64 { return getValue(obj, key) },
		IsUsable:       func(obj *mapstr.M) bool { return hasKey(obj, key) && hasKey(obj, timestampKey) },
		WriteValue:     func(obj *mapstr.M, value float64) { setValue(obj, latencyKey, value) },
	}
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

func enrichNodeStats(id string, nodeStats *mapstr.M, timestampDiff int64) {
	if prevNodeStats, exists := cache.PreviousCache[id]; exists {
		utils.EnrichObject(nodeStats, &prevNodeStats, cache)

		setValue(nodeStats, "timestampDiff", timestampDiff)
	}
}
