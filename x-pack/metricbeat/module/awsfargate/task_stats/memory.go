// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package task_stats

import "github.com/docker/docker/api/types"

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

func getMemoryStats(taskStats types.StatsJSON) memoryStats {
	totalRSS := taskStats.Stats.MemoryStats.Stats["total_rss"]

	return memoryStats{
		TotalRss:  totalRSS,
		MaxUsage:  taskStats.Stats.MemoryStats.MaxUsage,
		TotalRssP: float64(totalRSS) / float64(taskStats.Stats.MemoryStats.Limit),
		Usage:     taskStats.Stats.MemoryStats.Usage,
		Stats:     taskStats.Stats.MemoryStats.Stats,
		//Windows memory statistics
		Commit:            taskStats.Stats.MemoryStats.Commit,
		CommitPeak:        taskStats.Stats.MemoryStats.CommitPeak,
		PrivateWorkingSet: taskStats.Stats.MemoryStats.PrivateWorkingSet,
	}
}
