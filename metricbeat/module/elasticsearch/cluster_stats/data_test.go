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

package cluster_stats

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/metricbeat/helper"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
	"github.com/menderesk/beats/v7/metricbeat/module/elasticsearch"
)

func createEsMuxer(license string) *http.ServeMux {
	nodesLocalHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"nodes": { "foobar": {}}}`))
	}
	clusterStateMasterHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"master_node": "foobar"}`))
	}
	rootHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
		}

		input, _ := ioutil.ReadFile("./_meta/test/root.710.json")
		w.Write(input)
	}
	licenseHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{ "license": { "type": "` + license + `" } }`))
	}

	mux := http.NewServeMux()
	mux.Handle("/_nodes/_local/nodes", http.HandlerFunc(nodesLocalHandler))
	mux.Handle("/_cluster/state/master_node", http.HandlerFunc(clusterStateMasterHandler))
	mux.Handle("/", http.HandlerFunc(rootHandler))
	mux.Handle("/_license", http.HandlerFunc(licenseHandler))       // for 7.0 and above
	mux.Handle("/_xpack/license", http.HandlerFunc(licenseHandler)) // for before 7.0

	mux.Handle("/_xpack/usage", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			input, _ := ioutil.ReadFile("./_meta/test/xpack-usage.710.json")
			w.Write(input)
		}))

	mux.Handle("/_cluster/settings", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			input, _ := ioutil.ReadFile("./_meta/test/cluster-settings.710.json")
			w.Write(input)
		}))

	mux.Handle("/_cluster/stats", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			input, _ := ioutil.ReadFile("./_meta/test/cluster_stats.710.json")
			w.Write(input)
		}))

	mux.Handle("/_cluster/state/version,master_node,nodes,routing_table", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			input, _ := ioutil.ReadFile("./_meta/test/cluster_state.710.json")
			w.Write(input)
		}))

	return mux
}

func TestMapper(t *testing.T) {
	mux := createEsMuxer("platinum")

	server := httptest.NewServer(mux)
	defer server.Close()

	httpHelper, err := helper.NewHTTPFromConfig(helper.Config{}, mb.HostData{
		URI:          server.URL,
		SanitizedURI: server.URL,
		Host:         server.URL,
	})
	require.NoError(t, err)

	elasticsearch.TestMapperWithHttpHelper(t, "./_meta/test/cluster_stats.*.json",
		httpHelper, eventMapping)
}

func TestData(t *testing.T) {
	mux := createEsMuxer("platinum")

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
		"metricsets": []string{"cluster_stats"},
		"hosts":      []string{host},
	}
}
