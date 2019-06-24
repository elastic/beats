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

package state_statefulset

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	_ "github.com/elastic/beats/metricbeat/module/kubernetes"
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
		"metricsets": []string{"state_statefulset"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewReportingMetricSetV2(t, config)
	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)

	assert.Equal(t, 3, len(events), "Wrong number of returned events")

	testCases := testCases()
	for _, event := range events {
		metricsetFields := event.MetricSetFields
		name, err := metricsetFields.GetValue("name")
		if err == nil {
			eventKey := event.ModuleFields["namespace"].(string) + "@" + name.(string)
			oneTestCase, oneTestCaseFound := testCases[eventKey]
			if oneTestCaseFound {
				for k, v := range oneTestCase {
					testValue(t, metricsetFields, k, v)
				}
				delete(testCases, eventKey)
			}
		}
	}

	if len(testCases) > 0 {
		t.Errorf("Test reference events not found: %v", testCases)
	}
}

func testValue(t *testing.T, event common.MapStr, field string, expected interface{}) {
	data, err := event.GetValue(field)
	assert.NoError(t, err, "Could not read field "+field)
	assert.EqualValues(t, expected, data, "Wrong value for field "+field)
}

func testCases() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{
		"default@elasticsearch": {
			"name": "elasticsearch",

			"created":             1511973651,
			"replicas.observed":   1,
			"replicas.desired":    4,
			"generation.observed": 1,
			"generation.desired":  3,
		},
		"default@mysql": {
			"name": "mysql",

			"created":             1511989697,
			"replicas.observed":   2,
			"replicas.desired":    5,
			"generation.observed": 2,
			"generation.desired":  4,
		},
		"custom@mysql": {
			"name": "mysql",

			"created":             1511999697,
			"replicas.observed":   3,
			"replicas.desired":    6,
			"generation.observed": 3,
			"generation.desired":  5,
		},
	}
}

func TestData(t *testing.T) {
	mbtest.TestDataFiles(t, "kubernetes", "state_statefulset")
}
