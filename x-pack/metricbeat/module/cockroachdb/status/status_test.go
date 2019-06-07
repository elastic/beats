// +build !integration

package status

import (
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper/prometheus/ptest"
	"github.com/elastic/beats/metricbeat/mb"

	// Register input module and metricset
	_ "github.com/elastic/beats/metricbeat/module/prometheus"
	_ "github.com/elastic/beats/metricbeat/module/prometheus/collector"
)

func init() {
	// To be moved to some kind of helper
	os.Setenv("BEAT_STRICT_PERMS", "false")
	mb.Registry.SetSecondarySource(mb.NewLightModulesSource("../../../module"))
}

func TestEventMapping(t *testing.T) {
	logp.TestingSetup()

	ptest.TestMetricSet(t, "cockroachdb", "status",
		ptest.TestCases{
			{
				MetricsFile:  "./_meta/test/cockroachdb-status.v19.1.1",
				ExpectedFile: "./_meta/test/cockroachdb-status.v19.1.1.expected",
			},
		},
	)
}
