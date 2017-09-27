// +build integration

package collector

import (
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

// These tests are running with prometheus metrics as an example as this container is already available
// Every prometheus exporter should work here.

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "prometheus")

	f := mbtest.NewEventsFetcher(t, getConfig())
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "prometheus")

	f := mbtest.NewEventsFetcher(t, getConfig())

	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "prometheus",
		"metricsets": []string{"collector"},
		"hosts":      []string{getPrometheusEnvHost() + ":" + getPrometheusEnvPort()},
		"namespace":  "collector",
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
