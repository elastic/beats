// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package task_stats

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/arn"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	clusterLabel = "com_amazonaws_ecs_cluster"
	taskLabel    = "com_amazonaws_ecs_task-definition-family"
)

func eventsMapping(r mb.ReporterV2, statsList []Stats) {
	for _, stats := range statsList {
		r.Event(createEvent(&stats))
	}
}

func createEvent(stats *Stats) mb.Event {
	e := mb.Event{
		Timestamp: time.Time(stats.Time),
		MetricSetFields: mapstr.M{
			"cpu":     createCPUFields(stats),
			"memory":  createMemoryFields(stats),
			"network": createNetworkFields(stats),
			"diskio":  createDiskIOFields(stats),
		},
	}

	regionName, clusterName := getRegionAndClusterName(stats.Container.Labels)
	e.RootFields = createRootFields(stats, regionName)
	if clusterName != "" {
		e.MetricSetFields.Put("cluster_name", clusterName)
	}

	taskName := stats.Container.Labels[taskLabel]
	if taskName != "" {
		e.MetricSetFields.Put("task_name", taskName)
	}

	e.MetricSetFields.Put("identifier", generateIdentifier(stats.Container.Name, stats.Container.DockerId))
	return e
}

func generateIdentifier(containerName string, containerID string) string {
	return containerName + "/" + containerID
}

func getRegionAndClusterName(labels map[string]string) (regionName string, clusterName string) {
	if v, ok := labels[clusterLabel]; ok {
		vSplit := strings.Split(v, "cluster/")
		if len(vSplit) == 2 {
			clusterName = vSplit[1]
		}

		arnParsed, err := arn.Parse(v)
		if err == nil {
			regionName = arnParsed.Region
		}
		return
	}
	return
}

func createRootFields(stats *Stats, regionName string) mapstr.M {
	rootFields := mapstr.M{
		"container": mapstr.M{
			"id": stats.Container.DockerId,
			"image": mapstr.M{
				"name": stats.Container.Image,
			},
			"name":   stats.Container.Name,
			"labels": stats.Container.Labels,
		},
	}

	// add cloud.region
	if regionName != "" {
		cloud := mapstr.M{
			"region": regionName,
		}
		rootFields.Put("cloud", cloud)
	}
	return rootFields
}

func createCPUFields(stats *Stats) mapstr.M {
	return mapstr.M{
		"core": stats.cpuStats.PerCPUUsage,
		"total": mapstr.M{
			"pct": stats.cpuStats.TotalUsage,
			"norm": mapstr.M{
				"pct": stats.cpuStats.TotalUsageNormalized,
			},
		},
		"kernel": mapstr.M{
			"ticks": stats.cpuStats.UsageInKernelmode,
			"pct":   stats.cpuStats.UsageInKernelmodePercentage,
			"norm": mapstr.M{
				"pct": stats.cpuStats.UsageInKernelmodePercentageNormalized,
			},
		},
		"user": mapstr.M{
			"ticks": stats.cpuStats.UsageInUsermode,
			"pct":   stats.cpuStats.UsageInUsermodePercentage,
			"norm": mapstr.M{
				"pct": stats.cpuStats.UsageInUsermodePercentageNormalized,
			},
		},
		"system": mapstr.M{
			"ticks": stats.cpuStats.SystemUsage,
			"pct":   stats.cpuStats.SystemUsagePercentage,
			"norm": mapstr.M{
				"pct": stats.cpuStats.SystemUsagePercentageNormalized,
			},
		},
	}
}

func createMemoryFields(stats *Stats) mapstr.M {
	var memoryFields mapstr.M
	if stats.memoryStats.Commit+stats.memoryStats.CommitPeak+stats.memoryStats.PrivateWorkingSet > 0 {
		memoryFields = mapstr.M{
			"commit": mapstr.M{
				"total": stats.memoryStats.Commit,
				"peak":  stats.memoryStats.CommitPeak,
			},
			"private_working_set": mapstr.M{
				"total": stats.memoryStats.PrivateWorkingSet,
			},
		}
	} else {
		memoryFields = mapstr.M{
			"stats": stats.memoryStats.Stats,
			"fail": mapstr.M{
				"count": stats.memoryStats.Failcnt,
			},
			"limit": stats.memoryStats.Limit,
			"rss": mapstr.M{
				"total": stats.memoryStats.TotalRss,
				"pct":   stats.memoryStats.TotalRssP,
			},
			"usage": mapstr.M{
				"total": stats.memoryStats.Usage,
				"pct":   stats.memoryStats.UsageP,
				"max":   stats.memoryStats.MaxUsage,
			},
		}
	}

	return memoryFields
}

func createNetworkFields(stats *Stats) mapstr.M {
	networkFields := mapstr.M{}
	for _, n := range stats.networkStats {
		networkFields.Put(n.NameInterface,
			mapstr.M{"inbound": mapstr.M{
				"bytes":   n.Total.RxBytes,
				"dropped": n.Total.RxDropped,
				"errors":  n.Total.RxErrors,
				"packets": n.Total.RxPackets,
			},
				"outbound": mapstr.M{
					"bytes":   n.Total.TxBytes,
					"dropped": n.Total.TxDropped,
					"errors":  n.Total.TxErrors,
					"packets": n.Total.TxPackets,
				}})
	}
	return networkFields
}

func createDiskIOFields(stats *Stats) mapstr.M {
	return mapstr.M{
		"reads":  stats.blkioStats.reads,
		"writes": stats.blkioStats.writes,
		"total":  stats.blkioStats.totals,
		"read": mapstr.M{
			"ops":          stats.blkioStats.serviced.reads,
			"bytes":        stats.blkioStats.servicedBytes.reads,
			"rate":         stats.blkioStats.reads,
			"service_time": stats.blkioStats.servicedTime.reads,
			"wait_time":    stats.blkioStats.waitTime.reads,
			"queued":       stats.blkioStats.queued.reads,
		},
		"write": mapstr.M{
			"ops":          stats.blkioStats.serviced.writes,
			"bytes":        stats.blkioStats.servicedBytes.writes,
			"rate":         stats.blkioStats.writes,
			"service_time": stats.blkioStats.servicedTime.writes,
			"wait_time":    stats.blkioStats.waitTime.writes,
			"queued":       stats.blkioStats.queued.writes,
		},
		"summary": mapstr.M{
			"ops":          stats.blkioStats.serviced.totals,
			"bytes":        stats.blkioStats.servicedBytes.totals,
			"rate":         stats.blkioStats.totals,
			"service_time": stats.blkioStats.servicedTime.totals,
			"wait_time":    stats.blkioStats.waitTime.totals,
			"queued":       stats.blkioStats.queued.totals,
		},
	}

}
