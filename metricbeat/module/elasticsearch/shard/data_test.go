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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestStats(t *testing.T) {
	files, err := filepath.Glob("./_meta/test/routing_table.*.json")
	require.NoError(t, err)

	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			input, err := os.ReadFile(f)
			require.NoError(t, err)

			reporter := &mbtest.CapturingReporterV2{}
			_ = eventsMapping(reporter, input, true, &lastClusterState{}, logp.NewLogger("test"))

			require.True(t, len(reporter.GetEvents()) >= 1)
			require.Equal(t, 0, len(reporter.GetErrors()))
		})
	}
}

func TestEventsMappingSkipsUnchangedClusterState(t *testing.T) {
	input, err := os.ReadFile("./_meta/test/routing_table.710.json")
	require.NoError(t, err)

	prev := &lastClusterState{}
	log := logp.NewLogger("test")

	first := &mbtest.CapturingReporterV2{}
	require.NoError(t, eventsMapping(first, input, true, prev, log))
	require.NotEmpty(t, first.GetEvents())
	require.True(t, prev.ok)
	require.Equal(t, "n-UoXaqYRoOe9qAC76IG6A", prev.id)

	second := &mbtest.CapturingReporterV2{}
	require.NoError(t, eventsMapping(second, input, true, prev, log))
	require.Empty(t, second.GetEvents())
	require.Equal(t, "n-UoXaqYRoOe9qAC76IG6A", prev.id)
}

func TestEventsMappingEmitsWhenClusterStateChanges(t *testing.T) {
	input, err := os.ReadFile("./_meta/test/routing_table.710.json")
	require.NoError(t, err)

	prev := &lastClusterState{}
	log := logp.NewLogger("test")

	first := &mbtest.CapturingReporterV2{}
	require.NoError(t, eventsMapping(first, input, true, prev, log))
	require.NotEmpty(t, first.GetEvents())
	require.Equal(t, "n-UoXaqYRoOe9qAC76IG6A", prev.id)

	changed := []byte(`{"cluster_name":"docker-cluster","cluster_uuid":"tMjf3CQ_TyCXNfcoR9eTWw","state_uuid":"changed-state-uuid","master_node":"hx-oJ1-aT_-5pRG22JMI1Q","nodes":{"hx-oJ1-aT_-5pRG22JMI1Q":{"name":"1fb2aa83efac"}},"routing_table":{"indices":{"test":{"shards":{"0":[{"state":"STARTED","primary":true,"node":"hx-oJ1-aT_-5pRG22JMI1Q","relocating_node":null,"shard":0,"index":"test"}]}}}}}`)
	second := &mbtest.CapturingReporterV2{}
	require.NoError(t, eventsMapping(second, changed, true, prev, log))
	require.NotEmpty(t, second.GetEvents())
	require.Equal(t, "changed-state-uuid", prev.id)
}

func TestEventsMappingEmitsFirstFetchWithEmptyStateUUID(t *testing.T) {
	// Fixtures without state_uuid must still emit on the first fetch; an empty
	// initial tracker must not be treated as "unchanged".
	input, err := os.ReadFile("./_meta/test/routing_table.623.json")
	require.NoError(t, err)

	prev := &lastClusterState{}
	reporter := &mbtest.CapturingReporterV2{}
	_ = eventsMapping(reporter, input, true, prev, logp.NewLogger("test"))
	require.NotEmpty(t, reporter.GetEvents())
	require.True(t, prev.ok)
	require.Equal(t, "", prev.id)
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
			input, _ := os.ReadFile("./_meta/test/routing_table.710.json")
			w.Write(input)
		}))

	server := httptest.NewServer(mux)
	defer server.Close()

	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(server.URL))
	if err := mbtest.WriteEventsReporterV2Error(ms, t, ""); err != nil {
		t.Fatal("write", err)
	}
}
func getConfig(host string) map[string]any {
	return map[string]any{
		"module":     elasticsearch.ModuleName,
		"metricsets": []string{"shard"},
		"hosts":      []string{host},
	}
}
