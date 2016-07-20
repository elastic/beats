// +build darwin freebsd linux openbsd windows

package swap

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/pkg/errors"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "swap", New); err != nil {
		panic(err)
	}
}

// MetricSet for fetching system swap metrics.
type MetricSet struct {
	mb.BaseMetricSet
}

// New is a mb.MetricSetFactory that returns a swap.MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{base}, nil
}

// Fetch fetches swap metrics from the OS.
func (m *MetricSet) Fetch() (event common.MapStr, err error) {
	swapStat, err := GetSwap()
	if err != nil {
		return nil, errors.Wrap(err, "swap")
	}

	AddSwapPercentage(swapStat)

	swap := common.MapStr{
		"total": swapStat.Total,
		"used": common.MapStr{
			"bytes": swapStat.Used,
			"pct":   swapStat.UsedPercent,
		},
		"free": swapStat.Free,
	}

	return swap, nil
}
