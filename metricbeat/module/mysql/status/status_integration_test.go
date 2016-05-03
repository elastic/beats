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
	event, err := f.Fetch(f.Module().Config().Hosts[0])
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check event fields
	connections := event["Connections"].(int)
	open := event["open"].(common.MapStr)
	openTables := open["Open_tables"].(int)
	openFiles := open["Open_files"].(int)
	openStreams := open["Open_streams"].(int)

	assert.True(t, connections > 0)
	assert.True(t, openTables > 0)
	assert.True(t, openFiles >= 0)
	assert.True(t, openStreams == 0)
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "mysql",
		"metricsets": []string{"status"},
		"hosts":      []string{mysql.GetMySQLEnvDSN()},
	}
}
