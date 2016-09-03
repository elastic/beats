// +build integration

package redis

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/garyburd/redigo/redis"
	"github.com/stretchr/testify/assert"
)

const (
	RedisDefaultHost = "localhost"
	RedisDefaultPort = "6379"

	SRedisDefaultHost = "localhost"
	SRedisDefaultPort = "6380"
)

func TestTopologyInRedisTCP(t *testing.T) {
	db := 1
	key := "test_topo_tcp"
	redisHosts := []string{getRedisAddr()}
	redisConfig := map[string]interface{}{
		"hosts":         redisHosts,
		"key":           key,
		"host_topology": redisHosts[0],
		"db_topology":   db,
		"timeout":       "5s",
	}

	testTopologyInRedis(t, redisConfig)
}

func TestTopologyInRedisTLS(t *testing.T) {
	db := 1
	key := "test_topo_tls"
	redisHosts := []string{getSRedisAddr()}
	redisConfig := map[string]interface{}{
		"hosts":         redisHosts,
		"key":           key,
		"host_topology": redisHosts[0],
		"db_topology":   db,
		"timeout":       "5s",

		"ssl.verification_mode": "full",
		"ssl.certificate_authorities": []string{
			"../../../testing/environments/docker/sredis/pki/tls/certs/sredis.crt",
		},
	}

	testTopologyInRedis(t, redisConfig)
}

func testTopologyInRedis(t *testing.T, cfg map[string]interface{}) {
	tests := []struct {
		out  *redisOut
		name string
		ips  []string
	}{
		{nil, "proxy1", []string{"10.1.0.4"}},
		{nil, "proxy2", []string{"10.1.0.9", "fe80::4e8d:79ff:fef2:de6a"}},
		{nil, "proxy3", []string{"10.1.0.10"}},
	}

	db := 0
	key := cfg["key"].(string)
	if v, ok := cfg["db_topology"]; ok {
		db = v.(int)
	}

	// prepare redis
	{
		conn, err := redis.Dial("tcp", getRedisAddr(), redis.DialDatabase(db))
		if err != nil {
			t.Fatalf("redis.Dial failed %v", err)
		}
		// delete old key if present
		defer conn.Close()
		conn.Do("DEL", key)
	}

	// 1. connect
	for i := range tests {
		tests[i].out = newRedisTestingOutput(t, cfg)
		defer tests[i].out.Close()
	}

	// 2. publish ips twice (so all outputs have same topology map)
	for i := 0; i < 2; i++ {
		for _, test := range tests {
			t.Logf("publish %v ips: %v", test.name, test.ips)
			err := test.out.PublishIPs(test.name, test.ips)
			assert.NoError(t, err)
		}
	}

	// 3. check names available
	for _, test := range tests {
		t.Logf("check %v knows ips", test.name)
		for _, other := range tests {
			t.Logf("  check ips of %v", other.name)
			for _, ip := range other.ips {
				name := test.out.GetNameByIP(ip)
				t.Logf("  check ip: %v -> %v", ip, other.name == name)
				assert.Equal(t, other.name, name)
			}
		}
	}
}

func TestPublishListTCP(t *testing.T) {
	key := "test_publist_tcp"
	db := 0
	redisConfig := map[string]interface{}{
		"hosts":    []string{getRedisAddr()},
		"key":      key,
		"db":       db,
		"datatype": "list",
		"timeout":  "5s",
	}

	testPublishList(t, redisConfig)
}

func TestPublishListTLS(t *testing.T) {
	key := "test_publist_tls"
	db := 0
	redisConfig := map[string]interface{}{
		"hosts":    []string{getSRedisAddr()},
		"key":      key,
		"db":       db,
		"datatype": "list",
		"timeout":  "5s",

		"ssl.verification_mode": "full",
		"ssl.certificate_authorities": []string{
			"../../../testing/environments/docker/sredis/pki/tls/certs/sredis.crt",
		},
	}

	testPublishList(t, redisConfig)
}

func testPublishList(t *testing.T, cfg map[string]interface{}) {
	batches := 100
	batchSize := 1000
	total := batches & batchSize

	db := 0
	key := cfg["key"].(string)
	if v, ok := cfg["db"]; ok {
		db = v.(int)
	}

	conn, err := redis.Dial("tcp", getRedisAddr(), redis.DialDatabase(db))
	if err != nil {
		t.Fatalf("redis.Dial failed %v", err)
	}

	// delete old key if present
	defer conn.Close()
	conn.Do("DEL", key)

	out := newRedisTestingOutput(t, cfg)
	err = sendTestEvents(out, batches, batchSize)
	assert.NoError(t, err)

	results := make([][]byte, total)
	for i := range results {
		results[i], err = redis.Bytes(conn.Do("LPOP", key))
		assert.NoError(t, err)
	}

	for i, raw := range results {
		evt := struct{ Message int }{}
		err = json.Unmarshal(raw, &evt)
		assert.NoError(t, err)
		assert.Equal(t, i+1, evt.Message)
	}
}

func TestPublishChannelTCP(t *testing.T) {
	db := 0
	key := "test_pubchan_tcp"
	redisConfig := map[string]interface{}{
		"hosts":    []string{getRedisAddr()},
		"key":      key,
		"db":       db,
		"datatype": "channel",
		"timeout":  "5s",
	}

	testPublishChannel(t, redisConfig)
}

func TestPublishChannelTLS(t *testing.T) {
	db := 0
	key := "test_pubchan_tls"
	redisConfig := map[string]interface{}{
		"hosts":    []string{getSRedisAddr()},
		"key":      key,
		"db":       db,
		"datatype": "channel",
		"timeout":  "5s",

		"ssl.verification_mode": "full",
		"ssl.certificate_authorities": []string{
			"../../../testing/environments/docker/sredis/pki/tls/certs/sredis.crt",
		},
	}

	testPublishChannel(t, redisConfig)
}

func testPublishChannel(t *testing.T, cfg map[string]interface{}) {
	batches := 100
	batchSize := 1000
	total := batches & batchSize

	db := 0
	key := cfg["key"].(string)
	if v, ok := cfg["db"]; ok {
		db = v.(int)
	}

	conn, err := redis.Dial("tcp", getRedisAddr(), redis.DialDatabase(db))
	if err != nil {
		t.Fatalf("redis.Dial failed %v", err)
	}

	// delete old key if present
	defer conn.Close()
	conn.Do("DEL", key)

	// subscribe to packetbeat channel
	psc := redis.PubSubConn{conn}
	if err := psc.Subscribe(key); err != nil {
		t.Fatal(err)
	}
	defer psc.Unsubscribe(key)

	// connect and publish events
	var wg sync.WaitGroup
	var pubErr error
	out := newRedisTestingOutput(t, cfg)
	wg.Add(1)
	go func() {
		defer wg.Done()
		pubErr = sendTestEvents(out, batches, batchSize)
	}()

	// collect published events by subscription
	var messages [][]byte
	assert.NoError(t, conn.Err())
	for conn.Err() == nil {
		t.Logf("try collect message")

		switch v := psc.Receive().(type) {
		case redis.Message:
			messages = append(messages, v.Data)
		case error:
			t.Error(v)
		default:
			t.Logf("received: %#v", v)
		}

		if len(messages) == total {
			break
		}
	}
	wg.Wait()

	// validate
	assert.NoError(t, pubErr)
	assert.Equal(t, total, len(messages))
	for i, raw := range messages {
		evt := struct{ Message int }{}
		err = json.Unmarshal(raw, &evt)
		assert.NoError(t, err)
		assert.Equal(t, i+1, evt.Message)
	}
}

func getEnv(name, or string) string {
	if x := os.Getenv(name); x != "" {
		return x
	}
	return or
}

func getRedisAddr() string {
	return fmt.Sprintf("%v:%v",
		getEnv("REDIS_HOST", RedisDefaultHost),
		getEnv("REDIS_PORT", RedisDefaultPort))
}

func getSRedisAddr() string {
	return fmt.Sprintf("%v:%v",
		getEnv("SREDIS_HOST", SRedisDefaultHost),
		getEnv("SREDIS_PORT", SRedisDefaultPort))
}

func newRedisTestingOutput(t *testing.T, cfg map[string]interface{}) *redisOut {
	params := struct {
		Expire int `config:"topology_expire"`
	}{15}

	config, err := common.NewConfigFrom(cfg)
	if err != nil {
		t.Fatalf("Error reading config: %v", err)
	}

	plugin := outputs.FindOutputPlugin("redis")
	if plugin == nil {
		t.Fatalf("redis output module not registered")
	}

	if err := config.Unpack(&params); err != nil {
		t.Fatalf("Failed to unpack topology_expire: %v", err)
	}

	out, err := plugin("libbeat", config, params.Expire)
	if err != nil {
		t.Fatalf("Failed to initialize redis output: %v", err)
	}

	return out.(*redisOut)
}

func sendTestEvents(out *redisOut, batches, N int) error {
	i := 1
	for b := 0; b < batches; b++ {
		batch := make([]outputs.Data, N)
		for n := range batch {
			batch[n] = outputs.Data{Event: common.MapStr{"message": i}}
			i++
		}

		err := out.BulkPublish(nil, outputs.Options{}, batch[:])
		if err != nil {
			return err
		}
	}

	return nil
}
