package mem

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
	assert.Contains(t, event["user_cache"], "size")
	assert.Contains(t, event["user_cache"], "used")
	assert.Contains(t, event["user_cache"], "free")
	assert.Contains(t, event["user_cache"], "memtotal")
	assert.Contains(t, event["credentials_cache"], "size")
	assert.Contains(t, event["db_cache"], "used")
	assert.Contains(t, event["peer_cache"], "free")
	assert.Contains(t, event["iobuf_cache"], "memtotal")
}
func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "pgbouncer",
		"metricsets": []string{"mem"},
		"hosts":      []string{"localhost:6432/pgbouncer?sslmode=disable"},
		"username":   "test",
		"password":   postgresql.GetEnvPassword(),
	}
}
