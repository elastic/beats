// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package channels

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
	content, err := ioutil.ReadFile("./_meta/test/channels.json")
	assert.NoError(t, err)
	reporter := &mbtest.CapturingReporterV2{}
	err = eventsMapping(content, reporter)
	assert.NoError(t, err)
	const total = 55
	// 55 per-channel events in the sample
	assert.Equal(t, len(reporter.GetEvents()), total)
	// the last one having non-zero bytes
	bytes, _ := reporter.GetEvents()[0].MetricSetFields.GetValue("bytes")
	assert.True(t, bytes.(int64) > 0)
	// check for existence of any non-zero channel / queue depth on known entities
	events := reporter.GetEvents()
	var maxDepth int64
	for _, evt := range events {
		fields := evt.MetricSetFields
		name, nameErr := fields.GetValue("name")
		assert.NoError(t, nameErr)
		depthIfc, depthErr := evt.MetricSetFields.GetValue("depth")
		depth := depthIfc.(int64)
		if depth > maxDepth {
			maxDepth = depth
		}
		assert.NoError(t, depthErr)
		if name == "system.index" {
			assert.Equal(t, depth, int64(1))
		}

	}
	// hacked in ONE queue where depth was exactly one
	// so maxDepth should be 1 as well
	assert.Equal(t, maxDepth, int64(1))
}

func TestFetchEventContent(t *testing.T) {
	server := initServer()
	defer server.Close()

	config := map[string]interface{}{
		"module":     "stan",
		"metricsets": []string{"channels"},
		"hosts":      []string{server.URL},
	}
	reporter := &mbtest.CapturingReporterV2{}

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	metricSet.Fetch(reporter)

	for _, evt := range reporter.GetEvents() {
		mbtest.StandardizeEvent(metricSet, evt)
	}

}

func initServer() *httptest.Server {
	absPath, _ := filepath.Abs("./_meta/test/")

	response, _ := ioutil.ReadFile(absPath + "/channels.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	return server
}
