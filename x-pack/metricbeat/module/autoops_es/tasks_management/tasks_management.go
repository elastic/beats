// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tasks_management

import "github.com/elastic/beats/v9/x-pack/metricbeat/module/autoops_es/metricset"

const (
	TasksMetricSet = "tasks_management"
	TasksPath      = "/_tasks?detailed&group_by=parents&actions=indices:data/write*,indices:data/read*,indices:admin/snapshot*,internal:index/shard*&filter_path=**.node,*.*.id,*.*.type,*.*.action,*.*.description,*.*.*time*,*.*.cancellable,*.*.headers"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	metricset.AddAutoOpsMetricSet(TasksMetricSet, TasksPath, eventsMapping)
}
