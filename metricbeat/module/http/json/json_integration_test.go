// +build integration

package json

import (
	"os"
	"testing"

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
		"module":     "http",
		"metricsets": []string{"json"},
		"hosts":      []string{getEnvHost() + ":" + getEnvPort()},
		"path":       "/",
		"namespace":  "testnamespace",
	}
}

func getEnvHost() string {
	host := os.Getenv("HTTP_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

func getEnvPort() string {
	port := os.Getenv("HTTP_PORT")

	if len(port) == 0 {
		port = "8080"
	}
	return port
}
