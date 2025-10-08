// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package memory

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/docker"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var defaultContainerID = "containerID"

var defaultLabels = map[string]string{
	"label1":     "val1",
	"label2":     "val2",
	"label2.foo": "val3",
}

var defaultContainerStats = container.Summary{
	ID:         defaultContainerID,
	Image:      "image",
	Command:    "command",
	Created:    123789,
	Status:     "Up",
	SizeRw:     123,
	SizeRootFs: 456,
	Names:      []string{"/name1", "name1/fake"},
	Labels:     defaultLabels,
}

func TestMemStatsV2(t *testing.T) {
	// Test to make sure we don't report any RSS metrics where they don't exist
	memoryService := &MemoryService{}
	memorystats := getMemoryStats(time.Now(), 1, false)

	memoryRawStats := docker.Stat{}
	memoryRawStats.Container = &defaultContainerStats
	memoryRawStats.Stats = memorystats

	rawStats := memoryService.getMemoryStats(memoryRawStats, false)
	require.False(t, rawStats.TotalRss.Exists())
	require.False(t, rawStats.TotalRssP.Exists())

	r := &mbtest.CapturingReporterV2{}
	eventMapping(r, &rawStats)
	events := r.GetEvents()
	require.NotContains(t, "rss", events[0].MetricSetFields)

}

func TestMemoryService_GetMemoryStats(t *testing.T) {

	memoryService := &MemoryService{}
	memorystats := getMemoryStats(time.Now(), 1, true)

	memoryRawStats := docker.Stat{}
	memoryRawStats.Container = &defaultContainerStats
	memoryRawStats.Stats = memorystats

	totalRSS := memorystats.MemoryStats.Stats["total_rss"]
	expectedRootFields := mapstr.M{
		"container": mapstr.M{
			"id":   defaultContainerID,
			"name": "name1",
			"image": mapstr.M{
				"name": "image",
			},
			"runtime": "docker",
			"memory": mapstr.M{
				"usage": 0.5,
			},
		},
		"docker": mapstr.M{
			"container": mapstr.M{
				"labels": mapstr.M{
					"label1": "val1",
					"label2": mapstr.M{
						"foo":   "val3",
						"value": "val2",
					},
				},
			},
		},
	}
	expectedFields := mapstr.M{
		"stats": map[string]uint64{
			"total_rss": 5,
		},
		"fail": mapstr.M{
			"count": memorystats.MemoryStats.Failcnt,
		},
		"limit": memorystats.MemoryStats.Limit,
		"rss": mapstr.M{
			"total": totalRSS,
			"pct":   float64(totalRSS) / float64(memorystats.MemoryStats.Limit),
		},
		"usage": mapstr.M{
			"total": memorystats.MemoryStats.Usage,
			"pct":   float64(memorystats.MemoryStats.Usage) / float64(memorystats.MemoryStats.Limit),
			"max":   memorystats.MemoryStats.MaxUsage,
		},
	}
	//WHEN
	rawStats := memoryService.getMemoryStats(memoryRawStats, false)
	r := &mbtest.CapturingReporterV2{}
	eventMapping(r, &rawStats)
	events := r.GetEvents()
	//THEN
	assert.Empty(t, r.GetErrors())
	assert.NotEmpty(t, events)
	event := events[0]
	assert.Equal(t, expectedRootFields, event.RootFields)
	assert.Equal(t, expectedFields, event.MetricSetFields)
}

func TestMemoryServiceBadData(t *testing.T) {

	badMemStats := container.StatsResponse{
		Read:        time.Now(),
		MemoryStats: container.MemoryStats{}, //Test for cases where this is empty
	}

	memoryService := &MemoryService{}
	memoryRawStats := []docker.Stat{{Stats: badMemStats}}
	rawStats := memoryService.getMemoryStatsList(memoryRawStats, false)
	assert.Len(t, rawStats, 0)

}

func TestMemoryMath(t *testing.T) {
	memStats := container.StatsResponse{
		Read: time.Now(),
		PreCPUStats: container.CPUStats{
			CPUUsage: container.CPUUsage{
				TotalUsage: 200,
			},
		},
		MemoryStats: container.MemoryStats{
			Limit: 5,
			Usage: 5000,
			Stats: map[string]uint64{
				"total_inactive_file": 1000, // CGV1
				"inactive_file":       900,
			},
		}, //Test for cases where this is empty
	}

	memoryService := &MemoryService{}
	memoryRawStats := []docker.Stat{
		{Stats: memStats, Container: &container.Summary{Names: []string{"test-container"}, Labels: map[string]string{}}},
	}
	rawStats := memoryService.getMemoryStatsList(memoryRawStats, false)
	assert.Equal(t, float64(800), rawStats[0].UsageP) // 5000-900 /5
}

func getMemoryStats(read time.Time, number uint64, rssExists bool) container.StatsResponse {

	myMemoryStats := container.StatsResponse{
		Read: read,
		MemoryStats: container.MemoryStats{
			MaxUsage: number,
			Usage:    number * 2,
			Failcnt:  number * 3,
			Limit:    number * 4,
			Stats:    map[string]uint64{},
		},
	}

	if rssExists {
		myMemoryStats.MemoryStats.Stats["total_rss"] = number * 5
	}

	return myMemoryStats
}
