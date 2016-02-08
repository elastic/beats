package status

import (
	"testing"

	"github.com/elastic/beats/metricbeat/module/mysql"
	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {

	if testing.Short() {
		t.Skip("Skipping in short mode, because it requires MySQL")
	}

	// Setup Module config
	mysql.Module.BaseConfig.Hosts = []string{mysql.GetMySQLEnvDSN()}

	// Setup Metric
	m := MetricSeter{}
	err := m.Setup()
	assert.NoError(t, err)

	// Load events
	events, err := m.Fetch()
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
