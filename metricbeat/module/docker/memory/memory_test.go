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
	"reflect"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/docker"
)

func TestMemoryService_GetMemoryStats(t *testing.T) {
	//Container  + dockerstats
	containerID := "containerID"
	labels := map[string]string{
		"label1":     "val1",
		"label2":     "val2",
		"label2.foo": "val3",
	}
	container := types.Container{
		ID:         containerID,
		Image:      "image",
		Command:    "command",
		Created:    123789,
		Status:     "Up",
		SizeRw:     123,
		SizeRootFs: 456,
		Names:      []string{"/name1", "name1/fake"},
		Labels:     labels,
	}
	memoryService := &MemoryService{}
	memorystats := getMemoryStats(time.Now(), 1)

	memoryRawStats := docker.Stat{}
	memoryRawStats.Container = &container
	memoryRawStats.Stats = memorystats

	totalRSS := memorystats.MemoryStats.Stats["total_rss"]
	expectedRootFields := common.MapStr{
		"container": common.MapStr{
			"id":   containerID,
			"name": "name1",
			"image": common.MapStr{
				"name": "image",
			},
			"runtime": "docker",
		},
		"docker": common.MapStr{
			"container": common.MapStr{
				"labels": common.MapStr{
					"label1": "val1",
					"label2": common.MapStr{
						"foo":   "val3",
						"value": "val2",
					},
				},
			},
		},
	}
	expectedFields := common.MapStr{
		"stats": map[string]uint64{
			"total_rss": 5,
		},
		"fail": common.MapStr{
			"count": memorystats.MemoryStats.Failcnt,
		},
		"limit": memorystats.MemoryStats.Limit,
		"rss": common.MapStr{
			"total": totalRSS,
			"pct":   float64(totalRSS) / float64(memorystats.MemoryStats.Limit),
		},
		"usage": common.MapStr{
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

func getMemoryStats(read time.Time, number uint64) types.StatsJSON {

	myMemoryStats := types.StatsJSON{
		Stats: types.Stats{
			Read: read,
			MemoryStats: types.MemoryStats{
				MaxUsage: number,
				Usage:    number * 2,
				Failcnt:  number * 3,
				Limit:    number * 4,
				Stats:    map[string]uint64{},
			},
		},
	}

	myMemoryStats.MemoryStats.Stats["total_rss"] = number * 5

	return myMemoryStats
}
func equalEvent(expectedEvent common.MapStr, event common.MapStr) bool {
	return reflect.DeepEqual(expectedEvent, event)
}
