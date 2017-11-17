package memory

import (
	"reflect"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestMemoryService_GetMemoryStats(t *testing.T) {
	//Container  + dockerstats
	containerID := "containerID"
	labels := map[string]string{
		"label1": "val1",
		"label2": "val2",
	}
	container := dc.APIContainers{
		ID:         containerID,
		Image:      "image",
		Command:    "command",
		Created:    123789,
		Status:     "Up",
		Ports:      []dc.APIPort{{PrivatePort: 1234, PublicPort: 4567, Type: "portType", IP: "123.456.879.1"}},
		SizeRw:     123,
		SizeRootFs: 456,
		Names:      []string{"/name1", "name1/fake"},
		Labels:     labels,
		Networks:   dc.NetworkList{},
	}
	memoryService := &MemoryService{}
	memorystats := getMemoryStats(time.Now(), 1)

	memoryRawStats := docker.Stat{}
	memoryRawStats.Container = container
	memoryRawStats.Stats = memorystats

	expectedEvent := common.MapStr{
		"_module": common.MapStr{
			"container": common.MapStr{
				"id":     containerID,
				"name":   "name1",
				"labels": docker.DeDotLabels(labels),
			},
		},
		"fail": common.MapStr{
			"count": memorystats.MemoryStats.Failcnt,
		},
		"limit": memorystats.MemoryStats.Limit,
		"rss": common.MapStr{
			"total": memorystats.MemoryStats.Stats.TotalRss,
			"pct":   float64(memorystats.MemoryStats.Stats.TotalRss) / float64(memorystats.MemoryStats.Limit),
		},
		"usage": common.MapStr{
			"total": memorystats.MemoryStats.Usage,
			"pct":   float64(memorystats.MemoryStats.Usage) / float64(memorystats.MemoryStats.Limit),
			"max":   memorystats.MemoryStats.MaxUsage,
		},
	}
	//WHEN
	rawStats := memoryService.GetMemoryStats(memoryRawStats)
	event := eventMapping(&rawStats)
	//THEN
	assert.True(t, equalEvent(expectedEvent, event))
	t.Logf(" expected : %v", expectedEvent)
	t.Logf(" returned : %v", event)
}

func getMemoryStats(read time.Time, number uint64) dc.Stats {
	type memoryStatsStructure struct {
		Stats struct {
			TotalPgmafault          uint64 `json:"total_pgmafault,omitempty" yaml:"total_pgmafault,omitempty" toml:"total_pgmafault,omitempty"`
			Cache                   uint64 `json:"cache,omitempty" yaml:"cache,omitempty" toml:"cache,omitempty"`
			MappedFile              uint64 `json:"mapped_file,omitempty" yaml:"mapped_file,omitempty" toml:"mapped_file,omitempty"`
			TotalInactiveFile       uint64 `json:"total_inactive_file,omitempty" yaml:"total_inactive_file,omitempty" toml:"total_inactive_file,omitempty"`
			Pgpgout                 uint64 `json:"pgpgout,omitempty" yaml:"pgpgout,omitempty" toml:"pgpgout,omitempty"`
			Rss                     uint64 `json:"rss,omitempty" yaml:"rss,omitempty" toml:"rss,omitempty"`
			TotalMappedFile         uint64 `json:"total_mapped_file,omitempty" yaml:"total_mapped_file,omitempty" toml:"total_mapped_file,omitempty"`
			Writeback               uint64 `json:"writeback,omitempty" yaml:"writeback,omitempty" toml:"writeback,omitempty"`
			Unevictable             uint64 `json:"unevictable,omitempty" yaml:"unevictable,omitempty" toml:"unevictable,omitempty"`
			Pgpgin                  uint64 `json:"pgpgin,omitempty" yaml:"pgpgin,omitempty" toml:"pgpgin,omitempty"`
			TotalUnevictable        uint64 `json:"total_unevictable,omitempty" yaml:"total_unevictable,omitempty" toml:"total_unevictable,omitempty"`
			Pgmajfault              uint64 `json:"pgmajfault,omitempty" yaml:"pgmajfault,omitempty" toml:"pgmajfault,omitempty"`
			TotalRss                uint64 `json:"total_rss,omitempty" yaml:"total_rss,omitempty" toml:"total_rss,omitempty"`
			TotalRssHuge            uint64 `json:"total_rss_huge,omitempty" yaml:"total_rss_huge,omitempty" toml:"total_rss_huge,omitempty"`
			TotalWriteback          uint64 `json:"total_writeback,omitempty" yaml:"total_writeback,omitempty" toml:"total_writeback,omitempty"`
			TotalInactiveAnon       uint64 `json:"total_inactive_anon,omitempty" yaml:"total_inactive_anon,omitempty" toml:"total_inactive_anon,omitempty"`
			RssHuge                 uint64 `json:"rss_huge,omitempty" yaml:"rss_huge,omitempty" toml:"rss_huge,omitempty"`
			HierarchicalMemoryLimit uint64 `json:"hierarchical_memory_limit,omitempty" yaml:"hierarchical_memory_limit,omitempty" toml:"hierarchical_memory_limit,omitempty"`
			TotalPgfault            uint64 `json:"total_pgfault,omitempty" yaml:"total_pgfault,omitempty" toml:"total_pgfault,omitempty"`
			TotalActiveFile         uint64 `json:"total_active_file,omitempty" yaml:"total_active_file,omitempty" toml:"total_active_file,omitempty"`
			ActiveAnon              uint64 `json:"active_anon,omitempty" yaml:"active_anon,omitempty" toml:"active_anon,omitempty"`
			TotalActiveAnon         uint64 `json:"total_active_anon,omitempty" yaml:"total_active_anon,omitempty" toml:"total_active_anon,omitempty"`
			TotalPgpgout            uint64 `json:"total_pgpgout,omitempty" yaml:"total_pgpgout,omitempty" toml:"total_pgpgout,omitempty"`
			TotalCache              uint64 `json:"total_cache,omitempty" yaml:"total_cache,omitempty" toml:"total_cache,omitempty"`
			InactiveAnon            uint64 `json:"inactive_anon,omitempty" yaml:"inactive_anon,omitempty" toml:"inactive_anon,omitempty"`
			ActiveFile              uint64 `json:"active_file,omitempty" yaml:"active_file,omitempty" toml:"active_file,omitempty"`
			Pgfault                 uint64 `json:"pgfault,omitempty" yaml:"pgfault,omitempty" toml:"pgfault,omitempty"`
			InactiveFile            uint64 `json:"inactive_file,omitempty" yaml:"inactive_file,omitempty" toml:"inactive_file,omitempty"`
			TotalPgpgin             uint64 `json:"total_pgpgin,omitempty" yaml:"total_pgpgin,omitempty" toml:"total_pgpgin,omitempty"`
			HierarchicalMemswLimit  uint64 `json:"hierarchical_memsw_limit,omitempty" yaml:"hierarchical_memsw_limit,omitempty" toml:"hierarchical_memsw_limit,omitempty"`
			Swap                    uint64 `json:"swap,omitempty" yaml:"swap,omitempty" toml:"swap,omitempty"`
		} `json:"stats,omitempty" yaml:"stats,omitempty" toml:"stats,omitempty"`
		MaxUsage uint64 `json:"max_usage,omitempty" yaml:"max_usage,omitempty" toml:"max_usage,omitempty"`
		Usage    uint64 `json:"usage,omitempty" yaml:"usage,omitempty" toml:"usage,omitempty"`
		Failcnt  uint64 `json:"failcnt,omitempty" yaml:"failcnt,omitempty" toml:"failcnt,omitempty"`
		Limit    uint64 `json:"limit,omitempty" yaml:"limit,omitempty" toml:"limit,omitempty"`
	}

	myMemoryStats := dc.Stats{
		Read: read,
		MemoryStats: memoryStatsStructure{
			MaxUsage: number,
			Usage:    number * 2,
			Failcnt:  number * 3,
			Limit:    number * 4,
		},
	}

	myMemoryStats.MemoryStats.Stats.TotalRss = number * 5

	return myMemoryStats
}
func equalEvent(expectedEvent common.MapStr, event common.MapStr) bool {
	return reflect.DeepEqual(expectedEvent, event)
}
