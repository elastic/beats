// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package task_stats

import (
	"github.com/docker/docker/api/types"

	"github.com/menderesk/beats/v7/metricbeat/module/docker"
	"github.com/menderesk/beats/v7/metricbeat/module/docker/cpu"
)

func getCPUStats(taskStats types.StatsJSON) cpu.CPUStats {
	usage := cpu.CPUUsage{Stat: &docker.Stat{Stats: taskStats}}

	return cpu.CPUStats{
		TotalUsage:                            usage.Total(),
		TotalUsageNormalized:                  usage.TotalNormalized(),
		UsageInKernelmode:                     taskStats.Stats.CPUStats.CPUUsage.UsageInKernelmode,
		UsageInKernelmodePercentage:           usage.InKernelMode(),
		UsageInKernelmodePercentageNormalized: usage.InKernelModeNormalized(),
		UsageInUsermode:                       taskStats.Stats.CPUStats.CPUUsage.UsageInUsermode,
		UsageInUsermodePercentage:             usage.InUserMode(),
		UsageInUsermodePercentageNormalized:   usage.InUserModeNormalized(),
		SystemUsage:                           taskStats.Stats.CPUStats.SystemUsage,
		SystemUsagePercentage:                 usage.System(),
		SystemUsagePercentageNormalized:       usage.SystemNormalized(),
	}
}
