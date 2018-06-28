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

package health

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

// raw response copied from Traefik instance's health API endpoint
const response = `{
	"pid": 1,
	"uptime": "17h51m23.252891567s",
	"uptime_sec": 64283.252891567,
	"time": "2018-06-27 22:07:28.966768969 +0000 UTC m=+64283.314491879",
	"unixtime": 1530137248,
	"status_code_count": {},
	"total_status_code_count": {
		"200": 17,
	  	"404": 1
	},
	"count": 0,
	"total_count": 18,
	"total_response_time": "272.119µs",
	"total_response_time_sec": 0.000272119,
	"average_response_time": "15.117µs",
	"average_response_time_sec": 1.5117e-05
}
`

func TestFetchEventContents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "traefik",
		"metricsets": []string{"health"},
		"hosts":      []string{server.URL},
	}

	fetcher := mbtest.NewReportingMetricSetV2(t, config)
	reporter := &mbtest.CapturingReporterV2{}

	fetcher.Fetch(reporter)
	assert.Nil(t, reporter.GetErrors(), "Errors while fetching metrics")

	event := reporter.GetEvents()[0]
	fmt.Println(event.MetricSetFields)
	metricSetFields := event.MetricSetFields

	uptime := metricSetFields["uptime"].(common.MapStr)
	assert.EqualValues(t, 64283, uptime["sec"])

	response := metricSetFields["response"].(common.MapStr)
	assert.EqualValues(t, 18, response["count"])

	avgTime := response["avg_time"].(common.MapStr)
	assert.EqualValues(t, 1.5117e-05, avgTime["sec"])

	statusCodes := response["status_codes"].(common.MapStr)
	assert.EqualValues(t, 17, statusCodes["200"])
	assert.EqualValues(t, 1, statusCodes["404"])
}
