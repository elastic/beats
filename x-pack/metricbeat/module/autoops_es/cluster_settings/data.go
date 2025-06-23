// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cluster_settings

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

var (
	clusterSchema = c.Dict("cluster", s.Schema{
		"metadata": c.Dict("metadata", s.Schema{
			"display_name": c.Str("display_name", s.IgnoreAllErrors),
		}, c.DictOptional),
		"name":                       c.Str("name", s.IgnoreAllErrors),
		"max_shards_per_node":        c.Ifc("max_shards_per_node", s.IgnoreAllErrors),
		"max_shards_per_node_frozen": c.Str("max_shards_per_node.frozen", s.IgnoreAllErrors),
		"routing": c.Dict("routing", s.Schema{
			"allocation": c.Dict("allocation", s.Schema{
				"disk": c.Dict("disk", s.Schema{
					"watermark": c.Dict("watermark", s.Schema{
						"low":         c.Str("low", s.IgnoreAllErrors),
						"high":        c.Str("high", s.IgnoreAllErrors),
						"flood_stage": c.Str("flood_stage", s.IgnoreAllErrors),
					}, c.DictOptional),
				}, c.DictOptional),
				"node_concurrent_outgoing_recoveries": c.Str("node_concurrent_outgoing_recoveries", s.IgnoreAllErrors),
				"cluster_concurrent_rebalance":        c.Str("cluster_concurrent_rebalance", s.IgnoreAllErrors),
				"node_concurrent_recoveries":          c.Str("node_concurrent_recoveries", s.IgnoreAllErrors),
				"total_shards_per_node":               c.Str("total_shards_per_node", s.IgnoreAllErrors),
				"exclude": c.Dict("exclude", s.Schema{
					"_ip":   c.Str("_ip", s.IgnoreAllErrors),
					"_host": c.Str("_host", s.IgnoreAllErrors),
					"_name": c.Str("_name", s.IgnoreAllErrors),
				}, c.DictOptional),
				"include": c.Dict("include", s.Schema{
					"_ip":   c.Str("_ip", s.IgnoreAllErrors),
					"_host": c.Str("_host", s.IgnoreAllErrors),
					"_name": c.Str("_name", s.IgnoreAllErrors),
				}, c.DictOptional),
			}, c.DictOptional),
		}, c.DictOptional),
		"blocks": c.Dict("blocks", s.Schema{
			"read_only":              c.Str("read_only", s.IgnoreAllErrors),
			"create_index":           c.Str("create_index", s.IgnoreAllErrors),
			"read_only_allow_delete": c.Str("read_only_allow_delete", s.IgnoreAllErrors),
		}, c.DictOptional),
	}, c.DictOptional)

	schema = s.Schema{
		"defaults": c.Dict("defaults", s.Schema{
			"path": c.Dict("path", s.Schema{
				"data": c.Ifc("data", s.Optional),
			}, c.DictOptional),
			"serverless": c.Dict("serverless", s.Schema{
				"search": c.Dict("search", s.Schema{
					"boost_window":     c.Str("boost_window", s.IgnoreAllErrors),
					"search_power_max": c.Str("search_power_max", s.IgnoreAllErrors),
					"search_power_min": c.Str("search_power_min", s.IgnoreAllErrors),
				}, c.DictOptional),
			}, c.DictOptional),
			"discovery": c.Dict("discovery", s.Schema{
				"zen": c.Dict("zen", s.Schema{
					"minimum_master_nodes": c.Str("minimum_master_nodes", s.IgnoreAllErrors),
				}, c.DictOptional),
			}, c.DictOptional),
			"processors": c.Str("processors", s.IgnoreAllErrors),
			"cluster":    clusterSchema,
			"repositories": c.Dict("repositories", s.Schema{
				"fs": c.Dict("fs", s.Schema{
					"compress":   c.Str("compress", s.IgnoreAllErrors),
					"chunk_size": c.Str("chunk_size", s.IgnoreAllErrors),
					"location":   c.Str("location", s.IgnoreAllErrors),
				}, c.DictOptional),
				"url": c.Dict("url", s.Schema{
					"url": c.Str("url", s.IgnoreAllErrors),
				}, c.DictOptional),
			}, c.DictOptional),
			"bootstrap": c.Dict("bootstrap", s.Schema{
				"memory_lock": c.Str("memory_lock", s.IgnoreAllErrors),
			}, c.DictOptional),
			"search": c.Dict("search", s.Schema{
				"default_search_timeout": c.Str("default_search_timeout", s.IgnoreAllErrors),
				"max_buckets":            c.Str("max_buckets", s.IgnoreAllErrors),
			}, c.DictOptional),
			"indices": c.Dict("indices", s.Schema{
				"recovery": c.Dict("recovery", s.Schema{
					"max_bytes_per_sec": c.Str("max_bytes_per_sec", s.IgnoreAllErrors),
				}, c.DictOptional),
				"breaker": c.Dict("breaker", s.Schema{
					"request": c.Dict("request", s.Schema{
						"limit": c.Str("limit", s.IgnoreAllErrors),
					}, c.DictOptional),
					"total": c.Dict("total", s.Schema{
						"limit": c.Str("limit", s.IgnoreAllErrors),
					}, c.DictOptional),
					"fielddata": c.Dict("fielddata", s.Schema{
						"limit": c.Str("limit", s.IgnoreAllErrors),
					}, c.DictOptional),
				}, c.DictOptional),
				"query": c.Dict("query", s.Schema{
					"query_string": c.Dict("query_string", s.Schema{
						"allowLeadingWildcard": c.Str("allowLeadingWildcard", s.IgnoreAllErrors),
					}, c.DictOptional),
				}, c.DictOptional),
			}, c.DictOptional),
			"action": c.Dict("action", s.Schema{
				"destructive_requires_name": c.Str("destructive_requires_name", s.IgnoreAllErrors),
			}, c.DictOptional),
		}, c.DictRequired),
		"persistent": c.Dict("persistent", s.Schema{
			"serverless": c.Dict("serverless", s.Schema{
				"search": c.Dict("search", s.Schema{
					"boost_window":     c.Str("boost_window", s.IgnoreAllErrors),
					"search_power_max": c.Str("search_power_max", s.IgnoreAllErrors),
					"search_power_min": c.Str("search_power_min", s.IgnoreAllErrors),
				}, c.DictOptional),
			}, c.DictOptional),
			"discovery": c.Dict("discovery", s.Schema{
				"zen": c.Dict("zen", s.Schema{
					"minimum_master_nodes": c.Str("minimum_master_nodes", s.IgnoreAllErrors),
				}, c.DictOptional),
			}, c.DictOptional),
			"processors": c.Str("processors", s.IgnoreAllErrors),
			"cluster":    clusterSchema,
			"bootstrap": c.Dict("bootstrap", s.Schema{
				"memory_lock": c.Str("memory_lock", s.IgnoreAllErrors),
			}, c.DictOptional),
			"search": c.Dict("search", s.Schema{
				"default_search_timeout": c.Str("default_search_timeout", s.IgnoreAllErrors),
				"max_buckets":            c.Str("max_buckets", s.IgnoreAllErrors),
			}, c.DictOptional),
			"indices": c.Dict("indices", s.Schema{
				"recovery": c.Dict("recovery", s.Schema{
					"max_bytes_per_sec": c.Str("max_bytes_per_sec", s.IgnoreAllErrors),
				}, c.DictOptional),
				"breaker": c.Dict("breaker", s.Schema{
					"request": c.Dict("request", s.Schema{
						"limit": c.Str("limit", s.IgnoreAllErrors),
					}, c.DictOptional),
					"total": c.Dict("total", s.Schema{
						"limit": c.Str("limit", s.IgnoreAllErrors),
					}, c.DictOptional),
					"fielddata": c.Dict("fielddata", s.Schema{
						"limit": c.Str("limit", s.IgnoreAllErrors),
					}, c.DictOptional),
				}, c.DictOptional),
				"query": c.Dict("query", s.Schema{
					"query_string": c.Dict("query_string", s.Schema{
						"allowLeadingWildcard": c.Str("allowLeadingWildcard", s.IgnoreAllErrors),
					}, c.DictOptional),
				}, c.DictOptional),
			}, c.DictOptional),
			"action": c.Dict("action", s.Schema{
				"destructive_requires_name": c.Str("destructive_requires_name", s.IgnoreAllErrors),
			}, c.DictOptional),
		}, c.DictOptional),
		"transient": c.Dict("transient", s.Schema{
			"serverless": c.Dict("serverless", s.Schema{
				"search": c.Dict("search", s.Schema{
					"boost_window":     c.Str("boost_window", s.IgnoreAllErrors),
					"search_power_max": c.Str("search_power_max", s.IgnoreAllErrors),
					"search_power_min": c.Str("search_power_min", s.IgnoreAllErrors),
				}, c.DictOptional),
			}, c.DictOptional),
			"discovery": c.Dict("discovery", s.Schema{
				"zen": c.Dict("zen", s.Schema{
					"minimum_master_nodes": c.Str("minimum_master_nodes", s.IgnoreAllErrors),
				}, c.DictOptional),
			}, c.DictOptional),
			"processors": c.Str("processors", s.IgnoreAllErrors),
			"cluster":    clusterSchema,
			"bootstrap": c.Dict("bootstrap", s.Schema{
				"memory_lock": c.Str("memory_lock", s.IgnoreAllErrors),
			}, c.DictOptional),
			"search": c.Dict("search", s.Schema{
				"default_search_timeout": c.Str("default_search_timeout", s.IgnoreAllErrors),
				"max_buckets":            c.Str("max_buckets", s.IgnoreAllErrors),
			}, c.DictOptional),
			"indices": c.Dict("indices", s.Schema{
				"recovery": c.Dict("recovery", s.Schema{
					"max_bytes_per_sec": c.Str("max_bytes_per_sec", s.IgnoreAllErrors),
				}, c.DictOptional),
				"breaker": c.Dict("breaker", s.Schema{
					"request": c.Dict("request", s.Schema{
						"limit": c.Str("limit", s.IgnoreAllErrors),
					}, c.DictOptional),
					"total": c.Dict("total", s.Schema{
						"limit": c.Str("limit", s.IgnoreAllErrors),
					}, c.DictOptional),
					"fielddata": c.Dict("fielddata", s.Schema{
						"limit": c.Str("limit", s.IgnoreAllErrors),
					}, c.DictOptional),
				}, c.DictOptional),
				"query": c.Dict("query", s.Schema{
					"query_string": c.Dict("query_string", s.Schema{
						"allowLeadingWildcard": c.Str("allowLeadingWildcard", s.IgnoreAllErrors),
					}, c.DictOptional),
				}, c.DictOptional),
			}, c.DictOptional),
			"action": c.Dict("action", s.Schema{
				"destructive_requires_name": c.Str("destructive_requires_name", s.IgnoreAllErrors),
			}, c.DictOptional),
		}, c.DictOptional),
	}
)

func eventsMapping(r mb.ReporterV2, info *utils.ClusterInfo, settings *map[string]interface{}) error {
	metricSetFields, err := schema.Apply(*settings)

	if err != nil {
		err = fmt.Errorf("failed applying cluster settings schema %w", err)
		events.SendErrorEventWithRandomTransactionId(err, info, r, ClusterSettingsMetricSet, ClusterSettingsPath)
		return err
	}

	r.Event(events.CreateEventWithRandomTransactionId(info, metricSetFields))

	return nil
}
