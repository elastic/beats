// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package channels

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/stretchr/testify/assert"
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
	absPath, _ := filepath.Abs("./_meta/test/")

	response, _ := ioutil.ReadFile(absPath + "/channels.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "stan",
		"metricsets": []string{"channels"},
		"hosts":      []string{server.URL},
	}
	reporter := &mbtest.CapturingReporterV2{}

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	metricSet.Fetch(reporter)

	for idx, evt := range reporter.GetEvents() {
		e := mbtest.StandardizeEvent(metricSet, evt)
		t.Logf("[%d] %s/%s event: %+v", idx, metricSet.Module().Name(), metricSet.Name(), e.Fields.StringToPrint())
	}

}
