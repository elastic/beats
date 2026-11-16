// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package task_stats

import (
	dcontainer "github.com/moby/moby/api/types/container"

	"github.com/elastic/beats/v7/metricbeat/module/docker"
	"github.com/elastic/beats/v7/metricbeat/module/docker/cpu"
)

func getCPUStats(taskStats dcontainer.StatsResponse) cpu.CPUStats {
	usage := cpu.CPUUsage{Stat: &docker.Stat{Stats: taskStats}}
	return usage.Metrics()
}
