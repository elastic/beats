package broker

import (
	"testing"

	"github.com/elastic/beats/v7/metricbeat/helper/prometheus/ptest"
)

func TestEventMapping(t *testing.T) {
	ptest.TestMetricSet(t, "pulsar", "broker",
		ptest.TestCases{
			{
				MetricsFile:  "./_meta/test/broker",
				ExpectedFile: "./_meta/test/broker.expected",
			},
		},
	)
}
