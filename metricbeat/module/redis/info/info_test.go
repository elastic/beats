// +build integration

package info

import (
	"testing"

	rd "github.com/garyburd/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/urso/ucfg"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/module/redis"
)

// Test Redis specific struct
type RedisModuleConfig struct {
	Hosts  []string `config:"hosts"`
	Module string   `config:"module"`
}

func TestConnect(t *testing.T) {

	config, _ := getRedisModuleConfig()

	module, mErr := helper.NewModule(config, redis.New)
	assert.NoError(t, mErr)
	ms, msErr := helper.NewMetricSet("info", New, module)
	assert.NoError(t, msErr)

	// Setup metricset and metricseter
	err := ms.Setup()
	assert.NoError(t, err)
	err = ms.MetricSeter.Setup(ms)
	assert.NoError(t, err)

	// Check that host is correctly set
	assert.Equal(t, redis.GetRedisEnvHost()+":"+redis.GetRedisEnvPort(), ms.Config.Hosts[0])

	data, err := ms.MetricSeter.Fetch(ms, ms.Config.Hosts[0])
	assert.NoError(t, err)

	// Check fields
	assert.Equal(t, 9, len(data))
	server := data["server"].(common.MapStr)
	assert.Equal(t, "3.0.7", server["redis_version"])
	assert.Equal(t, "standalone", server["redis_mode"])
}

func TestKeyspace(t *testing.T) {

	// Config redis and metricset
	config, _ := getRedisModuleConfig()
	module, mErr := helper.NewModule(config, redis.New)
	ms, msErr := helper.NewMetricSet("info", New, module)
	assert.NoError(t, mErr)
	assert.NoError(t, msErr)

	ms.Setup()

	// Write to DB to enable Keyspace stats
	rErr := writeToRedis(redis.GetRedisEnvHost() + ":" + redis.GetRedisEnvPort())
	assert.NoError(t, rErr)

	// Fetch metrics
	data, err := ms.MetricSeter.Fetch(ms, redis.GetRedisEnvHost()+":"+redis.GetRedisEnvPort())
	assert.NoError(t, err)
	keyspace := data["keyspace"].(map[string]common.MapStr)
	keyCount := keyspace["db0"]["keys"].(int)
	assert.True(t, (keyCount > 0))
}

func getRedisModuleConfig() (*ucfg.Config, error) {
	return ucfg.NewFrom(RedisModuleConfig{
		Module: "redis",
		Hosts:  []string{redis.GetRedisEnvHost() + ":" + redis.GetRedisEnvPort()},
	})
}

// writeToRedis will write to the default DB 0
func writeToRedis(host string) error {
	c, err := rd.Dial("tcp", host)
	if err != nil {
		return err
	}
	defer c.Close()

	_, cErr := c.Do("SET", "foo", "bar")
	if cErr != nil {
		return cErr
	}
	return nil
}
