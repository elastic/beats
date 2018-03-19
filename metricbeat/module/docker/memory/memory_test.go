package memory

import (
	"reflect"
	"testing"
	"time"

	"github.com/docker/docker/api/types"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"

	"github.com/stretchr/testify/assert"
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
	expectedEvent := common.MapStr{
		"_module": common.MapStr{
			"container": common.MapStr{
				"id":   containerID,
				"name": "name1",
				"labels": common.MapStr{
					"label1": "val1",
					"label2": common.MapStr{
						"foo":   "val3",
						"value": "val2",
					},
				},
			},
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
	event := eventMapping(&rawStats)
	//THEN
	assert.True(t, equalEvent(expectedEvent, event))
	t.Logf(" expected : %v", expectedEvent)
	t.Logf(" returned : %v", event)
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
