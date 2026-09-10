// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package cluster_settings

import (
	"strings"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"

	"github.com/stretchr/testify/require"
)

// Tests that Cluster Info is consistently reported, the Cluster Name, and the dynamic status from the filename
func expectValidParsedData(t *testing.T, data metricset.FetcherData[map[string]any]) {
	require.NoError(t, data.Error)
	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	require.Equal(t, 1, len(data.Reporter.GetEvents()))

	event := data.Reporter.GetEvents()[0]

	auto_ops_testing.CheckEventWithoutTransactionId(t, event, data.ClusterInfo)

	// metrics exist
	require.True(t, len(*event.MetricSetFields.FlattenKeys()) > 3)

	// filename includes the "display_name" as the second part
	displayName := strings.Split(data.File, ".")[1]

	// Settings are flattened with precedence: transient > persistent > defaults
	require.Equal(t, "4000", auto_ops_testing.GetObjectValue(event.MetricSetFields, "cluster.max_shards_per_node"))
	require.Equal(t, displayName, auto_ops_testing.GetObjectValue(event.MetricSetFields, "cluster.metadata.display_name"))
	require.ElementsMatch(t, []string{"/app/data"}, auto_ops_testing.GetObjectValue(event.MetricSetFields, "path.data"))
	require.Equal(t, "3", auto_ops_testing.GetObjectValue(event.MetricSetFields, "serverless.search.search_power_min"))

	// the frozen flood stage watermark is reported alongside the regular flood stage watermark
	require.Equal(t, "95%", auto_ops_testing.GetObjectValue(event.MetricSetFields, "cluster.routing.allocation.disk.watermark.flood_stage"))
	require.Equal(t, "95%", auto_ops_testing.GetObjectValue(event.MetricSetFields, "cluster.routing.allocation.disk.watermark.flood_stage_frozen"))

	// allocation and rebalance settings present in every supported version
	require.Equal(t, "all", auto_ops_testing.GetObjectValue(event.MetricSetFields, "cluster.routing.allocation.enable"))
	require.Equal(t, "all", auto_ops_testing.GetObjectValue(event.MetricSetFields, "cluster.routing.rebalance.enable"))
	require.Equal(t, "4", auto_ops_testing.GetObjectValue(event.MetricSetFields, "cluster.routing.allocation.node_initial_primaries_recoveries"))
	require.Equal(t, "true", auto_ops_testing.GetObjectValue(event.MetricSetFields, "cluster.routing.allocation.disk.threshold_enabled"))

	// awareness attributes are a list setting, so they survive as an array rather than a string
	require.ElementsMatch(t, []string{"region", "logical_availability_zone"}, auto_ops_testing.GetObjectValue(event.MetricSetFields, "cluster.routing.allocation.awareness.attributes"))

	// gateway.* is static node-scoped config, so it is only ever reported from defaults
	require.Equal(t, "0", auto_ops_testing.GetObjectValue(event.MetricSetFields, "gateway.expected_data_nodes"))
	require.Equal(t, "-1", auto_ops_testing.GetObjectValue(event.MetricSetFields, "gateway.recover_after_data_nodes"))
	require.Equal(t, "5m", auto_ops_testing.GetObjectValue(event.MetricSetFields, "gateway.recover_after_time"))

	// version-specific settings
	if data.Version == "7.17.0" {
		// the pre-8.0 gateway settings, removed in 8.0 in favour of the *_data_nodes variants
		require.Equal(t, "0", auto_ops_testing.GetObjectValue(event.MetricSetFields, "gateway.expected_nodes"))
		require.Equal(t, "-1", auto_ops_testing.GetObjectValue(event.MetricSetFields, "gateway.recover_after_nodes"))

		// transient overrides persistent for auto_create_index
		require.Equal(t, "false", auto_ops_testing.GetObjectValue(event.MetricSetFields, "action.auto_create_index"))

		// the shard filter groups accept _tier alongside _ip/_host/_name
		require.Equal(t, "", auto_ops_testing.GetObjectValue(event.MetricSetFields, "cluster.routing.allocation.exclude._tier"))
		require.Equal(t, "", auto_ops_testing.GetObjectValue(event.MetricSetFields, "cluster.routing.allocation.include._tier"))
		require.Equal(t, "", auto_ops_testing.GetObjectValue(event.MetricSetFields, "cluster.routing.allocation.require._tier"))

		// discovery.zen.* is a 7.x-only namespace
		require.Equal(t, "-1", auto_ops_testing.GetObjectValue(event.MetricSetFields, "discovery.zen.minimum_master_nodes"))
		require.Equal(t, "3s", auto_ops_testing.GetObjectValue(event.MetricSetFields, "discovery.zen.ping_timeout"))
		require.Equal(t, "60000ms", auto_ops_testing.GetObjectValue(event.MetricSetFields, "discovery.zen.join_timeout"))
		require.Equal(t, "30s", auto_ops_testing.GetObjectValue(event.MetricSetFields, "discovery.zen.publish_timeout"))
		require.Equal(t, "30s", auto_ops_testing.GetObjectValue(event.MetricSetFields, "discovery.zen.fd.ping_timeout"))
		require.Equal(t, "30000ms", auto_ops_testing.GetObjectValue(event.MetricSetFields, "discovery.zen.master_election.wait_for_joins_timeout"))
		// list settings are reported as empty arrays rather than dropped
		require.Equal(t, []any{}, auto_ops_testing.GetObjectValue(event.MetricSetFields, "discovery.zen.hosts_provider"))

		// `hosts` is a list and `hosts.resolve_timeout` a literal dotted sibling key: both are reported
		require.Equal(t, []any{}, auto_ops_testing.GetObjectValue(event.MetricSetFields, "discovery.zen.ping.unicast.hosts"))
		require.Equal(t, "5s", auto_ops_testing.GetObjectValue(event.MetricSetFields, "discovery.zen.ping.unicast.hosts_resolve_timeout"))

		// this cluster has no archived settings, so no empty key is added to the event
		require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "archived"))
	} else if data.Version == "8.15.3" {
		require.Equal(t, ".ent-search-*-logs-*,-.ent-search-*,+*", auto_ops_testing.GetObjectValue(event.MetricSetFields, "action.auto_create_index"))

		// discovery.zen.* was removed in 8.0, so nothing is reported for it
		require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "discovery.zen"))

		// likewise the pre-8.0 gateway settings
		require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "gateway.expected_nodes"))
		require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "gateway.recover_after_nodes"))

		// archived setting names are reported with the `archived.` prefix stripped, unioned across
		// persistent and transient rather than one scope overriding the other, and sorted. The
		// fixture nests most keys but leaves `disk.include_relocations` as a literal dotted key,
		// because Elasticsearch emits both shapes.
		require.Equal(t, []string{
			"cluster.routing.allocation.disk.include_relocations",
			"search.remote.connect",
			"xpack.monitoring.exporters.cloud_monitoring.type",
		}, auto_ops_testing.GetObjectValue(event.MetricSetFields, "archived"))
	}

	// schema is expected to drop this field if it appears (it does in one file)
	require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "ignored_field"))
}

// Expect a valid response from Elasticsearch to create a single event
func TestProperlyHandlesResponse(t *testing.T) {
	metricset.RunTestsForServerlessMetricSetWithGlobFiles(t, "./_meta/test/cluster_settings.*.json", ClusterSettingsMetricSet, eventsMapping, expectValidParsedData)
}

func TestCollectArchivedSettings(t *testing.T) {
	tests := map[string]struct {
		settings map[string]any
		expected []string
	}{
		"no archived settings": {
			settings: map[string]any{
				"persistent": map[string]any{"cluster": map[string]any{"max_shards_per_node": "4000"}},
				"transient":  map[string]any{},
			},
			expected: nil,
		},
		"an empty archived object contributes no names": {
			settings: map[string]any{
				"persistent": map[string]any{"archived": map[string]any{}},
			},
			expected: nil,
		},
		"nested and literal dotted keys flatten to the same shape": {
			settings: map[string]any{
				"persistent": map[string]any{
					"archived": map[string]any{
						"search": map[string]any{"remote": map[string]any{"connect": "false"}},
						"cluster": map[string]any{
							"routing": map[string]any{"allocation.disk.include_relocations": "false"},
						},
					},
				},
			},
			expected: []string{
				"cluster.routing.allocation.disk.include_relocations",
				"search.remote.connect",
			},
		},
		"persistent and transient are unioned, not overridden": {
			settings: map[string]any{
				"persistent": map[string]any{
					"archived": map[string]any{"xpack": map[string]any{"monitoring": map[string]any{"enabled": "true"}}},
				},
				"transient": map[string]any{
					"archived": map[string]any{"search": map[string]any{"remote": map[string]any{"connect": "false"}}},
				},
			},
			expected: []string{"search.remote.connect", "xpack.monitoring.enabled"},
		},
		"a name archived in both scopes is reported once": {
			settings: map[string]any{
				"persistent": map[string]any{
					"archived": map[string]any{"search": map[string]any{"remote": map[string]any{"connect": "false"}}},
				},
				"transient": map[string]any{
					"archived": map[string]any{"search": map[string]any{"remote": map[string]any{"connect": "true"}}},
				},
			},
			expected: []string{"search.remote.connect"},
		},
		"list values are reported by name only": {
			settings: map[string]any{
				"persistent": map[string]any{
					"archived": map[string]any{"discovery": map[string]any{"zen": map[string]any{"hosts_provider": []any{"file"}}}},
				},
			},
			expected: []string{"discovery.zen.hosts_provider"},
		},
		"defaults are ignored: archived settings only exist in cluster state": {
			settings: map[string]any{
				"defaults": map[string]any{
					"archived": map[string]any{"search": map[string]any{"remote": map[string]any{"connect": "false"}}},
				},
			},
			expected: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.expected, collectArchivedSettings(test.settings))
		})
	}
}

// Go randomises map iteration order, so the same response must still produce an identical list.
func TestCollectArchivedSettingsIsDeterministic(t *testing.T) {
	settings := map[string]any{
		"persistent": map[string]any{
			"archived": map[string]any{
				"xpack":   map[string]any{"monitoring": map[string]any{"enabled": "true"}},
				"search":  map[string]any{"remote": map[string]any{"connect": "false"}},
				"cluster": map[string]any{"routing": map[string]any{"allocation.disk.include_relocations": "false"}},
			},
		},
		"transient": map[string]any{
			"archived": map[string]any{"node": map[string]any{"max_local_storage_nodes": "1"}},
		},
	}

	first := collectArchivedSettings(settings)

	for range 50 {
		require.Equal(t, first, collectArchivedSettings(settings))
	}
}
