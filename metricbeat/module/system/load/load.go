// +build darwin freebsd linux openbsd

package load

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/system"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "load", New, parse.EmptyHostParser); err != nil {
		panic(err)
	}
}

// MetricSet for fetching system CPU load metrics.
type MetricSet struct {
	mb.BaseMetricSet
}

// New returns a new load MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

// Fetch fetches system load metrics.
func (m *MetricSet) Fetch() (common.MapStr, error) {
	load, err := system.Load()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get CPU load values")
	}

	avgs := load.Averages()
	normAvgs := load.NormalizedAverages()

	event := common.MapStr{
		"cores": system.NumCPU,
		"1":     avgs.OneMinute,
		"5":     avgs.FiveMinute,
		"15":    avgs.FifteenMinute,
		"norm": common.MapStr{
			"1":  normAvgs.OneMinute,
			"5":  normAvgs.FiveMinute,
			"15": normAvgs.FifteenMinute,
		},
	}

	return event, nil
}
