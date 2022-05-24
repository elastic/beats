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

package leader

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"

	"github.com/stretchr/testify/assert"

	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestEventMapping(t *testing.T) {
	content, err := ioutil.ReadFile("../_meta/test/leaderstats.json")
	assert.NoError(t, err)

	var data Leader
	err = json.Unmarshal(content, &data)
	assert.NoError(t, err)

	event := eventMapping("6e3bd23ae5f1eae0", data, FollowersID{
		Counts: Counts{
			Success: 745,
		},
	})
	assert.NotZero(t, event)

	leader, err := event.MetricSetFields.GetValue("follower.leader")
	assert.NoError(t, err)
	assert.Equal(t, leader, "924e2e83e93f2560")

	followerID, err := event.MetricSetFields.GetValue("follower.id")
	assert.NoError(t, err)
	assert.Equal(t, followerID, "6e3bd23ae5f1eae0")

	successOps, err := event.MetricSetFields.GetValue("follower.success_operations")
	assert.NoError(t, err)
	assert.Equal(t, successOps, int64(745))
}

func TestFetchEventContent(t *testing.T) {

	const (
		module              = "etcd"
		metricset           = "leader"
		mockedFetchLocation = "../_meta/test/"
	)

	var testcases = []struct {
		name            string
		mockedFetchFile string
		httpCode        int

		expectedFetchErrorRegexp string
		expectedNumEvents        int
	}{
		{
			name:              "Leader member stats",
			mockedFetchFile:   "/leaderstats.json",
			httpCode:          http.StatusOK,
			expectedNumEvents: 1,
		},
		{
			name:              "Follower member",
			mockedFetchFile:   "/leaderstats_follower.json",
			httpCode:          http.StatusForbidden,
			expectedNumEvents: 0,
		},
		{
			name:                     "Simulating credentials issue",
			mockedFetchFile:          "/leaderstats_empty.json",
			httpCode:                 http.StatusForbidden,
			expectedFetchErrorRegexp: "fetching HTTP response returned status code 403",
			expectedNumEvents:        0,
		},
		{
			name:                     "Simulating failure message",
			mockedFetchFile:          "/leaderstats_internalerror.json",
			httpCode:                 http.StatusInternalServerError,
			expectedFetchErrorRegexp: "fetching HTTP response returned status code 500:.+",
			expectedNumEvents:        0,
		}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			absPath, err := filepath.Abs(mockedFetchLocation + tc.mockedFetchFile)
			assert.NoError(t, err)

			response, err := ioutil.ReadFile(absPath)
			assert.NoError(t, err)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.httpCode)
				w.Header().Set("Content-Type", "application/json;")
				_, _ = w.Write(response)
			}))
			defer server.Close()

			config := map[string]interface{}{
				"module":     module,
				"metricsets": []string{metricset},
				"hosts":      []string{server.URL},
			}

			f := mbtest.NewReportingMetricSetV2Error(t, config)
			events, errs := mbtest.ReportingFetchV2Error(f)

			if tc.expectedFetchErrorRegexp != "" {
				for _, err := range errs {
					if match, _ := regexp.MatchString(tc.expectedFetchErrorRegexp, err.Error()); match {
						// found expected fetch error, no need for further checks
						return
					}
				}
				t.Fatalf("Expected fetch error not found:\n Expected:%s\n Got: %+v",
					tc.expectedFetchErrorRegexp,
					errs)
			}

			if len(errs) > 0 {
				t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
			}

			assert.Equal(t, tc.expectedNumEvents, len(events))

			for i := range events {
				t.Logf("%s/%s event[%d]: %+v", f.Module().Name(), f.Name(), i, events[i])
			}
		})
	}
}
