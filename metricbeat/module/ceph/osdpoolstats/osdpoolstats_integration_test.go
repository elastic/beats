package osdpoolstats

import (
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestData(t *testing.T) {
	f := mbtest.NewEventsFetcher(t, getConfig())
	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func TestFetch(t *testing.T) {
	f := mbtest.NewEventsFetcher(t, getConfig())
	events, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	assert.True(t, len(events) > 0)

	event := events[0]
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":      "ceph",
		"metricsets":  []string{"osdpoolstats"},
		"binary_path": "/usr/bin/docker",
	}
}
