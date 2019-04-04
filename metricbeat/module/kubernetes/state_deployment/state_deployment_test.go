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

// +build !integration

package state_deployment

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

const testFile = "../_meta/test/kube-state-metrics"

func TestEventMapping(t *testing.T) {
	file, err := os.Open(testFile)
	assert.NoError(t, err, "cannot open test file "+testFile)

	body, err := ioutil.ReadAll(file)
	assert.NoError(t, err, "cannot read test file "+testFile)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/plain; charset=ISO-8859-1")
		w.Write([]byte(body))
	}))

	server.Start()
	defer server.Close()

	config := map[string]interface{}{
		"module":     "kubernetes",
		"metricsets": []string{"state_deployment"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewReportingMetricSetV2(t, config)
	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)

	assert.Equal(t, 5, len(events), "Wrong number of returned events")

	testCases := testCases()
	for _, event := range events {
		metricsetFields := event.MetricSetFields
		name, err := metricsetFields.GetValue("name")
		if err == nil {
			eventKey := event.ModuleFields["namespace"].(string) + "@" + name.(string)
			oneTestCase, oneTestCaseFound := testCases[eventKey]
			if oneTestCaseFound {
				for k, v := range oneTestCase {
					testValue(eventKey, t, metricsetFields, k, v)
				}
				delete(testCases, eventKey)
			}
		}
	}

	if len(testCases) > 0 {
		t.Errorf("Test reference events not found: %v, \n\ngot: %v", testCases, events)
	}
}

func testValue(eventKey string, t *testing.T, event common.MapStr, field string, expected interface{}) {
	data, err := event.GetValue(field)
	assert.NoError(t, err, eventKey+": Could not read field "+field)
	assert.EqualValues(t, expected, data, eventKey+": Wrong value for field "+field)
}

// Test cases built to match 3 examples in 'module/kubernetes/_meta/test/kube-state-metrics'.
// In particular, test same named deployments in different namespaces
func testCases() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"default@jumpy-owl-redis": {
			"name":   "jumpy-owl-redis",
			"paused": false,

			"replicas.available":   0,
			"replicas.desired":     1,
			"replicas.unavailable": 1,
			"replicas.updated":     1,
		},
		"test@jumpy-owl-redis": {
			"name":   "jumpy-owl-redis",
			"paused": true,

			"replicas.available":   6,
			"replicas.desired":     2,
			"replicas.unavailable": 7,
			"replicas.updated":     8,
		},
		"kube-system@tiller-deploy": {
			"name":   "tiller-deploy",
			"paused": false,

			"replicas.available":   1,
			"replicas.desired":     1,
			"replicas.unavailable": 0,
			"replicas.updated":     1,
		},
		"jenkins@wise-lynx-jenkins": {
			"name":   "wise-lynx-jenkins",
			"paused": false,

			"replicas.available":   1,
			"replicas.desired":     1,
			"replicas.unavailable": 0,
			"replicas.updated":     1,
		},
		"kube-system@kube-state-metrics": {
			"name":   "kube-state-metrics",
			"paused": false,

			"replicas.available":   1,
			"replicas.desired":     2,
			"replicas.unavailable": 1,
			"replicas.updated":     2,
		},
	}
}
