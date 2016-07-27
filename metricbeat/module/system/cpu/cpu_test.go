// +build !integration
// +build darwin freebsd linux openbsd windows

package cpu

import (
	"testing"

	"time"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig())

	// Do a first fetch to have precentages
	f.Fetch()
	time.Sleep(1 * time.Second)

	err := mbtest.WriteEvent(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"cpu"},
		"cpu_ticks":  true,
	}
}
