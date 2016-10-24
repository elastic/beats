// +build integration

package keyspace

import (
	"strings"
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/redis"

	rd "github.com/garyburd/redigo/redis"
	"github.com/stretchr/testify/assert"
)

var host = redis.GetRedisEnvHost() + ":" + redis.GetRedisEnvPort()

func TestFetch(t *testing.T) {

	addEntry(t)

	// Fetch data
	f := mbtest.NewEventsFetcher(t, getConfig())
	events, err := f.Fetch()
	if err != nil {
		t.Fatal("fetch", err)
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), events)

	// Make sure at least 1 db keyspace exists
	assert.True(t, len(events) > 0)

	keyspace := events[0]

	assert.True(t, keyspace["avg_ttl"].(int64) >= 0)
	assert.True(t, keyspace["expires"].(int64) >= 0)
	assert.True(t, keyspace["keys"].(int64) >= 0)
	assert.True(t, strings.Contains(keyspace["id"].(string), "db"))
}

func TestData(t *testing.T) {
	addEntry(t)

	f := mbtest.NewEventsFetcher(t, getConfig())

	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

// addEntry adds an entry to redis
func addEntry(t *testing.T) {
	// Insert at least one event to make sure db exists
	c, err := rd.Dial("tcp", host)
	if err != nil {
		t.Fatal("connect", err)
	}
	defer c.Close()
	_, err = c.Do("SET", "foo", "bar")
	if err != nil {
		t.Fatal("SET", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "redis",
		"metricsets": []string{"keyspace"},
		"hosts":      []string{host},
	}
}
