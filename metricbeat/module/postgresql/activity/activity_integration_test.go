// +build integration

package activity

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/postgresql"

	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	f := mbtest.NewEventsFetcher(t, getConfig())
	events, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	assert.True(t, len(events) > 0)
	event := events[0]

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check event fields
	assert.Contains(t, event, "database")
	db_oid := event["database"].(common.MapStr)["oid"].(int64)
	assert.True(t, db_oid > 0)

	assert.Contains(t, event, "pid")
	assert.True(t, event["pid"].(int64) > 0)

	assert.Contains(t, event, "user")
	assert.Contains(t, event["user"].(common.MapStr), "name")
	assert.Contains(t, event["user"].(common.MapStr), "id")

}

func TestData(t *testing.T) {
	f := mbtest.NewEventsFetcher(t, getConfig())

	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "postgresql",
		"metricsets": []string{"activity"},
		"hosts":      []string{postgresql.GetEnvDSN()},
		"username":   postgresql.GetEnvUsername(),
		"password":   postgresql.GetEnvPassword(),
	}
}
