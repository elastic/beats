// +build !linux

package auditd

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

func init() {
	mb.Registry.MustAddMetricSet(metricsetName, metricsetName, New,
		mb.DefaultMetricSet(),
		mb.WithHostParser(parse.EmptyHostParser),
	)
}

// New constructs a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return nil, errors.Errorf("the %v module is only supported on Linux", metricsetName)
}
