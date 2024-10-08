package lists

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
	assert.Contains(t, event, "databases")
	assert.Contains(t, event, "users")
	assert.Contains(t, event, "peers")
	assert.Contains(t, event, "pools")
	assert.Contains(t, event, "peer_pools")
	assert.Contains(t, event, "used_clients")
	assert.Contains(t, event, "free_servers")
	assert.Contains(t, event, "used_servers")
}
func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "pgbouncer",
		"metricsets": []string{"lists"},
		"hosts":      []string{"localhost:6432/pgbouncer?sslmode=disable"},
		"username":   "test",
		"password":   postgresql.GetEnvPassword(),
	}
}
