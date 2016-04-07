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
)

func TestTopologyInRedis(t *testing.T) {
	index := "test_topo"
	db := 1

	tests := []struct {
		out  *redisOut
		name string
		ips  []string
	}{
		{nil, "proxy1", []string{"10.1.0.4"}},
		{nil, "proxy2", []string{"10.1.0.9", "fe80::4e8d:79ff:fef2:de6a"}},
		{nil, "proxy3", []string{"10.1.0.10"}},
	}

	redisHosts := []string{getRedisAddr()}
	redisConfig := map[string]interface{}{
		"hosts":         redisHosts,
		"index":         index,
		"host_topology": redisHosts[0],
		"db_topology":   db,
		"timeout":       "5s",
	}

	// prepare redis
	{
		conn, err := redis.Dial("tcp", redisHosts[0], redis.DialDatabase(db))
		if err != nil {
			t.Fatalf("redis.Dial failed %v", err)
		}
		// delete old key if present
		defer conn.Close()
		conn.Do("DEL", index)
	}

	// 1. connect
	for i := range tests {
		tests[i].out = newRedisTestingOutput(t, redisConfig)
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

func TestPublishList(t *testing.T) {
	index := "test_publist"
	batches := 100
	batchSize := 1000
	total := batches & batchSize
	db := 0
	redisHosts := []string{getRedisAddr()}

	redisConfig := map[string]interface{}{
		"hosts":    redisHosts,
		"index":    index,
		"db":       db,
		"datatype": "list",
		"timeout":  "5s",
	}

	conn, err := redis.Dial("tcp", redisHosts[0], redis.DialDatabase(db))
	if err != nil {
		t.Fatalf("redis.Dial failed %v", err)
	}

	// delete old key if present
	defer conn.Close()
	conn.Do("DEL", index)

	out := newRedisTestingOutput(t, redisConfig)
	err = sendTestEvents(out, batches, batchSize)
	assert.NoError(t, err)

	results := make([][]byte, total)
	for i := range results {
		results[i], err = redis.Bytes(conn.Do("LPOP", index))
		assert.NoError(t, err)
	}

	for i, raw := range results {
		evt := struct{ Message int }{}
		err = json.Unmarshal(raw, &evt)
		assert.NoError(t, err)
		assert.Equal(t, i+1, evt.Message)
	}
}

func TestPublishChannel(t *testing.T) {
	index := "test_pubchan"
	batches := 100
	batchSize := 1000
	total := batches & batchSize
	db := 0
	redisHosts := []string{getRedisAddr()}

	redisConfig := map[string]interface{}{
		"hosts":    redisHosts,
		"index":    index,
		"db":       db,
		"datatype": "channel",
		"timeout":  "5s",
	}

	conn, err := redis.Dial("tcp", redisHosts[0], redis.DialDatabase(db))
	if err != nil {
		t.Fatalf("redis.Dial failed %v", err)
	}

	// delete old key if present
	defer conn.Close()
	conn.Do("DEL", index)

	// subscribe to packetbeat channel
	psc := redis.PubSubConn{conn}
	if err := psc.Subscribe(index); err != nil {
		t.Fatal(err)
	}
	defer psc.Unsubscribe(index)

	// connect and publish events
	var wg sync.WaitGroup
	var pubErr error
	out := newRedisTestingOutput(t, redisConfig)
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

	out, err := plugin(config, params.Expire)
	if err != nil {
		t.Fatalf("Failed to initialize redis output: %v", err)
	}

	return out.(*redisOut)
}

func sendTestEvents(out *redisOut, batches, N int) error {
	i := 1
	for b := 0; b < batches; b++ {
		batch := make([]common.MapStr, N)
		for n := range batch {
			batch[n] = common.MapStr{"message": i}
			i++
		}

		err := out.BulkPublish(nil, outputs.Options{}, batch[:])
		if err != nil {
			return err
		}
	}

	return nil
}
