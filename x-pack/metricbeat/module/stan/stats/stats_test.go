// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stats

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
)

func TestEventMapping(t *testing.T) {
	content, err := ioutil.ReadFile("./_meta/test/serversz.json")
	assert.NoError(t, err)
	reporter := &mbtest.CapturingReporterV2{}
	err = eventMapping(content, reporter)
	assert.NoError(t, err)
	event := reporter.GetEvents()[0]
	d, _ := event.MetricSetFields.GetValue("channels")
	assert.Equal(t, d, int64(55))
}

func TestFetchEventContent(t *testing.T) {
	server := initServer()
	defer server.Close()

	config := map[string]interface{}{
		"module":     "stan",
		"metricsets": []string{"stats"},
		"hosts":      []string{server.URL},
	}
	reporter := &mbtest.CapturingReporterV2{}

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	metricSet.Fetch(reporter)

	events := reporter.GetEvents()
	e := mbtest.StandardizeEvent(metricSet, events[0])
	t.Logf("%s/%s event: %+v", metricSet.Module().Name(), metricSet.Name(), e.Fields.StringToPrint())
}

func initServer() *httptest.Server {
	absPath, _ := filepath.Abs("./_meta/test/")

	response, _ := ioutil.ReadFile(absPath + "/serversz.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	return server
}
