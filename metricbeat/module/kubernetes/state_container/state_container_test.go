// +build !integration

package state_container

import (
	"testing"

	"github.com/elastic/beats/metricbeat/helper/prometheus/ptest"
)

func TestEventMapping(t *testing.T) {
	ptest.TestMetricSetEventsFetcher(t, "kubernetes", "state_container",
		ptest.TestCases{
			{
				MetricsFile:  "../_meta/test/kube-state-metrics",
				ExpectedFile: "./_meta/test/kube-state-metrics.expected",
			},
			{
				MetricsFile:  "../_meta/test/kube-state-metrics.v1.3.0",
				ExpectedFile: "./_meta/test/kube-state-metrics.v1.3.0.expected",
			},
		},
	)
}
