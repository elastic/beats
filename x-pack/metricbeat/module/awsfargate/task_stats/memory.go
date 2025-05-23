// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package task_stats

import (
	dcontainer "github.com/docker/docker/api/types/container"
)

type memoryStats struct {
	Failcnt   uint64
	Limit     uint64
	MaxUsage  uint64
	TotalRss  uint64
	TotalRssP float64
	Usage     uint64
	//Raw stats from the cgroup subsystem
	Stats map[string]uint64
	//Windows-only memory stats
	Commit            uint64
	CommitPeak        uint64
	PrivateWorkingSet uint64
}

func getMemoryStats(taskStats dcontainer.StatsResponse) memoryStats {
	totalRSS := taskStats.MemoryStats.Stats["total_rss"]

	return memoryStats{
		TotalRss:  totalRSS,
		MaxUsage:  taskStats.MemoryStats.MaxUsage,
		TotalRssP: float64(totalRSS) / float64(taskStats.MemoryStats.Limit),
		Usage:     taskStats.MemoryStats.Usage,
		Stats:     taskStats.MemoryStats.Stats,
		//Windows memory statistics
		Commit:            taskStats.MemoryStats.Commit,
		CommitPeak:        taskStats.MemoryStats.CommitPeak,
		PrivateWorkingSet: taskStats.MemoryStats.PrivateWorkingSet,
	}
}
