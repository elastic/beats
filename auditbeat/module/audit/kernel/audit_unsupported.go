// +build !linux

package kernel

import (
	"errors"

	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

func init() {
	if err := mb.Registry.AddMetricSet("audit", "kernel", New, parse.EmptyHostParser); err != nil {
		panic(err)
	}
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return nil, errors.New("the audit.kernel metricset is only supported on linux")
}
