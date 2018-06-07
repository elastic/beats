// +build integration

package node

import (
	"testing"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/rabbitmq/mtest"
)

func TestData(t *testing.T) {
	compose.EnsureUp(t, "rabbitmq")

	ms := mbtest.NewReportingMetricSetV2(t, getConfig())
	err := mbtest.WriteEventsReporterV2(ms, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	config := mtest.GetIntegrationConfig()
	config["metricsets"] = []string{"node"}
	return config
}
