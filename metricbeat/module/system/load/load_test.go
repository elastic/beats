// +build !integration
// +build darwin freebsd linux openbsd

package load

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig())

	load, err := f.Fetch()
	if err != nil {
		t.Fatal(err)
	}

	event := mbtest.CreateFullEvent(f, load)
	mbtest.WriteEventToDataJSON(t, event)
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"load"},
	}
}
