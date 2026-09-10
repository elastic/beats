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
			createRate("ingest_docs_per_second", "indices.docs.count"),
			createRate("ingest_bytes_per_second", "indices.store.size_in_bytes"),
			createRate("bulk_bytes_per_second", "indices.bulk.total_size_in_bytes"),
			createRate("bulk_operations_per_second", "indices.bulk.total_operations"),
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
		MaxValueMillis: utils.SamplingIntervalMillis[mapstr.M],
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

const (
	cgroupUsageNanosKey  = "os.cgroup.cpuacct.usage_nanos"
	cgroupPeriodsKey     = "os.cgroup.cpu.stat.number_of_elapsed_periods"
	cgroupQuotaMicrosKey = "os.cgroup.cpu.cfs_quota_micros"
	cgroupCpuPercentKey  = "os.cgroup.cpu.usage_percent"
)

// Derive the percentage of the configured CFS quota that the process cgroup consumed
// between two consecutive samples:
//
//	(Δusage_nanos / (Δnumber_of_elapsed_periods × cfs_quota_micros × 1000)) × 100
//
// This is emitted *alongside* `process.cpu.percent`, never in place of it: the latter
// reports host-level CPU usage, which is misleading when a CFS quota caps the process,
// while this field is relative to that quota. Keeping both means consumers can compare
// the two views instead of guessing which one a sample represents.
//
// The value is deliberately not clamped to 100: a cgroup can burst above its quota
// within a sampling window, and that is meaningful signal rather than an error.
func enrichCgroupCpuUsagePercent(node *mapstr.M, prevNode *mapstr.M) {
	if !hasKey(node, cgroupUsageNanosKey) || !hasKey(prevNode, cgroupUsageNanosKey) ||
		!hasKey(node, cgroupPeriodsKey) || !hasKey(prevNode, cgroupPeriodsKey) ||
		!hasKey(node, cgroupQuotaMicrosKey) {
		return
	}
	quotaMicros := getValue(node, cgroupQuotaMicrosKey)
	if quotaMicros <= 0 {
		return
	}
	usageDelta := getValue(node, cgroupUsageNanosKey) - getValue(prevNode, cgroupUsageNanosKey)
	periodsDelta := getValue(node, cgroupPeriodsKey) - getValue(prevNode, cgroupPeriodsKey)
	if usageDelta < 0 || periodsDelta <= 0 {
		return
	}
	percent := float64(usageDelta) / (float64(periodsDelta) * float64(quotaMicros) * 1000) * 100

	// `setValue` writes a literal flat key; use `Put` so the value lands inside the
	// nested `os.cgroup.cpu` object that the schema already produces.
	_, _ = node.Put(cgroupCpuPercentKey, percent)
}

func enrichNodeStats(id string, nodeStats *mapstr.M, timestampDiff int64) {
	if prevNodeStats, exists := cache.PreviousCache[id]; exists {
		utils.EnrichObject(nodeStats, &prevNodeStats, cache)
		enrichCgroupCpuUsagePercent(nodeStats, &prevNodeStats)

		setValue(nodeStats, "timestampDiff", timestampDiff)
	}
}
