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

package ptest

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"

	"github.com/mitchellh/hashstructure"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/metricbeat/mb"
	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"
	"github.com/elastic/beats/v8/metricbeat/mb/testing/flags"
)

// TestCases holds the list of test cases to test a metricset
type TestCases []struct {
	// MetricsFile containing Prometheus outputted metrics
	MetricsFile string

	// ExpectedFile containing resulting documents
	ExpectedFile string
}

// TestMetricSet goes over the given TestCases and ensures that source Prometheus metrics gets converted into the expected
// events when passed by the given metricset.
// If -data flag is passed, the expected JSON file will be updated with the result
func TestMetricSet(t *testing.T, module, metricset string, cases TestCases) {
	for _, test := range cases {
		t.Logf("Testing %s file\n", test.MetricsFile)

		file, err := os.Open(test.MetricsFile)
		assert.NoError(t, err, "cannot open test file "+test.MetricsFile)

		body, err := ioutil.ReadAll(file)
		assert.NoError(t, err, "cannot read test file "+test.MetricsFile)

		server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "text/plain; charset=ISO-8859-1")
			w.Write([]byte(body))
		}))

		server.Start()
		defer server.Close()

		config := map[string]interface{}{
			"module":     module,
			"metricsets": []string{metricset},
			"hosts":      []string{server.URL},
		}

		f := mbtest.NewFetcher(t, config)
		events, errs := f.FetchEvents()
		assert.Nil(t, errs, "Errors while fetching metrics")

		if *flags.DataFlag {
			sort.SliceStable(events, func(i, j int) bool {
				h1, _ := hashstructure.Hash(events[i], nil)
				h2, _ := hashstructure.Hash(events[j], nil)
				return h1 < h2
			})
			eventsJSON, _ := json.MarshalIndent(events, "", "\t")
			err = ioutil.WriteFile(test.ExpectedFile, eventsJSON, 0644)
			assert.NoError(t, err)
		}

		// Read expected events from reference file
		expected, err := ioutil.ReadFile(test.ExpectedFile)
		if err != nil {
			t.Fatal(err)
		}

		var expectedEvents []mb.Event
		err = json.Unmarshal(expected, &expectedEvents)
		if err != nil {
			t.Fatal(err)
		}

		for _, event := range events {
			// ensure the event is in expected list
			found := -1
			for i, expectedEvent := range expectedEvents {
				if event.RootFields.String() == expectedEvent.RootFields.String() &&
					event.ModuleFields.String() == expectedEvent.ModuleFields.String() &&
					event.MetricSetFields.String() == expectedEvent.MetricSetFields.String() {
					found = i
					break
				}
			}
			if found > -1 {
				expectedEvents = append(expectedEvents[:found], expectedEvents[found+1:]...)
			} else {
				t.Errorf("Event was not expected: %+v", event)
			}
		}

		if len(expectedEvents) > 0 {
			t.Error("Some events were missing:")
			for _, e := range expectedEvents {
				t.Error(e)
			}
			t.Fatal()
		}
	}
}
