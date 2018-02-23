// +build integration

package jmx

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "jolokia")

	for _, config := range getConfigs() {
		f := mbtest.NewEventFetcher(t, config)
		event, err := f.Fetch()
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
		if len(event) <= 1 {
			t.Fatal("Empty event")
		}
	}
}

func TestData(t *testing.T) {
	for _, config := range getConfigs() {
		f := mbtest.NewEventFetcher(t, config)
		err := mbtest.WriteEvent(f, t)
		if err != nil {
			t.Fatal("write", err)
		}
	}
}

func getConfigs() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"module":     "jolokia",
			"metricsets": []string{"jmx"},
			"hosts":      []string{getEnvHost() + ":" + getEnvPort()},
			"namespace":  "testnamespace",
			"jmx.mappings": []map[string]interface{}{
				{
					"mbean": "java.lang:type=Runtime",
					"attributes": []map[string]string{
						{
							"attr":  "Uptime",
							"field": "uptime",
						},
					},
				},
				{
					"mbean": "java.lang:type=GarbageCollector,name=ConcurrentMarkSweep",
					"attributes": []map[string]string{
						{
							"attr":  "CollectionTime",
							"field": "gc.cms_collection_time",
						},
						{
							"attr":  "CollectionCount",
							"field": "gc.cms_collection_count",
						},
					},
				},
				{
					"mbean": "java.lang:type=Memory",
					"attributes": []map[string]string{
						{
							"attr":  "HeapMemoryUsage",
							"field": "memory.heap_usage",
						},
						{
							"attr":  "NonHeapMemoryUsage",
							"field": "memory.non_heap_usage",
						},
					},
				},
			},
		},
		{
			"module":     "jolokia",
			"metricsets": []string{"jmx"},
			"hosts":      []string{getEnvHost() + ":" + getEnvPort()},
			"namespace":  "testnamespace",
			"jmx.mappings": []map[string]interface{}{
				{
					"mbean": "Catalina:name=*,type=ThreadPool",
					"attributes": []map[string]string{
						{
							"attr":  "maxConnections",
							"field": "max_connections",
						},
						{
							"attr":  "port",
							"field": "port",
						},
					},
				},
			},
		},
	}
}

func getEnvHost() string {
	host := os.Getenv("JOLOKIA_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

func getEnvPort() string {
	port := os.Getenv("JOLOKIA_PORT")

	if len(port) == 0 {
		port = "8778"
	}
	return port
}
