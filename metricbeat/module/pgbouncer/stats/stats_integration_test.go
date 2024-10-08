package stats

import (
	"fmt"
	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/postgresql"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
)

func TestMetricSet_Fetch(t *testing.T) {
	service := compose.EnsureUp(t, "pgbouncer")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	fmt.Printf("%v", f)
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	event := events[0].MetricSetFields
	assert.Contains(t, event, "total_xact_count")
	assert.Contains(t, event, "total_server_assignment_count")
	assert.Contains(t, event, "total_wait_time_us")
	assert.Contains(t, event, "database")
	assert.Contains(t, event, "total_received")
	assert.Contains(t, event, "total_query_count")
}
func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "pgbouncer",
		"metricsets": []string{"stats"},
		"hosts":      []string{"localhost:6432/pgbouncer?sslmode=disable"},
		"username":   "test",
		"password":   postgresql.GetEnvPassword(),
	}
}
