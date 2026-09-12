// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cluster_settings

import "github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"

const (
	ClusterSettingsMetricSet = "cluster_settings"
	// `gateway.*` settings are static and node-scoped, so they are requested by exact path under
	// `defaults` only: `**.gateway` would pull in ~10 keys that the schema discards anyway.
	ClusterSettingsPath = "/_cluster/settings?include_defaults&filter_path=**.discovery,**.processors,**.cluster,**.repositories,**.bootstrap,**.search,**.indices,**.action,defaults.path.data,defaults.gateway.expected_nodes,defaults.gateway.expected_data_nodes,defaults.gateway.recover_after_nodes,defaults.gateway.recover_after_data_nodes,defaults.gateway.recover_after_time"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	metricset.AddAutoOpsMetricSet(ClusterSettingsMetricSet, ClusterSettingsPath, eventsMapping)
}
