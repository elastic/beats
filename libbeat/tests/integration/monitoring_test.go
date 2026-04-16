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

//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// beatStateResponse represents the JSON returned by the beat's /state HTTP endpoint.
type beatStateResponse struct {
	Monitoring beatStateMonitoring `json:"monitoring"`
}

type beatStateMonitoring struct {
	ClusterUUID string `json:"cluster_uuid"`
}

// freePort finds and returns an available TCP port on localhost.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err, "could not find a free port")
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// getBeatState queries the beat's /state HTTP endpoint on the given port
// and returns the parsed response.
func getBeatState(t *testing.T, port int) beatStateResponse {
	t.Helper()
	url := fmt.Sprintf("http://localhost:%d/state", port)
	resp, err := http.Get(url) //nolint:noctx // fine for tests
	require.NoError(t, err, "could not GET beat /state endpoint")
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "could not read /state response body")
	var state beatStateResponse
	require.NoError(t, json.Unmarshal(body, &state), "could not unmarshal /state response")
	return state
}

// getMonitoringESURL returns the URL of the monitoring Elasticsearch cluster,
// read from ES_MONITORING_HOST / ES_MONITORING_PORT environment variables.
func getMonitoringESURL(t *testing.T) url.URL {
	t.Helper()
	host := os.Getenv("ES_MONITORING_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("ES_MONITORING_PORT")
	if port == "" {
		port = "9210"
	}
	return url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", host, port),
	}
}

// ensureMonitoringESIsRunning skips the test if the monitoring Elasticsearch
// cluster is not reachable.
func ensureMonitoringESIsRunning(t *testing.T) {
	t.Helper()
	monURL := getMonitoringESURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, monURL.String(), nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("monitoring Elasticsearch not reachable at %s: %s", monURL.String(), err)
	}
	resp.Body.Close()
}

// cleanMonitoringCluster deletes all .monitoring-beats-* indices from the
// monitoring Elasticsearch cluster.
func cleanMonitoringCluster(t *testing.T) {
	t.Helper()
	monURL := getMonitoringESURL(t)
	monURL.Path = "/.monitoring-beats-*"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, monURL.String(), nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	// 404 is acceptable: the index may not exist yet
}

// cleanOutputCluster removes transient xpack monitoring exporter settings and
// disables collection on the main Elasticsearch cluster.
func cleanOutputCluster(t *testing.T) {
	t.Helper()
	esURL := GetESAdminURL(t, "http")
	esURL.Path = "/_cluster/settings"

	payload := `{"transient":{"xpack.monitoring.exporters.*":null,"xpack.monitoring.collection.enabled":null}}`
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, esURL.String(), bytes.NewBufferString(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	u := esURL.User.Username()
	p, _ := esURL.User.Password()
	req.SetBasicAuth(u, p)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
}

// monitoringDocExists returns true if at least one document of the given
// monitoring type exists in the .monitoring-beats-* index.
func monitoringDocExists(t *testing.T, monitoringType string) bool {
	t.Helper()
	monURL := getMonitoringESURL(t)
	monURL.Path = "/.monitoring-beats-*/_search"
	monURL.RawQuery = url.Values{
		"q":    []string{"type:" + monitoringType},
		"size": []string{"1"},
	}.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, monURL.String(), nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}
	return result.Hits.Total.Value >= 1
}

// getMonitoringDoc retrieves a monitoring document of the given type and returns
// its _source as a map.
func getMonitoringDoc(t *testing.T, monitoringType string) map[string]interface{} {
	t.Helper()
	monURL := getMonitoringESURL(t)
	monURL.Path = "/.monitoring-beats-*/_search"
	monURL.RawQuery = url.Values{
		"q":    []string{"type:" + monitoringType},
		"size": []string{"1"},
	}.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, monURL.String(), nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var raw struct {
		Hits struct {
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	require.NoError(t, json.Unmarshal(body, &raw))
	require.NotEmpty(t, raw.Hits.Hits, "no monitoring document of type %q found", monitoringType)
	return raw.Hits.Hits[0].Source
}

// TestMonitoringDirectToCluster tests shipping monitoring data directly to a
// dedicated monitoring Elasticsearch cluster and verifies that beats_stats and
// beats_state documents are indexed with the expected top-level fields.
func TestMonitoringDirectToCluster(t *testing.T) {
	EnsureESIsRunning(t)
	ensureMonitoringESIsRunning(t)

	monURL := getMonitoringESURL(t)
	cfg := fmt.Sprintf(`
mockbeat:
output:
  console:
    enabled: true
monitoring:
  elasticsearch:
    hosts: ["%s"]
logging:
  level: debug
`, monURL.String())

	cleanOutputCluster(t)
	cleanMonitoringCluster(t)

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()

	mockbeat.WaitLogsContains("mockbeat start running.", 60*time.Second)
	mockbeat.WaitLogsContains("[monitoring]", 60*time.Second)

	require.Eventually(t, func() bool {
		return monitoringDocExists(t, "beats_stats")
	}, 60*time.Second, time.Second, "beats_stats document never appeared in monitoring cluster")

	require.Eventually(t, func() bool {
		return monitoringDocExists(t, "beats_state")
	}, 60*time.Second, time.Second, "beats_state document never appeared in monitoring cluster")

	mockbeat.Stop()

	for _, monType := range []string{"beats_stats", "beats_state"} {
		doc := getMonitoringDoc(t, monType)
		for _, field := range []string{"cluster_uuid", "timestamp", "interval_ms", "type", monType} {
			require.Contains(t, doc, field, "monitoring document of type %q missing field %q", monType, field)
		}
	}
}

// TestMonitoringClusterUUID verifies that the monitoring.cluster_uuid setting
// can be configured without any other monitoring.* settings and is reflected
// in the beat's /state HTTP endpoint.
func TestMonitoringClusterUUID(t *testing.T) {
	testClusterUUID := "test-cluster-uuid-abcde12345"
	port := freePort(t)
	cfg := fmt.Sprintf(`
mockbeat:
output:
  console:
    enabled: true
monitoring.cluster_uuid: %s
http.enabled: true
http.port: %d
`, testClusterUUID, port)

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitLogsContains("Starting stats endpoint", 60*time.Second)
	time.Sleep(time.Second)

	state := getBeatState(t, port)
	require.Equal(t, testClusterUUID, state.Monitoring.ClusterUUID,
		"monitoring.cluster_uuid should appear in /state endpoint")
}

// TestMonitoringClusterUUIDMonitoringDisabled verifies that monitoring.cluster_uuid
// can be set alongside monitoring.enabled=false and still appears in the beat's
// /state HTTP endpoint.
func TestMonitoringClusterUUIDMonitoringDisabled(t *testing.T) {
	testClusterUUID := "test-cluster-uuid-fghij67890"
	port := freePort(t)
	cfg := fmt.Sprintf(`
mockbeat:
output:
  console:
    enabled: true
monitoring.cluster_uuid: %s
monitoring.enabled: false
http.enabled: true
http.port: %d
`, testClusterUUID, port)

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitLogsContains("Starting stats endpoint", 60*time.Second)
	time.Sleep(time.Second)

	state := getBeatState(t, port)
	require.Equal(t, testClusterUUID, state.Monitoring.ClusterUUID,
		"monitoring.cluster_uuid should appear in /state endpoint even when monitoring.enabled=false")
}
