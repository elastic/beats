// +build darwin freebsd linux openbsd windows

package filesystem

import (
	"testing"

	"time"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	f := mbtest.NewEventsFetcher(t, getConfig())

	// Do a first fetch to have precentages
	f.Fetch()
	time.Sleep(1 * time.Second)

	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"filesystem"},
	}
}
