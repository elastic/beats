// +build integration

package status

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/mysql"

	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig())
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check event fields
	connections := event["connections"].(int64)
	open := event["open"].(common.MapStr)
	openTables := open["tables"].(int64)
	openFiles := open["files"].(int64)
	openStreams := open["streams"].(int64)

	assert.True(t, connections > 0)
	assert.True(t, openTables > 0)
	assert.True(t, openFiles >= 0)
	assert.True(t, openStreams == 0)
}

func TestData(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig())

	err := mbtest.WriteEvent(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "mysql",
		"metricsets": []string{"status"},
		"hosts":      []string{mysql.GetMySQLEnvDSN()},
	}
}
