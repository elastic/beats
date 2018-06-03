// +build integration

package stats

import (
	"testing"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/kibana/mtest"
)

func TestData(t *testing.T) {
	compose.EnsureUp(t, "kibana")

	f := mbtest.NewReportingMetricSetV2(t, mtest.GetConfig("stats"))
	err := mbtest.WriteEventsReporterV2(f, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}
