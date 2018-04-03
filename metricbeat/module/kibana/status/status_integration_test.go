// +build integration

package status

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/kibana/mtest"
)

func TestFetch(t *testing.T) {
	compose.EnsureUpWithTimeout(t, 600, "elasticsearch", "kibana")

	f := mbtest.NewEventFetcher(t, mtest.GetConfig("status"))
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "elasticsearch", "kibana")

	f := mbtest.NewEventFetcher(t, mtest.GetConfig("status"))
	err := mbtest.WriteEvent(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}
