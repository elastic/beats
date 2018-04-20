// +build integration

package index

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "elasticsearch")

	err := createIndex(elasticsearch.GetEnvHost() + ":" + elasticsearch.GetEnvPort())
	assert.NoError(t, err)

	f := mbtest.NewReportingMetricSetV2(t, elasticsearch.GetConfig("index"))
	events, errs := mbtest.ReportingFetchV2(f)

	assert.NotNil(t, events)
	assert.Nil(t, errs)
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), events[0])
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "elasticsearch")

	err := createIndex(elasticsearch.GetEnvHost() + ":" + elasticsearch.GetEnvPort())
	assert.NoError(t, err)

	f := mbtest.NewReportingMetricSetV2(t, elasticsearch.GetConfig("index"))
	err = mbtest.WriteEventsReporterV2(f, t)
	if err != nil {
		t.Fatal("write", err)
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
