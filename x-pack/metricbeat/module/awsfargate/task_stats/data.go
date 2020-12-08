// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package task_stats

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

func eventsMapping(r mb.ReporterV2, statsList []Stats) {
	for _, stats := range statsList {
		r.Event(createEvent(&stats))
	}
}

func createEvent(stats *Stats) mb.Event {
	return mb.Event{
		Timestamp:  time.Time(stats.Time),
		RootFields: createContainerFields(stats),
		MetricSetFields: common.MapStr{
			"cpu":     createCPUFields(stats),
			"memory":  createMemoryFields(stats),
			"network": createNetworkFields(stats),
			"diskio":  createDiskIOFields(stats),
		},
	}
}

func createContainerFields(stats *Stats) common.MapStr {
	return common.MapStr{
		"container": common.MapStr{
			"id": stats.Container.DockerId,
			"image": common.MapStr{
				"name": stats.Container.Image,
			},
			"name":   stats.Container.Name,
			"labels": stats.Container.Labels,
		},
	}
}

func createCPUFields(stats *Stats) common.MapStr {
	return common.MapStr{
		"core": stats.cpuStats.PerCPUUsage,
		"total": common.MapStr{
			"pct": stats.cpuStats.TotalUsage,
			"norm": common.MapStr{
				"pct": stats.cpuStats.TotalUsageNormalized,
			},
		},
		"kernel": common.MapStr{
			"ticks": stats.cpuStats.UsageInKernelmode,
			"pct":   stats.cpuStats.UsageInKernelmodePercentage,
			"norm": common.MapStr{
				"pct": stats.cpuStats.UsageInKernelmodePercentageNormalized,
			},
		},
		"user": common.MapStr{
			"ticks": stats.cpuStats.UsageInUsermode,
			"pct":   stats.cpuStats.UsageInUsermodePercentage,
			"norm": common.MapStr{
				"pct": stats.cpuStats.UsageInUsermodePercentageNormalized,
			},
		},
		"system": common.MapStr{
			"ticks": stats.cpuStats.SystemUsage,
			"pct":   stats.cpuStats.SystemUsagePercentage,
			"norm": common.MapStr{
				"pct": stats.cpuStats.SystemUsagePercentageNormalized,
			},
		},
	}
}

func createMemoryFields(stats *Stats) common.MapStr {
	var memoryFields common.MapStr
	if stats.memoryStats.Commit+stats.memoryStats.CommitPeak+stats.memoryStats.PrivateWorkingSet > 0 {
		memoryFields = common.MapStr{
			"commit": common.MapStr{
				"total": stats.memoryStats.Commit,
				"peak":  stats.memoryStats.CommitPeak,
			},
			"private_working_set": common.MapStr{
				"total": stats.memoryStats.PrivateWorkingSet,
			},
		}
	} else {
		memoryFields = common.MapStr{
			"stats": stats.memoryStats.Stats,
			"fail": common.MapStr{
				"count": stats.memoryStats.Failcnt,
			},
			"limit": stats.memoryStats.Limit,
			"rss": common.MapStr{
				"total": stats.memoryStats.TotalRss,
				"pct":   stats.memoryStats.TotalRssP,
			},
			"usage": common.MapStr{
				"total": stats.memoryStats.Usage,
				"pct":   stats.memoryStats.UsageP,
				"max":   stats.memoryStats.MaxUsage,
			},
		}
	}

	return memoryFields
}

func createNetworkFields(stats *Stats) common.MapStr {
	networkFields := common.MapStr{}
	for _, n := range stats.networkStats {
		networkFields.Put(n.NameInterface,
			common.MapStr{"inbound": common.MapStr{
				"bytes":   n.Total.RxBytes,
				"dropped": n.Total.RxDropped,
				"errors":  n.Total.RxErrors,
				"packets": n.Total.RxPackets,
			},
				"outbound": common.MapStr{
					"bytes":   n.Total.TxBytes,
					"dropped": n.Total.TxDropped,
					"errors":  n.Total.TxErrors,
					"packets": n.Total.TxPackets,
				}})
	}
	return networkFields
}

func createDiskIOFields(stats *Stats) common.MapStr {
	return common.MapStr{
		"reads":  stats.blkioStats.reads,
		"writes": stats.blkioStats.writes,
		"total":  stats.blkioStats.totals,
		"read": common.MapStr{
			"ops":          stats.blkioStats.serviced.reads,
			"bytes":        stats.blkioStats.servicedBytes.reads,
			"rate":         stats.blkioStats.reads,
			"service_time": stats.blkioStats.servicedTime.reads,
			"wait_time":    stats.blkioStats.waitTime.reads,
			"queued":       stats.blkioStats.queued.reads,
		},
		"write": common.MapStr{
			"ops":          stats.blkioStats.serviced.writes,
			"bytes":        stats.blkioStats.servicedBytes.writes,
			"rate":         stats.blkioStats.writes,
			"service_time": stats.blkioStats.servicedTime.writes,
			"wait_time":    stats.blkioStats.waitTime.writes,
			"queued":       stats.blkioStats.queued.writes,
		},
		"summary": common.MapStr{
			"ops":          stats.blkioStats.serviced.totals,
			"bytes":        stats.blkioStats.servicedBytes.totals,
			"rate":         stats.blkioStats.totals,
			"service_time": stats.blkioStats.servicedTime.totals,
			"wait_time":    stats.blkioStats.waitTime.totals,
			"queued":       stats.blkioStats.queued.totals,
		},
	}

}
