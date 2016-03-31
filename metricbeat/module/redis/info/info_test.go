// +build integration

package info

import (
	"testing"

	rd "github.com/garyburd/redigo/redis"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/module/redis"
)

// Test Redis specific struct
type RedisModuleConfig struct {
	Hosts    []string `config:"hosts"`
	Module   string   `config:"module"`
	Password string   `config:"password"`
}

var DEFAULT_PASS string = "foobared"
var LOCAL_REDIS string = redis.GetRedisEnvHost() + ":" + redis.GetRedisEnvPort()

func TestConnect(t *testing.T) {

	ms, msErr := getRedisMetricSet("")
	assert.NoError(t, msErr)

	// Setup metricset and metricseter
	err := ms.Setup()
	assert.NoError(t, err)
	err = ms.MetricSeter.Setup(ms)
	assert.NoError(t, err)

	// Check that host is correctly set
	assert.Equal(t, LOCAL_REDIS, ms.Config.Hosts[0])

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
	ms, msErr := getRedisMetricSet("")
	assert.NoError(t, msErr)

	ms.Setup()

	// Write to DB to enable Keyspace stats
	rErr := writeToRedis(LOCAL_REDIS)
	assert.NoError(t, rErr)

	// Fetch metrics
	data, err := ms.MetricSeter.Fetch(ms, LOCAL_REDIS)
	assert.NoError(t, err)
	keyspace := data["keyspace"].(map[string]common.MapStr)
	keyCount := keyspace["db0"]["keys"].(int)
	assert.True(t, (keyCount > 0))
}

func TestPasswords(t *testing.T) {

	// Config redis and metricset with no password
	ms, msErr := getRedisMetricSet("")
	assert.NoError(t, msErr)
	ms.Setup()

	// Add password and ensure it gets reset
	authErr := addPassword(LOCAL_REDIS, DEFAULT_PASS)
	defer resetPassword(LOCAL_REDIS)
	assert.NoError(t, authErr)

	// Test Fetch metrics with missing password
	_, err := ms.MetricSeter.Fetch(ms, LOCAL_REDIS)
	if assert.Error(t, err) {
		assert.Contains(t, err, "NOAUTH Authentication required.")
	}

	// Config redis and metricset with an invalid password
	ms2, msErr := getRedisMetricSet("blah")
	assert.NoError(t, msErr)
	ms2.Setup()

	// Test Fetch metrics with invalid password
	_, err2 := ms2.MetricSeter.Fetch(ms2, LOCAL_REDIS)
	if assert.Error(t, err2) {
		assert.Contains(t, err2, "ERR invalid password")
	}

	// Config redis and metricset with a valid password
	ms3, msErr := getRedisMetricSet(DEFAULT_PASS)
	assert.NoError(t, msErr)
	ms3.Setup()

	// Test Fetch metrics with valid password
	_, err3 := ms3.MetricSeter.Fetch(ms3, LOCAL_REDIS)
	assert.NoError(t, err3)
}

// addPassword will add password to redis
func addPassword(host, pass string) error {
	c, err := rd.Dial("tcp", host)
	if err != nil {
		return err
	}
	defer c.Close()

	_, cErr := c.Do("CONFIG", "SET", "requirepass", pass)
	if cErr != nil {
		return cErr
	}
	return nil
}

func resetPassword(host string) error {
	c, err := rd.Dial("tcp", host)
	if err != nil {
		return err
	}
	defer c.Close()

	_, pErr := c.Do("AUTH", DEFAULT_PASS)
	if pErr != nil {
		return pErr
	}

	_, cErr := c.Do("CONFIG", "SET", "requirepass", "")
	if cErr != nil {
		return cErr
	}
	return nil

}

func getRedisMetricSet(pass string) (*helper.MetricSet, error) {
	config, _ := getRedisModuleConfig(pass)
	module, mErr := helper.NewModule(config, redis.New)
	if mErr != nil {
		return nil, mErr
	}
	return helper.NewMetricSet("info", New, module)
}

func getRedisModuleConfig(pass string) (*common.Config, error) {
	return common.NewConfigFrom(RedisModuleConfig{
		Module:   "redis",
		Hosts:    []string{LOCAL_REDIS},
		Password: pass,
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
