package info

import (
	"testing"

	"github.com/elastic/beats/metricbeat/module/redis"
	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {

	if testing.Short() {
		t.Skip("Skipping in short mode, because it requires Redis")
	}

	// Setup
	redis.Module.BaseConfig.Hosts = []string{redis.GetRedisEnvHost() + ":" + redis.GetRedisEnvPort()}
	r := MetricSeter{}
	err := r.Setup()
	assert.NoError(t, err)

	data, err := r.Fetch()
	assert.NoError(t, err)

	// Check fields
	assert.Equal(t, 4, len(data[0]))
	assert.Equal(t, "3.0.6", data[0]["version"])
	assert.Equal(t, "standalone", data[0]["mode"])
}
