// +build darwin freebsd linux openbsd windows

package cpu

import (
	"testing"
	"time"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig())
	_, err := f.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)

	fields, err := f.Fetch()
	if err != nil {
		t.Fatal(err)
	}

	event := mbtest.CreateFullEvent(f, fields)
	mbtest.WriteEventToDataJSON(t, event)
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":      "system",
		"metricsets":  []string{"cpu"},
		"cpu.metrics": []string{"percentages", "normalized_percentages", "ticks"},
	}
}
