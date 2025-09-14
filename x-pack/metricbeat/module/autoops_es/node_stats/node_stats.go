// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package node_stats

import (
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
)

const (
	ClusterStateMasterNodePath = "/_cluster/state/master_node?filter_path=master_node"
	NodesStatsMetricSet        = "node_stats"
	NodesStatsPath             = "/_nodes/stats/breaker,fs,http,indices,jvm,os,process,thread_pool,transport?filter_path=**.host,**.http.current_open,**.http.total_opened,**.transport.*_size_in_bytes,**.transport.*_count,**.fs,**.jvm,**.process,**.os,**.breaker,**.breakers,**.thread_pool.generic,**.thread_pool.get,**.thread_pool.management,**.thread_pool.search,**.thread_pool.watcher,**.thread_pool.write,**.indices,**.name,**.ip,**.roles,**.transport_address"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	metricset.AddNestedAutoOpsMetricSet(NodesStatsMetricSet, NodesStatsPath, eventsMapping)
}
