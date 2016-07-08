// +build darwin linux openbsd

package uptime

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"

	sigar "github.com/elastic/gosigar"
	"github.com/pkg/errors"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "uptime", New); err != nil {
		panic(err)
	}
}

// MetricSet for fetching an OS uptime metric.
type MetricSet struct {
	mb.BaseMetricSet
}

// New is a mb.MetricSetFactory that returns a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{base}, nil
}

// Fetch fetches the uptime metric from the OS.
func (m *MetricSet) Fetch() (common.MapStr, error) {
	uptime := sigar.Uptime{}
	if err := uptime.Get(); err != nil {
		return nil, errors.Wrap(err, "error fetching uptime")
	}

	return common.MapStr{
		"uptime_ms": int64(uptime.Length * 1000),
	}, nil
}
