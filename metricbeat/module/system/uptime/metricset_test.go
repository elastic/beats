// +build darwin linux openbsd windows

package uptime

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig())

	uptime, err := f.Fetch()
	if err != nil {
		t.Fatal(err)
	}

	event := mbtest.CreateFullEvent(f, uptime)
	mbtest.WriteEventToDataJSON(t, event)
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"uptime"},
	}
}
