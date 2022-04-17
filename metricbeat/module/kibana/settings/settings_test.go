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

//go:build !integration
// +build !integration

package settings

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
	"github.com/menderesk/beats/v7/metricbeat/module/kibana/mtest"
)

func TestFetchExcludeUsage(t *testing.T) {
	// Spin up mock Kibana server
	numStatsRequests := 0
	kib := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/status":
			w.Write([]byte("{ \"version\": { \"number\": \"7.5.0\" }}"))

		case "/api/stats":
			excludeUsage := r.FormValue("exclude_usage")

			// Make GET /api/stats return 503 for first call, 200 for subsequent calls
			switch numStatsRequests {
			case 0: // first call
				require.Equal(t, "true", excludeUsage) // exclude_usage is always true
				w.WriteHeader(503)

			case 1: // second call
				require.Equal(t, "true", excludeUsage) // exclude_usage is always true
				w.WriteHeader(200)

			case 2: // third call
				require.Equal(t, "true", excludeUsage) // exclude_usage is always true
				w.WriteHeader(200)
			}

			numStatsRequests++
		}
	}))
	defer kib.Close()

	config := mtest.GetConfig("settings", kib.URL, false)

	f := mbtest.NewReportingMetricSetV2Error(t, config)

	// First fetch
	mbtest.ReportingFetchV2Error(f)

	// Second fetch
	mbtest.ReportingFetchV2Error(f)

	// Third fetch
	mbtest.ReportingFetchV2Error(f)
}
