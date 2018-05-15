// +build !integration

package apiserver

import (
	"testing"

	"github.com/elastic/beats/metricbeat/helper/prometheus/ptest"
)

const testFile = "_meta/test/metrics"

func TestEventMapping(t *testing.T) {
	ptest.TestMetricSet(t, "kubernetes", "apiserver",
		ptest.TestCases{
			{
				MetricsFile:  "./_meta/test/metrics",
				ExpectedFile: "./_meta/test/metrics.expected",
			},
		},
	)
}
