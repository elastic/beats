// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package task_stats

import "github.com/docker/docker/api/types"

// BlkioRaw sums raw Blkio stats
type BlkioRaw struct {
	reads  uint64
	writes uint64
	totals uint64
}

type blkioStats struct {
	reads  float64
	writes float64
	totals float64

	serviced      BlkioRaw
	servicedBytes BlkioRaw
	servicedTime  BlkioRaw
	waitTime      BlkioRaw
	queued        BlkioRaw
}

// getBlkioStats collects diskio metrics from BlkioStats structures(not populated in Windows)
func getBlkioStats(raw types.BlkioStats) blkioStats {
	return blkioStats{
		serviced:      getNewStats(raw.IoServicedRecursive),
		servicedBytes: getNewStats(raw.IoServiceBytesRecursive),
		servicedTime:  getNewStats(raw.IoServiceTimeRecursive),
		waitTime:      getNewStats(raw.IoWaitTimeRecursive),
		queued:        getNewStats(raw.IoQueuedRecursive),
	}
}

func getNewStats(blkioEntry []types.BlkioStatEntry) BlkioRaw {
	stats := BlkioRaw{
		reads:  0,
		writes: 0,
		totals: 0,
	}

	for _, myEntry := range blkioEntry {
		switch myEntry.Op {
		case "Write":
			stats.writes += myEntry.Value
		case "Read":
			stats.reads += myEntry.Value
		case "Total":
			stats.totals += myEntry.Value
		}
	}
	return stats
}
