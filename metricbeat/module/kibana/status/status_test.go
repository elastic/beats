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

package status

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/kibana/mtest"

	_ "github.com/elastic/beats/v7/metricbeat/module/kibana"
)

func TestData(t *testing.T) {
	mbtest.TestDataFiles(t, "kibana", "status")
}

func TestFetch503ReadsBody(t *testing.T) {
	// Matches the actual Kibana v8 GET /api/status response shape when
	// coreOverall.level >= unavailable (which triggers a 503).
	const statusBody = `{
		"name": "kibana",
		"uuid": "5b2de169-2785-441b-ae8c-186a1936b17d",
		"version": {
			"number": "8.16.0",
			"build_hash": "abc123",
			"build_number": 1234,
			"build_snapshot": false,
			"build_flavor": "traditional",
			"build_date": "2024-07-16T00:00:00.000Z"
		},
		"status": {
			"overall": {
				"level": "unavailable",
				"summary": "Elasticsearch is not available",
				"meta": {}
			},
			"core": {
				"elasticsearch": {
					"level": "unavailable",
					"summary": "Unable to connect to Elasticsearch",
					"meta": {}
				},
				"savedObjects": {
					"level": "unavailable",
					"summary": "SavedObjects service is not available without a healthy Elasticsearch connection",
					"meta": {}
				}
			},
			"plugins": {}
		},
		"metrics": {
			"last_updated": "2024-07-17T09:35:11.129Z",
			"collection_interval_in_millis": 5000,
			"requests": {"total": 10, "disconnects": 2, "statusCodes": {}, "status_codes": {}},
			"concurrent_connections": 1
		}
	}`

	kib := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(503)
		w.Write([]byte(statusBody))
	}))
	defer kib.Close()

	config := mtest.GetConfig("status", kib.URL)
	f := mbtest.NewReportingMetricSetV2Error(t, config)

	events, errs := mbtest.ReportingFetchV2Error(f)
	require.Empty(t, errs)
	require.Len(t, events, 1)

	event := events[0]

	overallLevel, err := event.MetricSetFields.GetValue("status.overall.level")
	require.NoError(t, err)
	require.Equal(t, "unavailable", overallLevel)

	overallSummary, err := event.MetricSetFields.GetValue("status.overall.summary")
	require.NoError(t, err)
	require.Equal(t, "Elasticsearch is not available", overallSummary)

	esLevel, err := event.MetricSetFields.GetValue("status.core.elasticsearch.level")
	require.NoError(t, err)
	require.Equal(t, "unavailable", esLevel)

	soLevel, err := event.MetricSetFields.GetValue("status.core.savedObjects.level")
	require.NoError(t, err)
	require.Equal(t, "unavailable", soLevel)

	serviceID, err := event.RootFields.GetValue("service.id")
	require.NoError(t, err)
	require.Equal(t, "5b2de169-2785-441b-ae8c-186a1936b17d", serviceID)

	serviceVersion, err := event.RootFields.GetValue("service.version")
	require.NoError(t, err)
	require.Equal(t, "8.16.0", serviceVersion)
}

func TestFetch503InvalidBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"invalid json", "not json"},
		{"html error page", "<html><body>Service Unavailable</body></html>"},
		{"missing uuid", `{"name":"kibana","version":{"number":"8.16.0"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kib := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(503)
				w.Write([]byte(tt.body))
			}))
			defer kib.Close()

			config := mtest.GetConfig("status", kib.URL)
			f := mbtest.NewReportingMetricSetV2Error(t, config)

			events, errs := mbtest.ReportingFetchV2Error(f)
			require.Empty(t, events)
			require.NotEmpty(t, errs)
		})
	}
}
