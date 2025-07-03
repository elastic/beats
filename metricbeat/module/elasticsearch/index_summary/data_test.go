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

package index_summary

import (
	"github.com/elastic/elastic-agent-libs/version"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

var info = elasticsearch.Info{
	ClusterID:   "1234",
	ClusterName: "helloworld",
}

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

		input, _ := os.ReadFile("../index/_meta/test/root.710.json")
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
	mux.Handle("/", http.HandlerFunc(rootHandler))
	mux.Handle("/_stats", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, _ := os.ReadFile("../index/_meta/test/stats.700-alpha1.json")
		w.Write(content)
	}))

	return mux
}

func TestGetServicePath(t *testing.T) {
	tests := []struct {
		name         string
		version      version.V
		expectedPath string
	}{
		{
			name:         "version < BulkStatsAvailableVersion",
			version:      version.V{Major: 7, Minor: 9, Bugfix: 0},
			expectedPath: "/_stats?level=cluster",
		},
		{
			name:         "version >= BulkStatsAvailableVersion",
			version:      version.V{Major: 8, Minor: 17, Bugfix: 0},
			expectedPath: "/_stats?level=cluster&forbid_closed_indices=false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := getServicePath(tt.version)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if path != tt.expectedPath {
				t.Errorf("expected path %q, got %q", tt.expectedPath, path)
			}
		})
	}
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

func TestMapper(t *testing.T) {
	elasticsearch.TestMapperWithInfo(t, "../index/_meta/test/stats.*.json", eventMapping)
}

func TestSummaryFromNodeStatsWithExpectedEventsV817(t *testing.T) {
	elasticsearch.TestMapperWithExpectedEvents(
		t,
		"_meta/test/node_stats_v817.json",
		[]string{
			"_meta/test/expected_event_8.17_v2.json",
		},
		elasticsearch.Info{
			ClusterID:   "1234",
			ClusterName: "helloworld",
		},
		true,
		eventMapping,
	)
}

func TestSummaryMissingField(t *testing.T) {
	elasticsearch.TestMapperExpectingError(
		t,
		"_meta/test/node_stats_v817_missing_fields.json",
		elasticsearch.Info{
			ClusterID:   "1234",
			ClusterName: "helloworld",
		},
		true,
		"error processing node \"Hwq8Kg1eRNaFnFJKrKoqjA\": key `indices.docs.count` not found",
		eventMapping,
	)
}

func TestSummaryMissingBlock(t *testing.T) {
	elasticsearch.TestMapperExpectingError(
		t,
		"_meta/test/node_stats_v817_missing_block.json",
		elasticsearch.Info{
			ClusterID:   "1234",
			ClusterName: "helloworld",
		},
		true,
		"error processing node \"Hwq8Kg1eRNaFnFJKrKoqjA\": key `indices.segments` not found",
		eventMapping,
	)
}

func TestSummaryWrongFieldType_String(t *testing.T) {
	elasticsearch.TestMapperExpectingError(
		t,
		"_meta/test/node_stats_v717_field_as_string.json",
		elasticsearch.Info{
			ClusterID:   "1234",
			ClusterName: "helloworld",
		},
		true,
		"error processing node \"vF3ak-83RKu_020pnVZJ_w\": wrong format in `indices.store.size_in_bytes`: expected integer, found string",
		eventMapping,
	)
}

func TestSummaryFromNodeStatsWithExpectedEventsV717(t *testing.T) {
	elasticsearch.TestMapperWithExpectedEvents(
		t,
		"_meta/test/node_stats_v717.json",
		[]string{
			"_meta/test/expected_event_7.17_v2.json",
		},
		elasticsearch.Info{
			ClusterID:   "1234",
			ClusterName: "helloworld",
		},
		true,
		eventMapping,
	)
}

func TestSummaryFromNodeStatsWithExpectedEventsXPackV817(t *testing.T) {
	elasticsearch.TestMapperWithExpectedEvents(
		t,
		"_meta/test/node_stats_v817.json",
		[]string{
			"_meta/test/expected_event_xpack_8.17_v2.json",
		},
		elasticsearch.Info{
			ClusterID:   "1234",
			ClusterName: "helloworld",
		},
		false,
		eventMapping,
	)
}

func TestSummaryFromNodeStatsWithExpectedEventsXPackV717(t *testing.T) {
	elasticsearch.TestMapperWithExpectedEvents(
		t,
		"_meta/test/node_stats_v717.json",
		[]string{
			"_meta/test/expected_event_xpack_7.17_v2.json",
		},
		elasticsearch.Info{
			ClusterID:   "1234",
			ClusterName: "helloworld",
		},
		false,
		eventMapping,
	)
}

func TestEmpty(t *testing.T) {
	input, err := os.ReadFile("../index/_meta/test/empty.512.json")
	require.NoError(t, err)

	reporter := &mbtest.CapturingReporterV2{}
	eventMapping(reporter, info, input, true)
	require.Empty(t, reporter.GetErrors())
	require.Equal(t, 1, len(reporter.GetEvents()))
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":                     elasticsearch.ModuleName,
		"metricsets":                 []string{"index_summary"},
		"hosts":                      []string{host},
		"index_recovery.active_only": false,
	}
}
