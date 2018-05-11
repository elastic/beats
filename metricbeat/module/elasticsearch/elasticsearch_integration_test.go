// +build integration

package elasticsearch_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	_ "github.com/elastic/beats/metricbeat/module/elasticsearch/index"
	_ "github.com/elastic/beats/metricbeat/module/elasticsearch/index_summary"
	_ "github.com/elastic/beats/metricbeat/module/elasticsearch/node"
	_ "github.com/elastic/beats/metricbeat/module/elasticsearch/node_stats"
)

var metricSets = []string{
	"index",
	"index_summary",
	"node",
	"node_stats",
}

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "elasticsearch")

	err := createIndex(getEnvHost() + ":" + getEnvPort())
	assert.NoError(t, err)

	for _, metricSet := range metricSets {
		t.Run(metricSet, func(t *testing.T) {
			f := mbtest.NewReportingMetricSetV2(t, getConfig(metricSet))
			events, errs := mbtest.ReportingFetchV2(f)

			assert.NotNil(t, events)
			assert.Nil(t, errs)
			t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), events[0].BeatEvent("elasticsearch", metricSet).Fields.StringToPrint())
		})
	}
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "elasticsearch")

	for _, metricSet := range metricSets {
		t.Run(metricSet, func(t *testing.T) {
			f := mbtest.NewReportingMetricSetV2(t, getConfig(metricSet))
			err := mbtest.WriteEventsReporterV2(f, t, metricSet)
			if err != nil {
				t.Fatal("write", err)
			}
		})
	}
}

// GetEnvHost returns host for Elasticsearch
func getEnvHost() string {
	host := os.Getenv("ES_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

// GetEnvPort returns port for Elasticsearch
func getEnvPort() string {
	port := os.Getenv("ES_PORT")

	if len(port) == 0 {
		port = "9200"
	}
	return port
}

// GetConfig returns config for elasticsearch module
func getConfig(metricset string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "elasticsearch",
		"metricsets": []string{metricset},
		"hosts":      []string{getEnvHost() + ":" + getEnvPort()},
	}
}

// createIndex creates and elasticsearch index in case it does not exit yet
func createIndex(host string) error {
	client := &http.Client{}

	resp, err := http.Get("http://" + host + "/testindex")
	if err != nil {
		return err
	}
	resp.Body.Close()

	// This means index already exists
	if resp.StatusCode == 200 {
		return nil
	}

	req, err := http.NewRequest("PUT", "http://"+host+"/testindex", nil)
	if err != nil {
		return err
	}

	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	return nil
}
