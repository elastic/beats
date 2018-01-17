package status

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	mbSql "github.com/elastic/beats/metricbeat/module/mysql"

	"github.com/stretchr/testify/assert"
	"regexp"
)

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "galera")

	f := mbtest.NewEventFetcher(t, getConfig(false, "full"))
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check event fields
	status := event["status"].(common.MapStr)
	clusterSize := status["cluster"].(common.MapStr)["size"].(int)
	clusterStatus := status["cluster"].(common.MapStr)["status"].(string)
	connected := status["cluster"].(common.MapStr)["connected"].(string)
	evsState := status["evs"].(common.MapStr)["state"].(string)
	localState := status["local"].(common.MapStr)["state"].(string)
	ready := status["reads"].(string)

	expState := regexp.MustCompile(`(?i)^primary|open|joiner|joined|synced|donor$`)
	expOnOff := regexp.MustCompile(`^ON|OFF$`)

	assert.True(t, clusterSize > 0)
	assert.Regexp(t, "^(NON_)?PRIMARY$", clusterStatus)
	assert.Regexp(t, expOnOff, connected)
	assert.Regexp(t, expState, evsState)
	assert.Regexp(t, expState, localState)
	assert.Regexp(t, expOnOff, ready)
}

func TestFetchRaw(t *testing.T) {
	compose.EnsureUp(t, "galera")

	f := mbtest.NewEventFetcher(t, getConfig(true, "full"))
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check event fields
	cachedThreads := event["threads"].(common.MapStr)["cached"].(int64)
	assert.True(t, cachedThreads >= 0)

	rawData := event["raw"].(common.MapStr)

	// Make sure field was removed from raw fields as in schema
	_, exists := rawData["Threads_cached"]
	assert.False(t, exists)

	// Check a raw field if it is available
	_, exists = rawData["Slow_launch_threads"]
	assert.True(t, exists)
}

func TestData(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig(false, "full"))

	err := mbtest.WriteEvent(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(raw bool, queryMode string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "mysql",
		"metricsets": []string{"status"},
		"hosts":      []string{mbSql.GetMySQLEnvDSN()},
		"raw":        raw,
		"query_mode": queryMode,
	}
}
