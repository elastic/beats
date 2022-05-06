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

package health

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/elastic-agent-libs/mapstr"

	_ "github.com/elastic/beats/v7/metricbeat/module/traefik"
)

func TestFetchEventContents(t *testing.T) {
	mockResponse, err := ioutil.ReadFile("./_meta/test/simple.json")
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "traefik",
		"metricsets": []string{"health"},
		"hosts":      []string{server.URL},
	}

	fetcher := mbtest.NewReportingMetricSetV2Error(t, config)
	reporter := &mbtest.CapturingReporterV2{}

	fetcher.Fetch(reporter)
	assert.Nil(t, reporter.GetErrors(), "Errors while fetching metrics")

	event := reporter.GetEvents()[0]
	fmt.Println(event.MetricSetFields)
	metricSetFields := event.MetricSetFields

	uptime := metricSetFields["uptime"].(mapstr.M)
	assert.EqualValues(t, 64283, uptime["sec"])

	response := metricSetFields["response"].(mapstr.M)
	assert.EqualValues(t, 18, response["count"])

	avgTime := response["avg_time"].(mapstr.M)
	assert.EqualValues(t, 15, avgTime["us"])

	statusCodes := response["status_codes"].(mapstr.M)
	assert.EqualValues(t, 17, statusCodes["200"])
	assert.EqualValues(t, 1, statusCodes["404"])
}

func TestData(t *testing.T) {
	mbtest.TestDataFiles(t, "traefik", "health")
}
