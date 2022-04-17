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

package shard

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/menderesk/beats/v7/metricbeat/module/elasticsearch"

	"github.com/stretchr/testify/require"

	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
)

func TestStats(t *testing.T) {
	files, err := filepath.Glob("./_meta/test/routing_table.*.json")
	require.NoError(t, err)

	for _, f := range files {
		input, err := ioutil.ReadFile(f)
		require.NoError(t, err)

		reporter := &mbtest.CapturingReporterV2{}
		eventsMapping(reporter, input, true)

		require.True(t, len(reporter.GetEvents()) >= 1)
		require.Equal(t, 0, len(reporter.GetErrors()))
	}
}

func TestData(t *testing.T) {
	mux := http.NewServeMux()

	mux.Handle("/_nodes/_local/nodes", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"nodes": { "foobar": {}}}`))
	}))
	mux.Handle("/_cluster/state/master_node", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"master_node": "foobar"}`))
	}))
	mux.Handle("/_cluster/state/version,nodes,master_node,routing_table", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			input, _ := ioutil.ReadFile("./_meta/test/routing_table.710.json")
			w.Write(input)
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
		"module":     elasticsearch.ModuleName,
		"metricsets": []string{"shard"},
		"hosts":      []string{host},
	}
}
