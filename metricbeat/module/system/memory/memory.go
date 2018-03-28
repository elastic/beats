// +build darwin freebsd linux openbsd windows

package memory

import (
	"github.com/elastic/beats/libbeat/common"
	mem "github.com/elastic/beats/libbeat/metric/system/memory"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"

	"github.com/pkg/errors"
)

func init() {
	mb.Registry.MustAddMetricSet("system", "memory", New,
		mb.WithHostParser(parse.EmptyHostParser),
		mb.DefaultMetricSet(),
	)
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
	memStat, err := mem.Get()
	if err != nil {
		return nil, errors.Wrap(err, "memory")
	}
	mem.AddMemPercentage(memStat)

	swapStat, err := mem.GetSwap()
	if err != nil {
		return nil, errors.Wrap(err, "swap")
	}
	mem.AddSwapPercentage(swapStat)

	memory := common.MapStr{
		"total": memStat.Total,
		"used": common.MapStr{
			"bytes": memStat.Used,
			"pct":   memStat.UsedPercent,
		},
		"free": memStat.Free,
		"actual": common.MapStr{
			"free": memStat.ActualFree,
			"used": common.MapStr{
				"pct":   memStat.ActualUsedPercent,
				"bytes": memStat.ActualUsed,
			},
		},
	}

	swap := common.MapStr{
		"total": swapStat.Total,
		"used": common.MapStr{
			"bytes": swapStat.Used,
			"pct":   swapStat.UsedPercent,
		},
		"free": swapStat.Free,
	}
	memory["swap"] = swap

	hugePagesStat, err := mem.GetHugeTLBPages()
	if err != nil {
		return nil, errors.Wrap(err, "hugepages")
	}
	if hugePagesStat != nil {
		mem.AddHugeTLBPagesPercentage(hugePagesStat)
		memory["hugepages"] = common.MapStr{
			"total": hugePagesStat.Total,
			"used": common.MapStr{
				"bytes": hugePagesStat.TotalAllocatedSize,
				"pct":   hugePagesStat.UsedPercent,
			},
			"free":         hugePagesStat.Free,
			"reserved":     hugePagesStat.Reserved,
			"surplus":      hugePagesStat.Surplus,
			"default_size": hugePagesStat.DefaultSize,
		}
	}

	return memory, nil
}
