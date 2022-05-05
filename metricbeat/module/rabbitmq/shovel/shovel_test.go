package shovel

import (
	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/rabbitmq/mtest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
)

func TestFetchEventContents(t *testing.T) {
	server := mtest.Server(t, mtest.DefaultServerConfig)
	defer server.Close()

	reporter := &mbtest.CapturingReporterV2{}

	metricSet := mbtest.NewReportingMetricSetV2Error(t, getConfig(server.URL))
	err := metricSet.Fetch(reporter)
	assert.NoError(t, err)

	e := mbtest.StandardizeEvent(metricSet, reporter.GetEvents()[0])
	t.Logf("%s/%s event: %+v", metricSet.Module().Name(), metricSet.Name(), e.Fields.StringToPrint())

	ee, _ := e.Fields.GetValue("rabbitmq.shovel")
	event := ee.(mapstr.M)

	assert.Equal(t, "testshovel", event["name"])
	assert.Equal(t, "running", event["state"])
	assert.Equal(t, "dynamic", event["type"])
}

func getConfig(url string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "rabbitmq",
		"metricsets": []string{"shovel"},
		"hosts":      []string{url},
	}
}