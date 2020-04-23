package perf

import (
	"encoding/json"
	"log"
	"testing"
	"time"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/stretchr/testify/assert"
)

func TestProcessGet(t *testing.T) {

	testProcessList := []sampleConfig{sampleConfig{
		ProcessGlob: "sssd",
		Events:      eventsConfig{HardwareEvents: false, SoftwareEvents: true},
	},
	}

	matches, err := matchProcesses(testProcessList)
	assert.NoError(t, err)
	t.Logf("Got: %#v", matches)
}

func TestSampling(t *testing.T) {

	testProcessList := []sampleConfig{sampleConfig{
		ProcessGlob: "node",
		Events:      eventsConfig{HardwareEvents: false, SoftwareEvents: true},
	},
	}

	period := time.Second * 10

	matches, err := matchProcesses(testProcessList)
	assert.NoError(t, err)
	t.Logf("Got: %d", len(matches))

	perfData, err := runSampleForPeriod(matches, period, false)
	assert.NoError(t, err)
	for _, data := range perfData {
		log.Printf("Got data from %v", data.Metadata["pid"])

		pretty, err := json.MarshalIndent(data.HwMetrics, "", "    ")
		assert.NoError(t, err)
		log.Printf("%s\n", string(pretty))

		pretty, err = json.MarshalIndent(data.SwMetrics, "", "    ")
		assert.NoError(t, err)
		log.Printf("%s\n", string(pretty))

	}

}

func TestFetch(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	events, errs := mbtest.ReportingFetchV2Error(f)

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}

	//first := events[0]

	//t.Logf("Got: %#v", first.MetricSetFields)
	pretty, err := json.MarshalIndent(events, "", "    ")
	assert.NoError(t, err)
	log.Printf("%s\n", string(pretty))
}

func TestData(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	err := mbtest.WriteEventsReporterV2Error(f, t, ".")
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":             "linux",
		"metricsets":         []string{"perf"},
		"period":             "10s",
		"perf.sample_period": time.Second * 3,
		"perf.processes": []sampleConfig{
			sampleConfig{ProcessGlob: "node", Events: eventsConfig{SoftwareEvents: true, HardwareEvents: false}},
		},
	}
}
