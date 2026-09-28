// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cluster_settings

import (
	"fmt"
	"maps"
	"slices"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

// allocationFilterSchema builds the schema for one of the interchangeable
// `cluster.routing.allocation.{exclude,include,require}` shard filter groups,
// which all accept the same attributes.
func allocationFilterSchema(key string) c.ConvMap {
	return c.Dict(key, s.Schema{
		"_ip":   c.Str("_ip", s.IgnoreAllErrors),
		"_host": c.Str("_host", s.IgnoreAllErrors),
		"_name": c.Str("_name", s.IgnoreAllErrors),
		"_tier": c.Str("_tier", s.IgnoreAllErrors),
	}, c.DictOptional)
}

var (
	// `discovery.zen.*` is a 7.x-only namespace, removed in 8.0, so every key is optional.
	discoverySchema = c.Dict("discovery", s.Schema{
		"zen": c.Dict("zen", s.Schema{
			"minimum_master_nodes": c.Str("minimum_master_nodes", s.IgnoreAllErrors),
			"ping_timeout":         c.Str("ping_timeout", s.IgnoreAllErrors),
			"join_timeout":         c.Str("join_timeout", s.IgnoreAllErrors),
			"publish_timeout":      c.Str("publish_timeout", s.IgnoreAllErrors),
			"hosts_provider":       c.Ifc("hosts_provider", s.Optional),
			"fd": c.Dict("fd", s.Schema{
				"ping_timeout": c.Str("ping_timeout", s.IgnoreAllErrors),
			}, c.DictOptional),
			"master_election": c.Dict("master_election", s.Schema{
				"wait_for_joins_timeout": c.Str("wait_for_joins_timeout", s.IgnoreAllErrors),
			}, c.DictOptional),
			"ping": c.Dict("ping", s.Schema{
				"unicast": c.Dict("unicast", s.Schema{
					"hosts": c.Ifc("hosts", s.Optional),
					// `hosts` is a list and `hosts.resolve_timeout` a literal dotted sibling key,
					// so the two cannot share the `hosts` name in the event
					"hosts_resolve_timeout": c.Str("hosts.resolve_timeout", s.IgnoreAllErrors),
				}, c.DictOptional),
			}, c.DictOptional),
		}, c.DictOptional),
	}, c.DictOptional)

	actionSchema = c.Dict("action", s.Schema{
		"destructive_requires_name": c.Str("destructive_requires_name", s.IgnoreAllErrors),
		"auto_create_index":         c.Str("auto_create_index", s.IgnoreAllErrors),
	}, c.DictOptional)

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
					"threshold_enabled": c.Str("threshold_enabled", s.IgnoreAllErrors),
					"watermark": c.Dict("watermark", s.Schema{
						"low":                c.Str("low", s.IgnoreAllErrors),
						"high":               c.Str("high", s.IgnoreAllErrors),
						"flood_stage":        c.Str("flood_stage", s.IgnoreAllErrors),
						"flood_stage_frozen": c.Str("flood_stage.frozen", s.IgnoreAllErrors),
					}, c.DictOptional),
				}, c.DictOptional),
				"enable":                              c.Str("enable", s.IgnoreAllErrors),
				"node_concurrent_outgoing_recoveries": c.Str("node_concurrent_outgoing_recoveries", s.IgnoreAllErrors),
				"cluster_concurrent_rebalance":        c.Str("cluster_concurrent_rebalance", s.IgnoreAllErrors),
				"node_concurrent_recoveries":          c.Str("node_concurrent_recoveries", s.IgnoreAllErrors),
				"node_initial_primaries_recoveries":   c.Str("node_initial_primaries_recoveries", s.IgnoreAllErrors),
				"total_shards_per_node":               c.Str("total_shards_per_node", s.IgnoreAllErrors),
				"awareness": c.Dict("awareness", s.Schema{
					// a list setting: ES returns an array, or a comma-separated string when set that way
					"attributes": c.Ifc("attributes", s.Optional),
				}, c.DictOptional),
				"exclude": allocationFilterSchema("exclude"),
				"include": allocationFilterSchema("include"),
				"require": allocationFilterSchema("require"),
			}, c.DictOptional),
			"rebalance": c.Dict("rebalance", s.Schema{
				"enable": c.Str("enable", s.IgnoreAllErrors),
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
			"discovery":  discoverySchema,
			"processors": c.Str("processors", s.IgnoreAllErrors),
			"cluster":    clusterSchema,
			// `gateway.*` is static, node-scoped configuration: it can never be set as a persistent
			// or transient cluster setting, so it is only mapped under `defaults`. Both generations
			// are collected — `expected_nodes` and `recover_after_nodes` were removed in 8.0 in
			// favour of the `*_data_nodes` variants — so every key is optional.
			"gateway": c.Dict("gateway", s.Schema{
				"expected_nodes":           c.Str("expected_nodes", s.IgnoreAllErrors),
				"expected_data_nodes":      c.Str("expected_data_nodes", s.IgnoreAllErrors),
				"recover_after_nodes":      c.Str("recover_after_nodes", s.IgnoreAllErrors),
				"recover_after_data_nodes": c.Str("recover_after_data_nodes", s.IgnoreAllErrors),
				"recover_after_time":       c.Str("recover_after_time", s.IgnoreAllErrors),
			}, c.DictOptional),
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
			"action": actionSchema,
		}, c.DictRequired),
		"persistent": c.Dict("persistent", s.Schema{
			"serverless": c.Dict("serverless", s.Schema{
				"search": c.Dict("search", s.Schema{
					"boost_window":     c.Str("boost_window", s.IgnoreAllErrors),
					"search_power_max": c.Str("search_power_max", s.IgnoreAllErrors),
					"search_power_min": c.Str("search_power_min", s.IgnoreAllErrors),
				}, c.DictOptional),
			}, c.DictOptional),
			"discovery":  discoverySchema,
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
			"action": actionSchema,
		}, c.DictOptional),
		"transient": c.Dict("transient", s.Schema{
			"serverless": c.Dict("serverless", s.Schema{
				"search": c.Dict("search", s.Schema{
					"boost_window":     c.Str("boost_window", s.IgnoreAllErrors),
					"search_power_max": c.Str("search_power_max", s.IgnoreAllErrors),
					"search_power_min": c.Str("search_power_min", s.IgnoreAllErrors),
				}, c.DictOptional),
			}, c.DictOptional),
			"discovery":  discoverySchema,
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
			"action": actionSchema,
		}, c.DictOptional),
	}
)

// flattenSettings merges transient, persistent, and defaults settings
// with precedence: transient > persistent > defaults
func flattenSettings(metricSetFields mapstr.M) mapstr.M {
	result := mapstr.M{}

	// Apply in reverse precedence order (defaults first, transient last)
	if defaults, ok := metricSetFields["defaults"].(mapstr.M); ok {
		result.DeepUpdate(defaults)
	}
	if persistent, ok := metricSetFields["persistent"].(mapstr.M); ok {
		result.DeepUpdate(persistent)
	}
	if transient, ok := metricSetFields["transient"].(mapstr.M); ok {
		result.DeepUpdate(transient)
	}

	return result
}

// collectArchivedSettings returns the names of the settings Elasticsearch has archived, gathered
// from the raw response rather than the schema: `archived.*` keys are dynamic, and
// `libbeat/common/schema` can only describe a fixed set of keys.
//
// The `archived.` prefix is stripped, so `archived.search.remote.connect` is reported as
// `search.remote.connect`. Names from `persistent` and `transient` are unioned rather than
// overridden -- an archived setting in each scope is two independent facts, not two values of the
// same setting -- and the result is sorted so that repeated collections produce the same event.
//
// Returns nil when the cluster has no archived settings, so no empty key is added to the event.
func collectArchivedSettings(settings map[string]any) []string {
	names := map[string]struct{}{}

	for _, scope := range []string{"persistent", "transient"} {
		scoped, ok := settings[scope].(map[string]any)
		if !ok {
			continue
		}

		collectSettingNames("", scoped["archived"], names)
	}

	if len(names) == 0 {
		return nil
	}

	sorted := slices.Collect(maps.Keys(names))
	slices.Sort(sorted)

	return sorted
}

// collectSettingNames walks a settings subtree and records the dotted name of every leaf.
// Elasticsearch expands most dotted keys into nested objects, but leaves some as literal dotted
// keys, so both shapes end up producing the same flattened name.
func collectSettingNames(prefix string, value any, into map[string]struct{}) {
	switch typed := value.(type) {
	case nil:
		return
	case map[string]any:
		for key, child := range typed {
			collectSettingNames(joinSettingName(prefix, key), child, into)
		}
	default:
		// a leaf: the value is irrelevant, only the name of the archived setting is reported
		if prefix != "" {
			into[prefix] = struct{}{}
		}
	}
}

func joinSettingName(prefix string, key string) string {
	if prefix == "" {
		return key
	}

	return prefix + "." + key
}

func eventsMapping(r mb.ReporterV2, info *utils.ClusterInfo, settings *map[string]any) error {
	metricSetFields, err := schema.Apply(*settings)

	if err != nil {
		err = fmt.Errorf("failed applying cluster settings schema %w", err)
		events.LogAndSendErrorEventWithoutTransactionId(err, info, r, ClusterSettingsMetricSet, ClusterSettingsPath)
		return nil
	}

	// Flatten settings with precedence: transient > persistent > defaults
	flattenedSettings := flattenSettings(metricSetFields)

	// archived settings are collected outside the schema and are not subject to that precedence:
	// they are reported as a single, scope-independent list of names
	if archived := collectArchivedSettings(*settings); archived != nil {
		flattenedSettings["archived"] = archived
	}

	r.Event(events.CreateEventWithoutTransactionId(info, flattenedSettings))

	return nil
}
