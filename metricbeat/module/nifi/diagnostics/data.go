package diagnostics

import (
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstriface"
)

var schema = s.Schema{
	"system_diagnostics": c.Dict("systemDiagnostics", s.Schema{
		"aggregate_snapshot": c.Dict("aggregateSnapshot", s.Schema{
			"total_non_heap":         c.Str("totalNonHeap"),
			"total_non_heap_bytes":   c.Int("totalNonHeapBytes"),
			"used_non_heap":          c.Str("usedNonHeap"),
			"used_non_heap_bytes":    c.Int("usedNonHeapBytes"),
			"free_non_heap":          c.Str("freeNonHeap"),
			"free_non_heap_bytes":    c.Int("freeNonHeapBytes"),
			"max_non_heap":           c.Str("maxNonHeap"),
			"max_non_heap_bytes":     c.Int("maxNonHeapBytes"),
			"total_heap":             c.Str("totalHeap"),
			"total_heap_bytes":       c.Int("totalHeapBytes"),
			"used_heap":              c.Str("usedHeap"),
			"used_heap_bytes":        c.Int("usedHeapBytes"),
			"free_heap":              c.Str("freeHeap"),
			"free_heap_bytes":        c.Int("freeHeapBytes"),
			"max_heap":               c.Str("maxHeap"),
			"max_heap_bytes":         c.Int("maxHeapBytes"),
			"heap_utilization":       c.Str("heapUtilization"),
			"available_processors":   c.Int("availableProcessors"),
			"processor_load_average": c.Int("processorLoadAverage"),
			"total_threads":          c.Int("totalThreads"),
			"daemon_threads":         c.Int("daemonThreads"),

			"flow_file_repository_storage_usage": c.Dict("flowFileRepositoryStorageUsage", s.Schema{
				"free_space":        c.Str("freeSpace"),
				"free_space_bytes":  c.Int("freeSpaceBytes"),
				"total_space":       c.Str("totalSpace"),
				"total_space_bytes": c.Int("totalSpaceBytes"),
				"used_space":        c.Str("usedSpace"),
				"used_space_bytes":  c.Int("usedSpaceBytes"),
				"utilization":       c.Str("utilization"),
			}),

			"content_repository_storage_usage": c.Dict("contentRepositoryStorageUsage", s.Schema{
				"identifier":        c.Str("identifier"),
				"free_space":        c.Str("freeSpace"),
				"free_space_bytes":  c.Int("freeSpaceBytes"),
				"total_space":       c.Str("totalSpace"),
				"total_space_bytes": c.Int("totalSpaceBytes"),
				"used_space":        c.Str("usedSpace"),
				"used_space_bytes":  c.Int("usedSpaceBytes"),
				"utilization":       c.Str("utilization"),
			}),

			"garbage_collection": c.Dict("garbageCollection", s.Schema{
				"name":              c.Str("name"),
				"collection_count":  c.Int("collectionCount"),
				"collection_time":   c.Time("collectionTime"),
				"collection_millis": c.Int("collectionMillis"),
			}),

			"status_last_refreshed": c.Str("statusLastRefreshed"),

			"version_info": c.Dict("versionInfo", s.Schema{
				"java_vendor":     c.Str("javaVendor"),
				"java_version":    c.Str("javaVersion"),
				"os_name":         c.Str("osName"),
				"os_version":      c.Str("osVersion"),
				"os_architecture": c.Str("osArchitecture"),
				"build_tag":       c.Str("buildTag"),
				"build_revision":  c.Str("buildRevision"),
				"buildBranch":     c.Str("buildBranch"),
				"build_timestamp": c.Str("buildTimestamp"),
				"nifi_version":    c.Str("nifiVersion"),
			}),
		}),
	}),
}

var eventMapping = schema.Apply
