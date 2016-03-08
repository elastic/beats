package status

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/module/mysql"
)

func TestFetch(t *testing.T) {

	if testing.Short() {
		t.Skip("Skipping in short mode, because it requires MySQL")
	}

	config := helper.ModuleConfig{
		Hosts: []string{mysql.GetMySQLEnvDSN()},
	}
	module := &helper.Module{
		Config: config,
	}
	ms := helper.NewMetricSet("status", New, module)

	// Load events
	events, err := ms.MetricSeter.Fetch(ms)
	assert.NoError(t, err)

	// Check event fields
	connections := events[0]["Connections"].(int)
	openTables := events[0]["Open_tables"].(int)
	openFiles := events[0]["Open_files"].(int)
	openStreams := events[0]["Open_streams"].(int)

	assert.True(t, connections > 0)
	assert.True(t, openTables > 0)
	assert.True(t, openFiles > 0)
	assert.True(t, openStreams == 0)
}
