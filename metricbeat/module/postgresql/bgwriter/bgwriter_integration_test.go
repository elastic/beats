// +build integration

package bgwriter

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/postgresql"

	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "postgresql")

	f := mbtest.NewEventFetcher(t, getConfig())
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	assert.Contains(t, event, "checkpoints")
	assert.Contains(t, event, "buffers")
	assert.Contains(t, event, "stats_reset")

	checkpoints := event["checkpoints"].(common.MapStr)
	assert.Contains(t, checkpoints, "scheduled")
	assert.Contains(t, checkpoints, "requested")
	assert.Contains(t, checkpoints, "times")

	buffers := event["buffers"].(common.MapStr)
	assert.Contains(t, buffers, "checkpoints")
	assert.Contains(t, buffers, "clean")
	assert.Contains(t, buffers, "clean_full")
	assert.Contains(t, buffers, "backend")
	assert.Contains(t, buffers, "backend_fsync")
	assert.Contains(t, buffers, "allocated")
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "postgresql")

	f := mbtest.NewEventFetcher(t, getConfig())

	err := mbtest.WriteEvent(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "postgresql",
		"metricsets": []string{"bgwriter"},
		"hosts":      []string{postgresql.GetEnvDSN()},
		"username":   postgresql.GetEnvUsername(),
		"password":   postgresql.GetEnvPassword(),
	}
}
