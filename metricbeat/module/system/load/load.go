// +build darwin freebsd linux openbsd windows

package load

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/pkg/errors"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "load", New); err != nil {
		panic(err)
	}
}

// MetricSet for fetching system CPU metrics.
type MetricSet struct {
	mb.BaseMetricSet
}

// New is a mb.MetricSetFactory that returns a load.MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

// Fetch fetches load metrics from the OS.
func (m *MetricSet) Fetch() (common.MapStr, error) {

	loadStat, err := GetSystemLoad()
	if err != nil {
		return nil, errors.Wrap(err, "load statistics")
	}

	load := common.MapStr{
		"1":  loadStat.Load1,
		"5":  loadStat.Load5,
		"15": loadStat.Load15,
	}

	return load, nil
}
