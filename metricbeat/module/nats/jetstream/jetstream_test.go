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

package jetstream

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestEventMapping(t *testing.T) {
	content, err := os.ReadFile("./_meta/test/example.json")
	assert.NoError(t, err)
	reporter := &mbtest.CapturingReporterV2{}
	config := ModuleConfig{
		Jetstream: MetricsetConfig{
			Stats: StatsConfig{
				Enabled: true,
			},
			Account: AccountConfig{
				Enabled: true,
			},
			Stream: StreamConfig{
				Enabled: true,
			},
			Consumer: ConsumerConfig{
				Enabled: true,
			},
		},
	}
	ms := &MetricSet{
		Config: config.Jetstream,
	}
	err = eventMapping(ms, reporter, content)
	assert.NoError(t, err)
}

func TestFetchEventContent(t *testing.T) {
	absPath, _ := filepath.Abs("./_meta/test")

	response, _ := os.ReadFile(absPath + "/example.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "nats",
		"metricsets": []string{"jetstream"},
		"hosts":      []string{server.URL},
		"jetstream": map[string]interface{}{
			"stats": map[string]interface{}{
				"enabled": true,
			},
			"account": map[string]interface{}{
				"enabled": true,
			},
			"stream": map[string]interface{}{
				"enabled": true,
			},
			"consumer": map[string]interface{}{
				"enabled": true,
			},
		},
	}
	reporter := &mbtest.CapturingReporterV2{}

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	metricSet.Fetch(reporter)

	for _, event := range reporter.GetEvents() {
		e := mbtest.StandardizeEvent(metricSet, event)
		t.Logf("%s/%s event: %+v", metricSet.Module().Name(), metricSet.Name(), e.Fields.StringToPrint())
	}

	errors := reporter.GetErrors()
	assert.Len(t, errors, 0)
}
