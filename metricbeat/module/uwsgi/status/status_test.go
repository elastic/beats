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

package status

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func testData(t *testing.T) (data []byte) {
	absPath, err := filepath.Abs(filepath.Join("..", "_meta", "testdata"))
	if err != nil {
		t.Fatalf("filepath failed: %s", err.Error())
		return
	}

	data, err = ioutil.ReadFile(filepath.Join(absPath, "/data.json"))
	if err != nil {
		t.Fatalf("ReadFile failed: %s", err.Error())
		return
	}
	return
}

func findItems(mp []mb.Event, key string) []mapstr.M {
	result := make([]mapstr.M, 0, 1)
	for _, v := range mp {
		if el, ok := v.MetricSetFields[key]; ok {
			result = append(result, el.(mapstr.M))
		}
	}

	return result
}

func assertTestData(t *testing.T, evt []mb.Event) {
	totals := findItems(evt, "total")
	assert.Equal(t, 1, len(totals))
	assert.Equal(t, 2042, totals[0]["requests"])
	assert.Equal(t, 0, totals[0]["exceptions"])
	assert.Equal(t, 34, totals[0]["write_errors"])
	assert.Equal(t, 38, totals[0]["read_errors"])

	workers := findItems(evt, "core")
	assert.Equal(t, 4, len(workers))
}

func TestFetchDataTCP(t *testing.T) {

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		conn, err := listener.Accept()
		assert.NoError(t, err)

		data := testData(t)
		conn.Write(data)
		conn.Close()
		wg.Done()
	}()

	config := map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
		"hosts":      []string{"tcp://" + listener.Addr().String()},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}

	assertTestData(t, events)
	wg.Wait()
}

func TestFetchDataHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := testData(t)

		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write(data)
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}

	assertTestData(t, events)
}

func TestFetchDataUnmarshalledError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte("fail json.Unmarshal"))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)
	_, errs := mbtest.ReportingFetchV2Error(f)
	assert.NotEmpty(t, errs)
}

func TestFetchDataSourceDown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	server.Close()

	config := map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)
	_, errs := mbtest.ReportingFetchV2Error(f)
	assert.NotEmpty(t, errs)
}

func TestConfigError(t *testing.T) {
	config := map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
		"hosts":      []string{"unix://127.0.0.1:8080"},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)
	_, errs := mbtest.ReportingFetchV2Error(f)
	assert.NotEmpty(t, errs)

	config = map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
		"hosts":      []string{"unknown_url_format"},
	}

	f = mbtest.NewReportingMetricSetV2Error(t, config)
	_, errs = mbtest.ReportingFetchV2Error(f)
	assert.NotEmpty(t, errs)

	config = map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
		"hosts":      []string{"ftp://127.0.0.1:8080"},
	}

	f = mbtest.NewReportingMetricSetV2Error(t, config)
	_, errs = mbtest.ReportingFetchV2Error(f)
	assert.NotEmpty(t, errs)
}
