package info

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urso/ucfg"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/module/redis"
)

// Test Redis specific struct
type RedisModuleConfig struct {
	Hosts      []string `config:"hosts"`
	Period     string   `config:"period"`
	Module     string   `config:"module"`
	MetricSets []string `config:"metricsets"`
	Enabled    bool     `config:"enabled"`
	MaxConn    int      `config:"maxconn"`
	Network    string   `config:"network"`

	common.EventMetadata `config:",inline"` // Fields and tags to add to events.
}

func TestConnect(t *testing.T) {

	if testing.Short() {
		t.Skip("Skipping in short mode, because it requires Redis")
	}

	config, _ := ucfg.NewFrom(RedisModuleConfig{
		Module:  "redis",
		Hosts:   []string{redis.GetRedisEnvHost() + ":" + redis.GetRedisEnvPort()},
		Network: "tcp",
		MaxConn: 10,
	})

	module, mErr := helper.NewModule(config, redis.New)
	ms, msErr := helper.NewMetricSet("info", New, module)
	assert.NoError(t, mErr)
	assert.NoError(t, msErr)

	data, err := ms.MetricSeter.Fetch(ms)
	assert.NoError(t, err)

	// Check fields
	assert.Equal(t, 8, len(data[0]))

	server := data[0]["server"].(common.MapStr)

	assert.Equal(t, "3.0.7", server["redis_version"])
	assert.Equal(t, "standalone", server["redis_mode"])
}
