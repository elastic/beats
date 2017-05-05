// +build darwin freebsd linux openbsd windows

// +build darwin freebsd linux openbsd windows

package memory

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/shirou/gopsutil/mem"

	"github.com/pkg/errors"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "memory", New, parse.EmptyHostParser); err != nil {
		panic(err)
	}
}

// MetricSet for fetching system memory metrics.
type MetricSet struct {
	mb.BaseMetricSet
}

// New is a mb.MetricSetFactory that returns a memory.MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{base}, nil
}

// Fetch fetches memory metrics from the OS.
func (m *MetricSet) Fetch() (event common.MapStr, err error) {
	memStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, errors.Wrap(err, "memory")
	}

	memory := common.MapStr{
		"total": memStat.Total,
		"used": common.MapStr{
			"bytes": memStat.Used,
			"pct":   GetPercentage(memStat.Used, memStat.Total),
		},
		"free": memStat.Free,
		"actual": common.MapStr{
			"free": memStat.Available,
			"used": common.MapStr{
				"bytes": memStat.Total - memStat.Available,
				"pct":   GetPercentage(memStat.Total-memStat.Available, memStat.Total),
			},
		},
	}

	swapStat, err := mem.SwapMemory()
	if err != nil {
		return nil, errors.Wrap(err, "swap")
	}

	swap := common.MapStr{
		"total": swapStat.Total,
		"used": common.MapStr{
			"bytes": swapStat.Used,
			"pct":   GetPercentage(swapStat.Used, swapStat.Total),
		},
		"free": swapStat.Free,
	}

	memory["swap"] = swap
	return memory, nil
}
