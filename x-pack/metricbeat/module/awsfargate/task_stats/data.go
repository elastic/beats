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
		eventMapping(r, &stats)
	}
}

func eventMapping(r mb.ReporterV2, stats *Stats) {
	containerFields := common.MapStr{
		"container": common.MapStr{
			"id": stats.Container.DockerId,
			"image": common.MapStr{
				"name": stats.Container.Image,
			},
			"name":   stats.Container.Name,
			"labels": stats.Container.Labels,
		},
	}

	cpuFields := common.MapStr{
		"core": stats.PerCPUUsage,
		"total": common.MapStr{
			"pct": stats.TotalUsage,
			"norm": common.MapStr{
				"pct": stats.TotalUsageNormalized,
			},
		},
		"kernel": common.MapStr{
			"ticks": stats.UsageInKernelmode,
			"pct":   stats.UsageInKernelmodePercentage,
			"norm": common.MapStr{
				"pct": stats.UsageInKernelmodePercentageNormalized,
			},
		},
		"user": common.MapStr{
			"ticks": stats.UsageInUsermode,
			"pct":   stats.UsageInUsermodePercentage,
			"norm": common.MapStr{
				"pct": stats.UsageInUsermodePercentageNormalized,
			},
		},
		"system": common.MapStr{
			"ticks": stats.SystemUsage,
			"pct":   stats.SystemUsagePercentage,
			"norm": common.MapStr{
				"pct": stats.SystemUsagePercentageNormalized,
			},
		},
	}

	var memoryFields common.MapStr
	if stats.Commit+stats.CommitPeak+stats.PrivateWorkingSet > 0 {
		memoryFields = common.MapStr{
			"commit": common.MapStr{
				"total": stats.Commit,
				"peak":  stats.CommitPeak,
			},
			"private_working_set": common.MapStr{
				"total": stats.PrivateWorkingSet,
			},
		}
	} else {
		memoryFields = common.MapStr{
			"stats": stats.Stats,
			"fail": common.MapStr{
				"count": stats.Failcnt,
			},
			"limit": stats.Limit,
			"rss": common.MapStr{
				"total": stats.TotalRss,
				"pct":   stats.TotalRssP,
			},
			"usage": common.MapStr{
				"total": stats.Usage,
				"pct":   stats.UsageP,
				"max":   stats.MaxUsage,
			},
		}
	}

	networkFields := common.MapStr{
		"interface": stats.NameInterface,
		// Deprecated
		"in": common.MapStr{
			"bytes":   stats.RxBytes,
			"dropped": stats.RxDropped,
			"errors":  stats.RxErrors,
			"packets": stats.RxPackets,
		},
		// Deprecated
		"out": common.MapStr{
			"bytes":   stats.TxBytes,
			"dropped": stats.TxDropped,
			"errors":  stats.TxErrors,
			"packets": stats.TxPackets,
		},
		"inbound": common.MapStr{
			"bytes":   stats.Total.RxBytes,
			"dropped": stats.Total.RxDropped,
			"errors":  stats.Total.RxErrors,
			"packets": stats.Total.RxPackets,
		},
		"outbound": common.MapStr{
			"bytes":   stats.Total.TxBytes,
			"dropped": stats.Total.TxDropped,
			"errors":  stats.Total.TxErrors,
			"packets": stats.Total.TxPackets,
		},
	}

	diskFields := common.MapStr{
		"reads":  stats.reads,
		"writes": stats.writes,
		"total":  stats.totals,
		"read": common.MapStr{
			"ops":          stats.serviced.reads,
			"bytes":        stats.servicedBytes.reads,
			"rate":         stats.reads,
			"service_time": stats.servicedTime.reads,
			"wait_time":    stats.waitTime.reads,
			"queued":       stats.queued.reads,
		},
		"write": common.MapStr{
			"ops":          stats.serviced.writes,
			"bytes":        stats.servicedBytes.writes,
			"rate":         stats.writes,
			"service_time": stats.servicedTime.writes,
			"wait_time":    stats.waitTime.writes,
			"queued":       stats.queued.writes,
		},
		"summary": common.MapStr{
			"ops":          stats.serviced.totals,
			"bytes":        stats.servicedBytes.totals,
			"rate":         stats.totals,
			"service_time": stats.servicedTime.totals,
			"wait_time":    stats.waitTime.totals,
			"queued":       stats.queued.totals,
		},
	}

	r.Event(mb.Event{
		Timestamp:  time.Time(stats.Time),
		RootFields: containerFields,
		MetricSetFields: common.MapStr{
			"cpu":     cpuFields,
			"memory":  memoryFields,
			"network": networkFields,
			"diskio":  diskFields,
		},
	})
}
