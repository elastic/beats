// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package node_stats

import (
	"errors"
	"fmt"
	"time"

	e "github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	threadPoolStatsSchema = s.Schema{
		"threads":   c.Int("threads", s.IgnoreAllErrors),
		"queue":     c.Int("queue", s.IgnoreAllErrors),
		"rejected":  c.Int("rejected", s.IgnoreAllErrors),
		"completed": c.Int("completed", s.IgnoreAllErrors),
	}

	schema = s.Schema{
		"name":  c.Str("name", s.Required),
		"roles": c.Ifc("roles", s.Optional),
		"host":  c.Str("host", s.Required),
		"ip":    c.Str("ip", s.IgnoreAllErrors),
		"indices": c.Dict("indices", s.Schema{
			"docs": c.Dict("docs", s.Schema{
				"count": c.Int("count", s.IgnoreAllErrors),
			}, c.DictOptional),
			"store": c.Dict("store", s.Schema{
				"size_in_bytes": c.Int("size_in_bytes", s.IgnoreAllErrors),
			}, c.DictOptional),
			"indexing": c.Dict("indexing", s.Schema{
				"index_total":             c.Int("index_total", s.IgnoreAllErrors),
				"index_time_in_millis":    c.Int("index_time_in_millis", s.IgnoreAllErrors),
				"index_failed":            c.Int("index_failed", s.IgnoreAllErrors),
				"delete_time_in_millis":   c.Int("delete_time_in_millis", s.IgnoreAllErrors),
				"delete_total":            c.Int("delete_total", s.IgnoreAllErrors),
				"is_throttled":            c.Bool("is_throttled", s.IgnoreAllErrors),
				"throttle_time_in_millis": c.Int("throttle_time_in_millis", s.IgnoreAllErrors),
			}, c.DictOptional),
			"get": c.Dict("get", s.Schema{
				"missing_total":          c.Int("missing_total", s.IgnoreAllErrors),
				"missing_time_in_millis": c.Int("missing_time_in_millis", s.IgnoreAllErrors),
			}, c.DictOptional),
			"search": c.Dict("search", s.Schema{
				"query_total":          c.Int("query_total", s.IgnoreAllErrors),
				"query_time_in_millis": c.Int("query_time_in_millis", s.IgnoreAllErrors),
			}, c.DictOptional),
			"merges": c.Dict("merges", s.Schema{
				"total":                        c.Int("total", s.IgnoreAllErrors),
				"total_time_in_millis":         c.Int("total_time_in_millis", s.IgnoreAllErrors),
				"total_docs":                   c.Int("total_docs", s.IgnoreAllErrors),
				"total_size_in_bytes":          c.Int("total_size_in_bytes", s.IgnoreAllErrors),
				"total_auto_throttle_in_bytes": c.Int("total_auto_throttle_in_bytes", s.IgnoreAllErrors),
			}, c.DictOptional),
			"query_cache": c.Dict("query_cache", s.Schema{
				"memory_size_in_bytes": c.Int("memory_size_in_bytes", s.IgnoreAllErrors),
				"hit_count":            c.Int("hit_count", s.IgnoreAllErrors),
				"miss_count":           c.Int("miss_count", s.IgnoreAllErrors),
				"evictions":            c.Int("evictions", s.IgnoreAllErrors),
			}, c.DictOptional),
			"fielddata": c.Dict("fielddata", s.Schema{
				"memory_size_in_bytes": c.Int("memory_size_in_bytes", s.IgnoreAllErrors),
				"evictions":            c.Int("evictions", s.IgnoreAllErrors),
			}, c.DictOptional),
			"segments": c.Dict("segments", s.Schema{
				"count":                         c.Int("count", s.IgnoreAllErrors),
				"memory_in_bytes":               c.Int("memory_in_bytes", s.IgnoreAllErrors),
				"terms_memory_in_bytes":         c.Int("terms_memory_in_bytes", s.IgnoreAllErrors),
				"stored_fields_memory_in_bytes": c.Int("stored_fields_memory_in_bytes", s.IgnoreAllErrors),
				"term_vectors_memory_in_bytes":  c.Int("term_vectors_memory_in_bytes", s.IgnoreAllErrors),
				"norms_memory_in_bytes":         c.Int("norms_memory_in_bytes", s.IgnoreAllErrors),
				"points_memory_in_bytes":        c.Int("points_memory_in_bytes", s.IgnoreAllErrors),
				"doc_values_memory_in_bytes":    c.Int("doc_values_memory_in_bytes", s.IgnoreAllErrors),
				"index_writer_memory_in_bytes":  c.Int("index_writer_memory_in_bytes", s.IgnoreAllErrors),
				"version_map_memory_in_bytes":   c.Int("version_map_memory_in_bytes", s.IgnoreAllErrors),
				"fixed_bit_set_memory_in_bytes": c.Int("fixed_bit_set_memory_in_bytes", s.IgnoreAllErrors),
			}, c.DictOptional),
			"request_cache": c.Dict("request_cache", s.Schema{
				"memory_size_in_bytes": c.Int("memory_size_in_bytes", s.IgnoreAllErrors),
				"evictions":            c.Int("evictions", s.IgnoreAllErrors),
				"hit_count":            c.Int("hit_count", s.IgnoreAllErrors),
				"miss_count":           c.Int("miss_count", s.IgnoreAllErrors),
			}, c.DictOptional),
		}, c.DictOptional),
		"os": c.Dict("os", s.Schema{
			"cpu": c.Dict("cpu", s.Schema{
				"load_average": c.Dict("load_average", s.Schema{
					"1m":  c.Float("1m", s.IgnoreAllErrors),
					"5m":  c.Float("5m", s.IgnoreAllErrors),
					"15m": c.Float("15m", s.IgnoreAllErrors),
				}, c.DictOptional), // No load average reported by ES on Windows
			}, c.DictOptional),
			"mem": c.Dict("mem", s.Schema{
				"total_in_bytes": c.Int("total_in_bytes", s.IgnoreAllErrors),
				"used_in_bytes":  c.Int("used_in_bytes", s.IgnoreAllErrors),
			}, c.DictOptional),
			"cgroup": c.Dict("cgroup", s.Schema{
				"cpuacct": c.Dict("cpuacct", s.Schema{
					"control_group": c.Str("control_group", s.IgnoreAllErrors),
					"usage_nanos":   c.Int("usage_nanos", s.IgnoreAllErrors),
				}, c.DictOptional),
				"cpu": c.Dict("cpu", s.Schema{
					"control_group":     c.Str("control_group", s.IgnoreAllErrors),
					"cfs_period_micros": c.Int("cfs_period_micros", s.IgnoreAllErrors),
					"cfs_quota_micros":  c.Int("cfs_quota_micros", s.IgnoreAllErrors),
					"stat": c.Dict("stat", s.Schema{
						"number_of_elapsed_periods": c.Int("number_of_elapsed_periods", s.IgnoreAllErrors),
						"number_of_times_throttled": c.Int("number_of_times_throttled", s.IgnoreAllErrors),
						"time_throttled_nanos":      c.Int("time_throttled_nanos", s.IgnoreAllErrors),
					}, c.DictOptional),
				}, c.DictOptional),
				"memory": c.Dict("memory", s.Schema{
					"control_group": c.Str("control_group", s.IgnoreAllErrors),
					// The two following values are currently string. See https://github.com/elastic/elasticsearch/pull/26166
					"limit_in_bytes": c.Str("limit_in_bytes", s.IgnoreAllErrors),
					"usage_in_bytes": c.Str("usage_in_bytes", s.IgnoreAllErrors),
				}, c.DictOptional),
			}, c.DictOptional),
		}, c.DictOptional),
		"process": c.Dict("process", s.Schema{
			"open_file_descriptors": c.Int("open_file_descriptors", s.IgnoreAllErrors),
			"max_file_descriptors":  c.Int("max_file_descriptors", s.IgnoreAllErrors),
			"cpu": c.Dict("cpu", s.Schema{
				"percent": c.Int("percent", s.IgnoreAllErrors),
			}, c.DictOptional),
		}, c.DictOptional),
		"jvm": c.Dict("jvm", s.Schema{
			"uptime_in_millis": c.Int("uptime_in_millis", s.IgnoreAllErrors),
			"mem": c.Dict("mem", s.Schema{
				"heap_used_in_bytes":          c.Int("heap_used_in_bytes", s.IgnoreAllErrors),
				"heap_used_percent":           c.Int("heap_used_percent", s.IgnoreAllErrors),
				"heap_max_in_bytes":           c.Int("heap_max_in_bytes", s.IgnoreAllErrors),
				"non_heap_committed_in_bytes": c.Int("non_heap_committed_in_bytes", s.IgnoreAllErrors),
			}, c.DictOptional),
			"gc": c.Dict("gc", s.Schema{
				"collectors": c.Dict("collectors", s.Schema{
					"young": c.Dict("young", s.Schema{
						"collection_count":          c.Int("collection_count", s.IgnoreAllErrors),
						"collection_time_in_millis": c.Int("collection_time_in_millis", s.IgnoreAllErrors),
					}, c.DictOptional),
					"old": c.Dict("old", s.Schema{
						"collection_count":          c.Int("collection_count", s.IgnoreAllErrors),
						"collection_time_in_millis": c.Int("collection_time_in_millis", s.IgnoreAllErrors),
					}, c.DictOptional),
				}),
			}, c.DictOptional),
		}, c.DictOptional),
		"thread_pool": c.Dict("thread_pool", s.Schema{
			"generic":    c.Dict("generic", threadPoolStatsSchema, c.DictOptional),
			"get":        c.Dict("get", threadPoolStatsSchema, c.DictOptional),
			"management": c.Dict("management", threadPoolStatsSchema, c.DictOptional),
			"search":     c.Dict("search", threadPoolStatsSchema, c.DictOptional),
			"watcher":    c.Dict("watcher", threadPoolStatsSchema, c.DictOptional),
			"write":      c.Dict("write", threadPoolStatsSchema, c.DictOptional),
		}),
		"fs": c.Dict("fs", s.Schema{
			"total": c.Dict("total", s.Schema{
				"total_in_bytes":     c.Int("total_in_bytes", s.IgnoreAllErrors),
				"free_in_bytes":      c.Int("free_in_bytes", s.IgnoreAllErrors),
				"available_in_bytes": c.Int("available_in_bytes", s.IgnoreAllErrors),
			}, c.DictOptional),
			"io_stats": c.Dict("io_stats", s.Schema{
				"total": c.Dict("total", s.Schema{
					"operations":       c.Int("operations", s.IgnoreAllErrors),
					"read_kilobytes":   c.Int("read_kilobytes", s.IgnoreAllErrors),
					"read_operations":  c.Int("read_operations", s.IgnoreAllErrors),
					"write_kilobytes":  c.Int("write_kilobytes", s.IgnoreAllErrors),
					"write_operations": c.Int("write_operations", s.IgnoreAllErrors),
				}, c.DictOptional),
			}, c.DictOptional),
		}),
		"transport": c.Dict("transport", s.Schema{
			"rx_count":         c.Int("rx_count", s.IgnoreAllErrors),
			"rx_size_in_bytes": c.Int("rx_size_in_bytes", s.IgnoreAllErrors),
			"tx_count":         c.Int("tx_count", s.IgnoreAllErrors),
			"tx_size_in_bytes": c.Int("tx_size_in_bytes", s.IgnoreAllErrors),
		}, c.DictOptional),
		"breakers": c.Dict("breakers", s.Schema{
			"request": c.Dict("request", s.Schema{
				"tripped":                 c.Int("tripped", s.IgnoreAllErrors),
				"estimated_size_in_bytes": c.Int("estimated_size_in_bytes", s.IgnoreAllErrors),
				"limit_size_in_bytes":     c.Int("limit_size_in_bytes", s.IgnoreAllErrors),
			}, c.DictOptional),
			"parent": c.Dict("parent", s.Schema{
				"tripped":                 c.Int("tripped", s.IgnoreAllErrors),
				"estimated_size_in_bytes": c.Int("estimated_size_in_bytes", s.IgnoreAllErrors),
				"limit_size_in_bytes":     c.Int("limit_size_in_bytes", s.IgnoreAllErrors),
			}, c.DictOptional),
			"fielddata": c.Dict("fielddata", s.Schema{
				"tripped":                 c.Int("tripped", s.IgnoreAllErrors),
				"estimated_size_in_bytes": c.Int("estimated_size_in_bytes", s.IgnoreAllErrors),
				"limit_size_in_bytes":     c.Int("limit_size_in_bytes", s.IgnoreAllErrors),
			}, c.DictOptional),
		}, c.DictOptional),
		"http": c.Dict("http", s.Schema{
			"current_open": c.Int("current_open", s.IgnoreAllErrors),
			"total_opened": c.Int("total_opened", s.IgnoreAllErrors),
		}, c.DictOptional),
	}
)

type ClusterStateMasterNode struct {
	MasterNodeId string `json:"master_node"`
}

type NodesStats struct {
	Nodes map[string]map[string]interface{} `json:"nodes"`
}

// Get the elected master node's ID
func GetMasterNodeId(m *elasticsearch.MetricSet) (string, error) {
	data, err := utils.FetchAPIData[ClusterStateMasterNode](m, ClusterStateMasterNodePath)

	if err != nil {
		return "", err
	}

	return data.MasterNodeId, nil
}

func eventsMapping(m *elasticsearch.MetricSet, r mb.ReporterV2, info *utils.ClusterInfo, nodeStats *NodesStats) error {
	var errs []error

	timestampDiff := int64(0)
	enrichedStats := map[string]mapstr.M{}
	transactionId := utils.NewUUIDV4()
	events := []mb.Event{}
	nodesList := make(map[string]string, len(nodeStats.Nodes))

	masterNodeId, err := GetMasterNodeId(m)

	// we simply won't know the master node
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to get master node id: %w", err))
	}

	// track latest timestamp
	cache.NewTimestamp = time.Now().UnixMilli()

	if cache.PreviousCache != nil && cache.PreviousTimestamp != 0 {
		timestampDiff = cache.NewTimestamp - cache.PreviousTimestamp
	}

	for id, node := range nodeStats.Nodes {
		metricSet, err := schema.Apply(node)

		if err != nil {
			errs = append(errs, fmt.Errorf("failed applying nodes_stats schema for %v: %w", id, err))
			continue
		}

		metricSet["id"] = id
		metricSet["is_elected_master"] = id == masterNodeId

		// schema requires the name as a string
		name, ok := metricSet["name"].(string)

		if !ok {
			errs = append(errs, fmt.Errorf("failed to get node name: %v", metricSet["name"]))
			continue
		}

		nodesList[id] = name

		if timestampDiff != 0 {
			enrichNodeStats(id, &metricSet, timestampDiff)
		}

		// remember the metricset for the next pass
		enrichedStats[id] = metricSet

		event := e.CreateEvent(info, metricSet, transactionId)

		// TODO: Update the indexer to use these from the metricset and remove this (then we can use utils.CreateAndReportEvents)
		event.ModuleFields["node"] = mapstr.M{
			"id":                id,
			"name":              name,
			"host":              node["host"],
			"is_elected_master": id == masterNodeId,
			"roles":             node["roles"],
		}

		events = append(events, event)
	}

	// replace the cache with the current data for the next run
	cache.PreviousCache = enrichedStats
	cache.PreviousTimestamp = cache.NewTimestamp

	e.ReportEvents(r, events)

	event := e.CreateEvent(info, mapstr.M{"nodes": nodesList, "subType": "list"}, transactionId)

	// TODO: Update the indexer to use these from the metricset and remove this
	event.RootFields["subType"] = "list"
	event.ModuleFields["nodes"] = nodesList

	r.Event(event)

	err = errors.Join(errs...)

	if err != nil {
		e.SendErrorEvent(err, info, r, NodesStatsMetricSet, NodesStatsPath, transactionId)
	}
	return err
}
