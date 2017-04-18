// +build integration

package collector

import (
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	f := mbtest.NewEventsFetcher(t, getConfig())
	events, err := f.Fetch()

	hasTag := false
	doesntHaveTag := false
	for _, event := range events {

		ok, _ := event.HasKey("my_histogram")
		if ok {
			_, err := event.GetValue("tags")
			if err == nil {
				t.Fatal("write", "my_counter not supposed to have tags")
			}
			doesntHaveTag = true
		}

		ok, _ = event.HasKey("my_counter")
		if ok {
			tagsRaw, err := event.GetValue("tags")
			if err != nil {
				t.Fatal("write", err)
			} else {
				tags, ok := tagsRaw.(common.MapStr)
				if !ok {
					t.Fatal("write", "unable to cast tags to common.MapStr")
				} else {
					assert.Equal(t, len(tags), 1)
					hasTag = true
				}
			}
		}
	}
	assert.Equal(t, hasTag, true)
	assert.Equal(t, doesntHaveTag, true)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), events)
}

func TestData(t *testing.T) {
	f := mbtest.NewEventsFetcher(t, getConfig())
	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getEnvHost() string {
	host := os.Getenv("DROPWIZARD_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

func getEnvPort() string {
	port := os.Getenv("DROPWIZARD_PORT")

	if len(port) == 0 {
		port = "8080"
	}
	return port
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":       "dropwizard",
		"metricsets":   []string{"collector"},
		"hosts":        []string{getEnvHost() + ":" + getEnvPort()},
		"namespace":    "testnamespace",
		"metrics_path": "/test/metrics",
		"enabled":      true,
	}
}
