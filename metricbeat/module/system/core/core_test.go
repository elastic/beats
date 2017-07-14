// +build darwin freebsd linux openbsd windows

package core

import (
	"testing"
	"time"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	f := mbtest.NewReportingMetricSet(t, getConfig())

	mbtest.ReportingFetch(f)
	time.Sleep(500 * time.Millisecond)

	events, errs := mbtest.ReportingFetch(f)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	if len(events) == 0 {
		t.Fatal("no events returned")
	}

	event := mbtest.CreateFullEvent(f, events[1])
	mbtest.WriteEventToDataJSON(t, event)
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":       "system",
		"metricsets":   []string{"core"},
		"core.metrics": []string{"percentages", "ticks"},
	}
}
