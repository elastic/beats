// +build integration

package status

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	compose.EnsureUpWithTimeout(t, 600, "elasticsearch", "kibana")

	f := mbtest.NewEventFetcher(t, getConfig())
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "elasticsearch", "kibana")

	f := mbtest.NewEventFetcher(t, getConfig())
	err := mbtest.WriteEvent(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "kibana",
		"metricsets": []string{"status"},
		"hosts":      []string{GetEnvHost() + ":" + GetEnvPort()},
	}
}

func GetEnvHost() string {
	host := os.Getenv("KIBANA_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

func GetEnvPort() string {
	port := os.Getenv("KIBANA_PORT")

	if len(port) == 0 {
		port = "5601"
	}
	return port
}
