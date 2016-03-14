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

	config, _ := getRedisModuleConfig()
	module, mErr := helper.NewModule(config, redis.New)
	ms, msErr := helper.NewMetricSet("info", New, module)
	assert.NoError(t, mErr)
	assert.NoError(t, msErr)

	ms.Setup()
	data, err := ms.MetricSeter.Fetch(ms)
	assert.NoError(t, err)

	// Check fields
	assert.Equal(t, 9, len(data[0]))

	server := data[0]["server"].(common.MapStr)

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
	data, err := ms.MetricSeter.Fetch(ms)
	assert.NoError(t, err)
	keyspace := data[0]["keyspace"].(map[string]common.MapStr)
	keyCount := keyspace["db0"]["keys"].(int)
	assert.True(t, (keyCount > 0))
}

func getRedisModuleConfig() (*ucfg.Config, error) {
	return ucfg.NewFrom(RedisModuleConfig{
		Module:  "redis",
		Hosts:   []string{redis.GetRedisEnvHost() + ":" + redis.GetRedisEnvPort()},
		Network: "tcp",
		MaxConn: 10,
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
