// +build integration

package redis

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/garyburd/redigo/redis"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/outest"

	_ "github.com/elastic/beats/libbeat/outputs/codec/format"
	_ "github.com/elastic/beats/libbeat/outputs/codec/json"
)

const (
	RedisDefaultHost = "localhost"
	RedisDefaultPort = "6379"

	SRedisDefaultHost = "localhost"
	SRedisDefaultPort = "6380"
)

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

func TestPublishChannelTCPWithFormatting(t *testing.T) {
	t.Skip("format string not yet supported")
	db := 0
	key := "test_pubchan_tcp"
	redisConfig := map[string]interface{}{
		"hosts":               []string{getRedisAddr()},
		"key":                 key,
		"db":                  db,
		"datatype":            "channel",
		"timeout":             "5s",
		"codec.format.string": "%{[message]}",
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
		if _, hasFmt := cfg["codec.format.string"]; hasFmt {
			t.Fatal("format string not yet supported")
			/*
				fmtString := fmtstr.MustCompileEvent(fmt.(string))
				expectedMessage, _ := fmtString.Run(createEvent(i + 1))
				assert.NoError(t, err)
				assert.Equal(t, string(expectedMessage), string(raw))
			*/
		} else {
			err = json.Unmarshal(raw, &evt)
			assert.NoError(t, err)
			assert.Equal(t, i+1, evt.Message)
		}
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

func newRedisTestingOutput(t *testing.T, cfg map[string]interface{}) *client {
	config, err := common.NewConfigFrom(cfg)
	if err != nil {
		t.Fatalf("Error reading config: %v", err)
	}

	plugin := outputs.FindFactory("redis")
	if plugin == nil {
		t.Fatalf("redis output module not registered")
	}

	out, err := plugin(beat.Info{Beat: "libbeat"}, outputs.NewNilObserver(), config)
	if err != nil {
		t.Fatalf("Failed to initialize redis output: %v", err)
	}

	client := out.Clients[0].(*client)
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect to redis host: %v", err)
	}

	return client
}

func sendTestEvents(out *client, batches, N int) error {
	i := 1
	for b := 0; b < batches; b++ {
		events := make([]beat.Event, N)
		for n := range events {
			events[n] = createEvent(i)
			i++
		}

		batch := outest.NewBatch(events...)
		err := out.Publish(batch)
		if err != nil {
			return err
		}
	}

	return nil
}

func createEvent(message int) beat.Event {
	return beat.Event{
		Timestamp: time.Now(),
		Fields:    common.MapStr{"message": message},
	}
}
