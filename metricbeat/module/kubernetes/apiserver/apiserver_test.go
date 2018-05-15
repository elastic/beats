// +build !integration

package apiserver

import (
	"testing"

	"github.com/elastic/beats/metricbeat/helper/prometheus"
)

const testFile = "_meta/test/metrics"

func TestEventMapping(t *testing.T) {
	prometheus.TestMetricSet(t, "kubernetes", "apiserver",
		prometheus.TestCases{
			{
				MetricsFile:  "./_meta/test/metrics",
				ExpectedFile: "./_meta/test/metrics.expected",
			},
		},
	)
}
