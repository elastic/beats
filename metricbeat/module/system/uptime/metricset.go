// +build darwin linux openbsd windows

package uptime

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	sigar "github.com/elastic/gosigar"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "uptime", New, parse.EmptyHostParser); err != nil {
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
	var uptime sigar.Uptime
	if err := uptime.Get(); err != nil {
		return nil, errors.Wrap(err, "failed to get uptime")
	}

	return common.MapStr{
		"duration": common.MapStr{
			"ms": int64(uptime.Length * 1000),
		},
	}, nil
}
