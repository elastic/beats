package info

import (
	"testing"

	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/module/redis"
	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {

	if testing.Short() {
		t.Skip("Skipping in short mode, because it requires Redis")
	}

	// Setup
	r := &MetricSeter{}

	config := helper.ModuleConfig{
		Hosts: []string{redis.GetRedisEnvHost() + ":" + redis.GetRedisEnvPort()},
	}
	module := &helper.Module{
		Config: config,
	}
	ms := helper.NewMetricSet("info", r, module)

	data, err := r.Fetch(ms)
	assert.NoError(t, err)

	// Check fields
	assert.Equal(t, 4, len(data[0]))
	assert.Equal(t, "3.0.6", data[0]["version"])
	assert.Equal(t, "standalone", data[0]["mode"])
}
