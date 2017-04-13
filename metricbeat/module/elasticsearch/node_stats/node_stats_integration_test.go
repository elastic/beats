// +build integration

package node_stats

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"

	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	f := mbtest.NewEventsFetcher(t, elasticsearch.GetConfig("node_stats"))
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	assert.NotNil(t, event)
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
}

func TestData(t *testing.T) {
	f := mbtest.NewEventsFetcher(t, elasticsearch.GetConfig("node_stats"))
	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}
