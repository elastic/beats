package info

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/module/redis"
)

func TestConnect(t *testing.T) {

	if testing.Short() {
		t.Skip("Skipping in short mode, because it requires Redis")
	}

	config := helper.ModuleConfig{
		Hosts: []string{redis.GetRedisEnvHost() + ":" + redis.GetRedisEnvPort()},
	}
	module := &helper.Module{
		Config: config,
	}
	ms := helper.NewMetricSet("info", New, module)

	data, err := ms.MetricSeter.Fetch(ms)
	assert.NoError(t, err)

	// Check fields
	assert.Equal(t, 8, len(data[0]))

	server := data[0]["server"].(common.MapStr)

	assert.Equal(t, "3.0.7", server["redis_version"])
	assert.Equal(t, "standalone", server["redis_mode"])
}
