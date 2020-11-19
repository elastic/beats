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

package ccr

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func createEsMuxer(esVersion, license string, ccrEnabled bool) *http.ServeMux {
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
		w.Write([]byte(`{"name":"a14cf47ef7f2","cluster_name":"docker-cluster","cluster_uuid":"8l_zoGznQRmtoX9iSC-goA","version":{"number":"`+ esVersion + `","build_flavor":"default","build_type":"docker","build_hash":"43884496262f71aa3f33b34ac2f2271959dbf12a","build_date":"2020-10-28T09:54:14.068503Z","build_snapshot":true,"lucene_version":"8.7.0","minimum_wire_compatibility_version":"7.11.0","minimum_index_compatibility_version":"7.0.0"},"tagline":"You Know, for Search"}`))
	}
	licenseHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{ "license": { "type": "` + license + `" } }`))
	}
	xpackHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{ "features": { "ccr": { "enabled": ` + strconv.FormatBool(ccrEnabled) + `}}}`))
	}

	mux := http.NewServeMux()
	mux.Handle("/_nodes/_local/nodes", http.HandlerFunc(nodesLocalHandler))
	mux.Handle("/_cluster/state/master_node", http.HandlerFunc(clusterStateMasterHandler))
	mux.Handle("/", http.HandlerFunc(rootHandler))
	mux.Handle("/_license", http.HandlerFunc(licenseHandler))       // for 7.0 and above
	mux.Handle("/_xpack/license", http.HandlerFunc(licenseHandler)) // for before 7.0
	mux.Handle("/_xpack", http.HandlerFunc(xpackHandler))

	return mux
}

func TestCCRNotAvailable(t *testing.T) {
	tests := map[string]struct {
		esVersion  string
		license    string
		ccrEnabled bool
	}{
		"old_version": {
			"6.4.0",
			"platinum",
			true,
		},
		"low_license": {
			"7.6.0",
			"basic",
			true,
		},
		"feature_unavailable": {
			"7.6.0",
			"platinum",
			false,
		},
	}

	// Disable license caching for these tests
	elasticsearch.LicenseCacheEnabled = false
	defer func() { elasticsearch.LicenseCacheEnabled = true }()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			mux := createEsMuxer(test.esVersion, test.license, test.ccrEnabled)
			mux.Handle("/_ccr/stats", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "this should never have been called", 418)
			}))

			server := httptest.NewServer(mux)
			defer server.Close()

			ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(server.URL))
			events, errs := mbtest.ReportingFetchV2Error(ms)

			require.Empty(t, errs)
			require.Empty(t, events)
		})
	}
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     elasticsearch.ModuleName,
		"metricsets": []string{"ccr"},
		"hosts":      []string{host},
	}
}
