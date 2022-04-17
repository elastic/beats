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

package server

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/menderesk/beats/v7/libbeat/common"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

const testFile = "../_meta/test/serverstats"

func TestEventMapping(t *testing.T) {
	content, err := ioutil.ReadFile(testFile)
	assert.NoError(t, err)

	event, err := eventMapping(content)
	assert.NoError(t, err, "error mapping "+testFile)

	assert.Len(t, event, 7, "got wrong number of event")

	clusterManager := event["cluster_manager"].(common.MapStr)
	assert.Equal(t, int64(1), clusterManager["active_clusters"])
	assert.Equal(t, int64(1), clusterManager["cluster_added"])
	assert.Equal(t, int64(0), clusterManager["cluster_modified"])
	assert.Equal(t, int64(0), clusterManager["cluster_removed"])
	assert.Equal(t, int64(0), clusterManager["warming_clusters"])
	assert.Equal(t, int64(0), clusterManager["cluster_updated"])
	assert.Equal(t, int64(0), clusterManager["cluster_updated_via_merge"])
	assert.Equal(t, int64(0), clusterManager["update_merge_cancelled"])
	assert.Equal(t, int64(0), clusterManager["update_out_of_merge_window"])

	fileSystem := event["filesystem"].(common.MapStr)
	assert.Equal(t, int64(389), fileSystem["flushed_by_timer"])
	assert.Equal(t, int64(0), fileSystem["reopen_failed"])
	assert.Equal(t, int64(44), fileSystem["write_buffered"])
	assert.Equal(t, int64(43), fileSystem["write_completed"])
	assert.Equal(t, int64(0), fileSystem["write_total_buffered"])
	assert.Equal(t, int64(0), fileSystem["write_total_buffered"])
	assert.Equal(t, int64(0), fileSystem["write_failed"])

	listenerManager := event["listener_manager"].(common.MapStr)
	assert.Equal(t, int64(1), listenerManager["listener_added"])
	assert.Equal(t, int64(0), listenerManager["listener_create_failure"])
	assert.Equal(t, int64(4), listenerManager["listener_create_success"])
	assert.Equal(t, int64(0), listenerManager["listener_modified"])
	assert.Equal(t, int64(0), listenerManager["listener_removed"])
	assert.Equal(t, int64(1), listenerManager["total_listeners_active"])
	assert.Equal(t, int64(0), listenerManager["total_listeners_draining"])
	assert.Equal(t, int64(0), listenerManager["total_listeners_warming"])
	assert.Equal(t, int64(0), listenerManager["listener_stopped"])

	runtime := event["runtime"].(common.MapStr)
	assert.Equal(t, int64(0), runtime["admin_overrides_active"])
	assert.Equal(t, int64(0), runtime["load_error"])
	assert.Equal(t, int64(0), runtime["load_success"])
	assert.Equal(t, int64(0), runtime["num_keys"])
	assert.Equal(t, int64(0), runtime["override_dir_exists"])
	assert.Equal(t, int64(0), runtime["override_dir_not_exists"])
	assert.Equal(t, int64(0), runtime["deprecated_feature_use"])
	assert.Equal(t, int64(2), runtime["num_layers"])

	server := event["server"].(common.MapStr)
	assert.Equal(t, int64(2147483647), server["days_until_first_cert_expiring"])
	assert.Equal(t, int64(1), server["live"])
	assert.Equal(t, int64(3120760), server["memory_allocated"])
	assert.Equal(t, int64(4194304), server["memory_heap_size"])
	assert.Equal(t, int64(0), server["parent_connections"])
	assert.Equal(t, int64(0), server["total_connections"])
	assert.Equal(t, int64(5025), server["uptime"])
	assert.Equal(t, int64(16364036), server["version"])
	assert.Equal(t, int64(4), server["watchdog_mega_miss"])
	assert.Equal(t, int64(4), server["watchdog_miss"])
	assert.Equal(t, int64(12), server["concurrency"])
	assert.Equal(t, int64(0), server["debug_assertion_failures"])
	assert.Equal(t, int64(0), server["dynamic_unknown_fields"])
	assert.Equal(t, int64(0), server["hot_restart_epoch"])
	assert.Equal(t, int64(0), server["state"])
	assert.Equal(t, int64(0), server["static_unknown_fields"])
	assert.Equal(t, int64(0), server["stats_recent_lookups"])

	stats := event["stats"].(common.MapStr)
	assert.Equal(t, int64(0), stats["overflow"])

	http2 := event["http2"].(common.MapStr)
	assert.Equal(t, int64(0), http2["header_overflow"])
	assert.Equal(t, int64(0), http2["headers_cb_no_stream"])
	assert.Equal(t, int64(0), http2["rx_reset"])
	assert.Equal(t, int64(0), http2["too_many_header_frames"])
	assert.Equal(t, int64(0), http2["trailers"])
	assert.Equal(t, int64(0), http2["tx_reset"])
}

func TestFetchEventContent(t *testing.T) {
	absPath, err := filepath.Abs("../_meta/test/")
	assert.NoError(t, err)

	response, err := ioutil.ReadFile(absPath + "/serverstats")
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "envoyproxy",
		"metricsets": []string{"server"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), events[0])
}

func testValue(t *testing.T, event common.MapStr, field string, value interface{}) {
	data, err := event.GetValue(field)
	assert.NoError(t, err, "Could not read field "+field)
	assert.EqualValues(t, data, value, "Wrong value for field "+field)
}

func TestFetchTimeout(t *testing.T) {
	absPath, err := filepath.Abs("../_meta/test/")
	assert.NoError(t, err)

	response, err := ioutil.ReadFile(absPath + "/serverstats")
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		w.Write([]byte(response))
		<-r.Context().Done()
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "envoyproxy",
		"metricsets": []string{"server"},
		"hosts":      []string{server.URL},
		"timeout":    "50ms",
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)

	start := time.Now()
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) == 0 {
		t.Fatalf("Expected an error, had %d. %v\n", len(errs), errs)
	}
	assert.Empty(t, events)
	elapsed := time.Since(start)
	var found bool
	for _, err := range errs {
		if strings.Contains(err.Error(), "Client.Timeout exceeded") {
			found = true
		}
	}
	if !found {
		assert.Failf(t, "", "expected an error containing '(Client.Timeout exceeded'. Got %v", errs)
	}

	assert.True(t, elapsed < 5*time.Second, "elapsed time: %s", elapsed.String())
}
