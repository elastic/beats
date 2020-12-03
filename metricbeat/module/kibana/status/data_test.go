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

package status

import (
	"github.com/elastic/beats/v7/metricbeat/module/kibana"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestEventMapping(t *testing.T) {
	f := "./_meta/test/input.json"
	content, err := ioutil.ReadFile(f)
	require.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	err = eventMapping(reporter, content)
	require.NoError(t, err, f)
	require.True(t, len(reporter.GetEvents()) >= 1, f)
	require.Equal(t, 0, len(reporter.GetErrors()), f)
}


func TestData2(t *testing.T) {
	mux := http.NewServeMux()

	mux.Handle("/api/status", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, _ := ioutil.ReadFile("./_meta/testdata/7.0.0.json")
		w.Write(content)
	}))

	server := httptest.NewServer(mux)
	defer server.Close()

	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(server.URL))
	if err := mbtest.WriteEventsReporterV2Error(ms, t, ""); err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     kibana.ModuleName,
		"metricsets": []string{"status"},
		"hosts":      []string{host},
	}
}
