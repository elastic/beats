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

package stats

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/mitchellh/hashstructure"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/kibana/mtest"
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
				w.Write([]byte("{ \"version\": { \"number\": \"7.5.0\" }}"))

			case 2: // third call
				require.Equal(t, "true", excludeUsage) // exclude_usage is always true
				w.WriteHeader(200)
			}

			numStatsRequests++
		}
	}))
	defer kib.Close()

	config := mtest.GetConfig("stats", kib.URL, true)

	f := mbtest.NewReportingMetricSetV2Error(t, config)

	// First fetch
	mbtest.ReportingFetchV2Error(f)

	// Second fetch
	mbtest.ReportingFetchV2Error(f)

	// Third fetch
	mbtest.ReportingFetchV2Error(f)
}

func TestFetchNoExcludeUsage(t *testing.T) {
	// Spin up mock Kibana server
	kib := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/status":
			w.Write([]byte("{ \"version\": { \"number\": \"7.0.0\" }}")) // v7.0.0 does not support exclude_usage and should not be sent

		case "/api/stats":
			excludeUsage := r.FormValue("exclude_usage")
			require.Empty(t, excludeUsage) // exclude_usage should not be provided
			w.WriteHeader(200)
		}
	}))
	defer kib.Close()

	config := mtest.GetConfig("stats", kib.URL, true)

	f := mbtest.NewReportingMetricSetV2Error(t, config)

	// First fetch
	mbtest.ReportingFetchV2Error(f)
}

func kibanaServer(t *testing.T, path string, version string) *httptest.Server {
	body, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read file: %s", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := ""
		v := r.URL.Query()
		if len(v) > 0 {
			query += "?" + v.Encode()
		}

		switch r.URL.Path+query {
		case "/api/status":
			response := fmt.Sprintf("{ \"version\": { \"number\": \"%s\" }}", version)
			w.Write([]byte(response))
		case "/api/stats?exclude_usage=true&extended=true":
			w.WriteHeader(200)
			w.Write(body)
		default:
			w.WriteHeader(404)
		}
	}))
	return server
}

func TestMultiProcess(t *testing.T) {
	kib := kibanaServer(t, "_meta/testdata/multi-process.json", "8.1.0")
	defer kib.Close()

	config := mtest.GetConfig("stats", kib.URL, true)

	f := mbtest.NewReportingMetricSetV2Error(t, config)

	events, errors := mbtest.ReportingFetchV2Error(f)
	fmt.Printf("%+v\n", events)
	fmt.Printf("%+v\n", errors)
	if (len(errors) > 0) {
		t.Errorf("unexpected error: %v", errors[0])
	}

	var data []common.MapStr

	for _, e := range events {
		beatEvent := mbtest.StandardizeEvent(f, e, mb.AddMetricSetInfo)
		// Overwrite service.address as the port changes every time
		beatEvent.Fields.Put("service.address", "127.0.0.1:55555")
		data = append(data, beatEvent.Fields)
	}

	// Sorting the events is necessary as events are not necessarily sent in the same order
	sort.SliceStable(data, func(i, j int) bool {
		h1, _ := hashstructure.Hash(data[i], nil)
		h2, _ := hashstructure.Hash(data[j], nil)
		return h1 < h2
	})

	fmt.Printf("data: %+v\n", data)
}

func TestData(t *testing.T) {
	mbtest.TestDataFiles(t, "kibana", "stats")
}
