// +build darwin linux openbsd windows

package memory

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/topbeat/system"

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
func (m *MetricSet) Fetch(host string) (event common.MapStr, err error) {
	memStat, err := system.GetMemory()
	if err != nil {
		return nil, errors.Wrap(err, "memory")
	}
	system.AddMemPercentage(memStat)

	swapStat, err := system.GetSwap()
	if err != nil {
		return nil, errors.Wrap(err, "swap")
	}
	system.AddSwapPercentage(swapStat)

	return common.MapStr{
		"mem":  system.GetMemoryEvent(memStat),
		"swap": system.GetSwapEvent(swapStat),
	}, nil
}
