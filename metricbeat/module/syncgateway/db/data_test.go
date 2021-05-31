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

package db

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

type mockReporter struct{}

func (m mockReporter) Event(event mb.Event) bool {
	fmt.Println(event.MetricSetFields.StringToPrint())
	return true
}

func (m mockReporter) Error(err error) bool {
	return true
}

func TestData(t *testing.T) {
	mux := http.NewServeMux()

	mux.Handle("/_expvar", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		input, _ := ioutil.ReadFile("./_meta/testdata/expvar.282c.json")
		w.Write(input)
	}))

	server := httptest.NewServer(mux)
	defer server.Close()

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig([]string{"syncgateway"}, server.URL))
	if err := mbtest.WriteEventsReporterV2Error(f, t, ""); err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(metricsets []string, host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "couchbase",
		"metricsets": metricsets,
		"hosts":      []string{host},
		"extra": map[string]interface{}{
			"per_replication": true,
			"mem_stats":       true,
		},
	}
}
