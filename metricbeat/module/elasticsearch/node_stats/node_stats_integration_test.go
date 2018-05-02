// +build integration

package node_stats

import (
	"testing"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"

	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "elasticsearch")

	f := mbtest.NewReportingMetricSetV2(t, elasticsearch.GetConfig("node_stats"))
	events, errs := mbtest.ReportingFetchV2(f)

	assert.NotNil(t, events)
	assert.Nil(t, errs)
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), events[0])
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "elasticsearch")

	f := mbtest.NewReportingMetricSetV2(t, elasticsearch.GetConfig("node_stats"))
	err := mbtest.WriteEventsReporterV2(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}
