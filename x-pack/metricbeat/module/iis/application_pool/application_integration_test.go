package application_pool

import (
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFetch(t *testing.T) {
	config := map[string]interface{}{
		"module":            "iis",
		"period":            "30s",
		"metricsets":        []string{"application_pool"},
	}
	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(metricSet)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}

	assert.NotEmpty(t, events)


}

func TestData(t *testing.T) {
	config := map[string]interface{}{
		"module":            "iis",
		"period":            "30s",
		"metricsets":        []string{"application_pool"},
	}

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	if err := mbtest.WriteEventsReporterV2Error(metricSet, t, "/"); err != nil {
		t.Fatal("write", err)
	}
}

