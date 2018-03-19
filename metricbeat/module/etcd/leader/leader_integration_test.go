// +build integration

package leader

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	compose.EnsureUp(t, "etcd")

	f := mbtest.NewEventFetcher(t, getConfig())
	err := mbtest.WriteEvent(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "etcd")

	f := mbtest.NewEventFetcher(t, getConfig())
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	assert.NotNil(t, event)
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "etcd",
		"metricsets": []string{"leader"},
		"hosts":      []string{GetEnvHost() + ":" + GetEnvPort()},
	}
}

func GetEnvHost() string {
	host := os.Getenv("ETCD_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

func GetEnvPort() string {
	port := os.Getenv("ETCD_PORT")

	if len(port) == 0 {
		port = "2379"
	}
	return port
}
