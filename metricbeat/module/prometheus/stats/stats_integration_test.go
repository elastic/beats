// +build integration

package stats

import (
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig())
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check number of fields.
	assert.Equal(t, 3, len(event))
	assert.True(t, event["processes"].(common.MapStr)["open_fds"].(int64) > 0)
}

func TestData(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig())

	err := mbtest.WriteEvent(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "prometheus",
		"metricsets": []string{"stats"},
		"hosts":      []string{getPrometheusEnvHost() + ":" + getPrometheusEnvPort()},
	}
}

func getPrometheusEnvHost() string {
	host := os.Getenv("PROMETHEUS_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

func getPrometheusEnvPort() string {
	port := os.Getenv("PROMETHEUS_PORT")

	if len(port) == 0 {
		port = "9090"
	}
	return port
}
