// +build darwin freebsd linux openbsd windows

// +build darwin freebsd linux openbsd windows

package memory

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/system"
	"github.com/shirou/gopsutil/mem"

	"github.com/pkg/errors"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "memory", New); err != nil {
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
func (m *MetricSet) Fetch() (common.MapStr, error) {

	memStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, errors.Wrap(err, "memory")
	}

	swapStat, err := mem.SwapMemory()
	if err != nil {
		return nil, errors.Wrap(err, "swap")
	}

	memory := common.MapStr{
		"total": memStat.Total,
		"used": common.MapStr{
			"bytes": memStat.Used,
			"pct":   system.Round(memStat.UsedPercent/100, .5, 4),
		},
		"free":      memStat.Free,
		"available": memStat.Available,
	}
	// OSX /BSD specific
	if memStat.Active != 0 {
		memory["active"] = memStat.Active
	}
	if memStat.Inactive != 0 {
		memory["inactive"] = memStat.Inactive
	}
	if memStat.Wired != 0 {
		memory["wired"] = memStat.Wired
	}

	// Linux specific
	if memStat.Buffers != 0 {
		memory["buffers"] = memStat.Buffers
	}
	if memStat.Cached != 0 {
		memory["cached"] = memStat.Cached
	}

	swap := common.MapStr{
		"total": swapStat.Total,
		"used": common.MapStr{
			"bytes": swapStat.Used,
			"pct":   system.Round(swapStat.UsedPercent/100, .5, 4),
		},
		"free": swapStat.Free,
		"in":   swapStat.Sin,
		"out":  swapStat.Sout,
	}

	event := common.MapStr{
		"memory": memory,
		"swap":   swap,
	}
	return event, nil
}
