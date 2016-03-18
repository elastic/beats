// +build integration

package status

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/module/mysql"
)

func TestFetch(t *testing.T) {

	config := helper.ModuleConfig{
		Hosts: []string{mysql.GetMySQLEnvDSN()},
	}
	module := &helper.Module{
		Config: config,
	}

	ms, msErr := helper.NewMetricSet("status", New, module)
	assert.NoError(t, msErr)

	var err error

	// Setup metricset
	err = ms.Setup()
	assert.NoError(t, err)

	// Load events
	event, err := ms.MetricSeter.Fetch(ms, module.Config.Hosts[0])
	assert.NoError(t, err)

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
