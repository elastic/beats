// +build integration

package memory

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

/*
// TODO: Enable
func TestFetch(t *testing.T) {
	f := mbtest.NewEventsFetcher(t, getConfig())
	event, err := f.Fetch()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf(" module : %s metricset : %s event: %+v", f.Module().Name(), f.Name(), event)
}*/

func TestData(t *testing.T) {
	f := mbtest.NewEventsFetcher(t, getConfig())
	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "docker",
		"metricsets": []string{"memory"},
		"hosts":      []string{"localhost"},
		"socket":     "unix:///var/run/docker.sock",
	}
}
