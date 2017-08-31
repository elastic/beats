// +build integration

package database

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

	f := mbtest.NewEventsFetcher(t, getConfig())
	events, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	assert.True(t, len(events) > 0)
	event := events[0]

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check event fields
	db_oid := event["oid"].(int64)
	assert.True(t, db_oid > 0)
	assert.Contains(t, event, "name")
	_, ok := event["name"].(string)
	assert.True(t, ok)

	rows := event["rows"].(common.MapStr)
	assert.Contains(t, rows, "returned")
	assert.Contains(t, rows, "fetched")
	assert.Contains(t, rows, "inserted")
	assert.Contains(t, rows, "updated")
	assert.Contains(t, rows, "deleted")
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "postgresql")

	f := mbtest.NewEventsFetcher(t, getConfig())

	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "postgresql",
		"metricsets": []string{"database"},
		"hosts":      []string{postgresql.GetEnvDSN()},
		"username":   postgresql.GetEnvUsername(),
		"password":   postgresql.GetEnvPassword(),
	}
}
