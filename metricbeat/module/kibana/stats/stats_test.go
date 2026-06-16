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

package stats

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/kibana/mtest"
)

func TestFetchExcludeUsage(t *testing.T) {
	// Spin up mock Kibana server
	numStatsRequests := 0
	kib := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/status":
			_, err := w.Write([]byte("{ \"version\": { \"number\": \"8.2.0\" }}"))
			if err != nil {
				t.Fatal("write", err)
			}

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

	config := mtest.GetConfig("stats", kib.URL)

	f := mbtest.NewReportingMetricSetV2Error(t, config)

	// First fetch
	mbtest.ReportingFetchV2Error(f)

	// Second fetch
	mbtest.ReportingFetchV2Error(f)

	// Third fetch
	mbtest.ReportingFetchV2Error(f)
}

func TestGetVersionFrom503Status(t *testing.T) {
	// Kibana returns the full status body (including version) even on 503.
	// GetVersion should parse the version from a 503 response.
	const statusBody = `{
		"name": "kibana",
		"uuid": "test-uuid",
		"version": {
			"number": "8.16.0",
			"build_hash": "abc123",
			"build_number": 1234,
			"build_snapshot": false,
			"build_flavor": "traditional",
			"build_date": "2024-07-16T00:00:00.000Z"
		},
		"status": {
			"overall": {"level": "unavailable", "summary": "Elasticsearch is not available", "meta": {}},
			"core": {
				"elasticsearch": {"level": "unavailable", "summary": "Unable to connect", "meta": {}},
				"savedObjects": {"level": "unavailable", "summary": "Not available", "meta": {}}
			},
			"plugins": {}
		},
		"metrics": {
			"last_updated": "2024-07-17T09:35:11.129Z",
			"collection_interval_in_millis": 5000,
			"requests": {"total": 0, "disconnects": 0, "statusCodes": {}, "status_codes": {}},
			"concurrent_connections": 0
		}
	}`

	kib := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/status":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(503)
			w.Write([]byte(statusBody))
		case "/api/stats":
			w.WriteHeader(503)
		}
	}))
	defer kib.Close()

	config := mtest.GetConfig("stats", kib.URL)
	f := mbtest.NewReportingMetricSetV2Error(t, config)

	_, errs := mbtest.ReportingFetchV2Error(f)
	require.NotEmpty(t, errs)
	// GetVersion should succeed (parsed version from 503 body).
	// The error should come from fetchStats, not from version detection.
	require.Contains(t, errs[0].Error(), "error trying to get stats data")
}

func TestFetchNoExcludeUsage(t *testing.T) {
	// Spin up mock Kibana server
	kib := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/status":
			_, err := w.Write([]byte("{ \"version\": { \"number\": \"7.0.0\" }}")) // v7.0.0 does not support exclude_usage and should not be sent
			if err != nil {
				t.Fatal("write", err)
			}

		case "/api/stats":
			excludeUsage := r.FormValue("exclude_usage")
			require.Empty(t, excludeUsage) // exclude_usage should not be provided
			w.WriteHeader(200)
		}
	}))
	defer kib.Close()

	config := mtest.GetConfig("stats", kib.URL)

	f := mbtest.NewReportingMetricSetV2Error(t, config)

	// First fetch
	mbtest.ReportingFetchV2Error(f)
}
