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

package index

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

var info = elasticsearch.Info{
	ClusterID:   "1234",
	ClusterName: "helloworld",
}

var testedVersions = []string{"8.17.0", "7.17.27"}

func TestMapper(t *testing.T) {

	for _, version := range testedVersions {

		mux := createEsMuxer(version, "platinum", false)

		server := httptest.NewServer(mux)
		defer server.Close()

		httpClient, err := helper.NewHTTPFromConfig(helper.Config{
			ConnectTimeout: 30 * time.Second,
			Transport: httpcommon.HTTPTransportSettings{
				Timeout: 30 * time.Second,
			},
		}, mb.HostData{
			URI:          server.URL,
			SanitizedURI: server.URL,
			Host:         server.URL,
		})
		if err != nil {
			t.Fatal(err)
		}

		statsFileGlob := fmt.Sprintf("./_meta/test/stats*.%s.json", version)
		elasticsearch.TestMapperWithHttpHelper(t, statsFileGlob, httpClient, eventsMapping)
	}
}

func TestEmpty(t *testing.T) {
	httpClient, err := helper.NewHTTPFromConfig(helper.Config{}, mb.HostData{})
	if err != nil {
		t.Fatal(err)
	}

	input, err := ioutil.ReadFile("./_meta/test/empty.512.json")
	require.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	eventsMapping(reporter, httpClient, info, input, true)
	require.Equal(t, 0, len(reporter.GetEvents()))
}

func createEsMuxer(esVersion, license string, ccrEnabled bool) *http.ServeMux {
	nodesLocalHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"nodes": { "foobar": {}}}`))
	}
	clusterStateMasterHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"master_node": "foobar"}`))
	}
	rootHandler := func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "_stats") {
			input, _ := ioutil.ReadFile(fmt.Sprintf("./_meta/test/stats.%s.json", esVersion))
			w.Write(input)
			return
		} else if r.URL.Path != "/" {
			input, _ := ioutil.ReadFile(fmt.Sprintf("./_meta/test/settings.%s.json", esVersion))
			w.Write(input)
			return
		}

		input, _ := ioutil.ReadFile(fmt.Sprintf("./_meta/test/root.%s.json", esVersion))
		w.Write(input)

	}
	licenseHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{ "license": { "type": "` + license + `" } }`))
	}

	mux := http.NewServeMux()
	mux.Handle("/_nodes/_local/nodes", http.HandlerFunc(nodesLocalHandler))
	mux.Handle("/_cluster/state/master_node", http.HandlerFunc(clusterStateMasterHandler))
	mux.Handle("/_license", http.HandlerFunc(licenseHandler))       // for 7.0 and above
	mux.Handle("/_xpack/license", http.HandlerFunc(licenseHandler)) // for before 7.0

	mux.Handle("/_xpack/usage", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			input, _ := ioutil.ReadFile(fmt.Sprintf("./_meta/test/xpack-usage.%s.json", esVersion))
			w.Write(input)
		}))

	mux.Handle("/_cluster/state/routing_table", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			input, _ := ioutil.ReadFile(fmt.Sprintf("./_meta/test/cluster_state.%s.json", esVersion))
			w.Write(input)
		}))

	mux.Handle("/", http.HandlerFunc(rootHandler))

	return mux
}

func TestData(t *testing.T) {
	mux := createEsMuxer("8.17.0", "platinum", false)

	server := httptest.NewServer(mux)
	defer server.Close()

	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(server.URL))
	if err := mbtest.WriteEventsReporterV2Error(ms, t, ""); err != nil {
		t.Fatal("errors writing events to data.json file", err)
	}
}
func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     elasticsearch.ModuleName,
		"metricsets": []string{"index"},
		"hosts":      []string{host},
	}
}
